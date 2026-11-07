package share

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"kosh/internal/auth"
	"kosh/internal/models"
	"kosh/internal/report"
	"kosh/internal/store"
	"kosh/internal/tenant"
)

const (
	cookieName = "kosh_share"
	// cookiePath scopes the share session to recipient endpoints only — the
	// browser never sends it to /api/data or any owner route.
	cookiePath = "/api/share/"
	// sessionTTL is how long a verified device stays unlocked before the
	// recipient is asked for a fresh code.
	sessionTTL  = 24 * time.Hour
	maxAttempts = 5
	lockout     = 15 * time.Minute
	issuer      = "Kosh"

	// TokenHeader carries the share-link token on recipient API calls. It's
	// never put in a URL query, so it can't end up in request logs; links
	// themselves keep it in the URL fragment (/s#<token>), which browsers
	// never send to the server.
	TokenHeader = "X-Kosh-Share-Token"
)

// TokenIndex records which tenant a share-link token belongs to, so a
// hosted deploy can route an anonymous recipient request to the right
// household. Self-hosted Kosh has one household and needs none.
type TokenIndex interface {
	PutShareToken(ctx context.Context, tokenHash, tenantID string) error
	DeleteShareToken(ctx context.Context, tokenHash string) error
}

// ErrGeneric is shown for every "no" that isn't a lockout, so a response
// never reveals whether a link exists, was revoked, or has expired.
const ErrGeneric = "This link or code isn't valid. Check the code in your authenticator app, or ask for a new link."

// Handler serves both the owner's share-management API (behind normal login)
// and the recipients' unauthenticated verify/report API.
type Handler struct {
	tenants      tenant.Resolver
	sealer       *Sealer
	session      *auth.Session
	familyData   func(*tenant.Tenant) (*models.Data, error)
	isDemo       func(*http.Request) bool
	cookieSecure bool
	now          func() time.Time

	// mu serialises verify attempts within this instance so two concurrent
	// guesses can't both read the same failed-attempt count. A pointer, so
	// the per-request copies made by scoped all share it.
	mu *sync.Mutex

	index TokenIndex // nil when self-hosted

	// Set only on the per-request copy returned by scoped.
	t      *tenant.Tenant
	shares *store.Repository[models.ShareRecipient]
	access *store.Repository[models.ShareAccess]
}

// NewHandler constructs a share Handler. sessionSecret is the app's
// SESSION_SECRET; share sessions are signed with a key derived from it, so a
// share cookie can never be accepted as an owner session (or vice versa).
func NewHandler(tenants tenant.Resolver, sealer *Sealer, sessionSecret string, familyData func(*tenant.Tenant) (*models.Data, error), isDemo func(*http.Request) bool, cookieSecure bool) *Handler {
	return &Handler{
		tenants:      tenants,
		sealer:       sealer,
		session:      auth.NewSession("kosh-share-session:" + sessionSecret),
		familyData:   familyData,
		isDemo:       isDemo,
		cookieSecure: cookieSecure,
		now:          time.Now,
		mu:           &sync.Mutex{},
	}
}

// SetTokenIndex enables share-token indexing (hosted mode).
func (h *Handler) SetTokenIndex(idx TokenIndex) { h.index = idx }

// indexPut / indexDelete are no-ops when self-hosted.
func (h *Handler) indexPut(ctx context.Context, tokenHash string) error {
	if h.index == nil {
		return nil
	}
	return h.index.PutShareToken(ctx, tokenHash, h.t.ID)
}

func (h *Handler) indexDelete(ctx context.Context, tokenHash string) {
	if h.index == nil {
		return
	}
	if err := h.index.DeleteShareToken(ctx, tokenHash); err != nil {
		// A stale index entry only points at a row that no longer matches
		// any token hash, so it grants nothing; log and move on.
		log.Printf("⚠ removing share token from index: %v", err)
	}
}

// scoped returns a copy of h bound to the request's tenant, so the helpers
// below can use h.shares / h.access without threading the tenant through.
// In hosted mode the resolver finds the tenant from the session (owner
// routes) or the link token (recipient routes).
func (h *Handler) scoped(w http.ResponseWriter, r *http.Request) (*Handler, bool) {
	t, ok := tenant.FromRequest(h.tenants, w, r)
	if !ok {
		return nil, false
	}
	cp := *h
	cp.t, cp.shares, cp.access = t, t.Repos.Shares, t.Repos.ShareAccess
	return &cp, true
}

// ── Owner API ────────────────────────────────────────────────────────────────

// shareView is what the owner sees for a recipient — never the token hash or
// secret.
type shareView struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Relation   string `json:"relation"`
	Expires    string `json:"expires"`
	Created    string `json:"created"`
	Status     string `json:"status"` // active | revoked | expired | locked
	LastOpened string `json:"last_opened"`
}

// enrolment is returned once, when a recipient is created or reset — the
// only time the link token and TOTP secret ever leave the server.
type enrolment struct {
	Share      shareView `json:"share"`
	LinkPath   string    `json:"link_path"`
	OTPAuthURI string    `json:"otpauth_uri"`
	Secret     string    `json:"secret"`
}

// Owner routes /api/shares[/{id}[/revoke|/reset|/access]].
func (h *Handler) Owner(w http.ResponseWriter, r *http.Request) {
	if h.isDemo != nil && h.isDemo(r) {
		http.Error(w, "Family sharing isn't available in the demo.", http.StatusForbidden)
		return
	}
	h, ok := h.scoped(w, r)
	if !ok {
		return
	}
	parts := strings.Split(strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/shares"), "/"), "/")
	switch {
	case parts[0] == "" && r.Method == http.MethodGet:
		h.list(w)
	case parts[0] == "" && r.Method == http.MethodPost:
		h.create(w, r)
	case len(parts) == 1 && r.Method == http.MethodDelete:
		h.remove(w, r, parts[0])
	case len(parts) == 2 && parts[1] == "revoke" && r.Method == http.MethodPost:
		h.revoke(w, parts[0])
	case len(parts) == 2 && parts[1] == "reset" && r.Method == http.MethodPost:
		h.reset(w, r, parts[0])
	case len(parts) == 2 && parts[1] == "access" && r.Method == http.MethodGet:
		h.accessLog(w, parts[0])
	default:
		http.Error(w, "not found", http.StatusNotFound)
	}
}

func (h *Handler) list(w http.ResponseWriter) {
	rows, err := h.shares.All()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	lastOpened := map[string]string{}
	if logs, err := h.access.All(); err == nil {
		for _, l := range logs {
			if (l.Event == "verified" || l.Event == "viewed") && l.At > lastOpened[l.ShareID] {
				lastOpened[l.ShareID] = l.At
			}
		}
	}
	out := make([]shareView, 0, len(rows))
	for _, s := range rows {
		v := h.view(s)
		v.LastOpened = lastOpened[s.ID]
		out = append(out, v)
	}
	writeJSON(w, out)
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name     string `json:"name"`
		Relation string `json:"relation"`
		Expires  string `json:"expires"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	body.Name = strings.TrimSpace(body.Name)
	if body.Name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}
	if body.Expires != "" {
		if _, err := time.Parse("2006-01-02", body.Expires); err != nil {
			http.Error(w, "expires must be YYYY-MM-DD", http.StatusBadRequest)
			return
		}
	}
	rec := models.ShareRecipient{
		ID:       randomID("sh_"),
		Name:     body.Name,
		Relation: strings.TrimSpace(body.Relation),
		Expires:  body.Expires,
		Created:  h.now().UTC().Format(time.RFC3339),
	}
	enr, err := h.issueCredentials(&rec)
	if err != nil {
		http.Error(w, "generating credentials: "+err.Error(), http.StatusInternalServerError)
		return
	}
	// Index first: a share row whose token isn't indexed would be a link
	// that silently never works in hosted mode.
	if err := h.indexPut(r.Context(), rec.TokenHash); err != nil {
		http.Error(w, "indexing share link: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if _, err := h.shares.Add(rec); err != nil {
		h.indexDelete(r.Context(), rec.TokenHash)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	enr.Share = h.view(rec)
	writeJSON(w, enr)
}

// reset issues a brand-new link and TOTP secret (lost phone, leaked link),
// killing the old ones, and lifts any revocation/lockout.
func (h *Handler) reset(w http.ResponseWriter, r *http.Request, id string) {
	rec, ok := h.byID(w, id)
	if !ok {
		return
	}
	oldHash := rec.TokenHash
	enr, err := h.issueCredentials(&rec)
	if err != nil {
		http.Error(w, "generating credentials: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if err := h.indexPut(r.Context(), rec.TokenHash); err != nil {
		http.Error(w, "indexing share link: "+err.Error(), http.StatusInternalServerError)
		return
	}
	err = h.shares.Update(id, map[string]any{
		"token_hash": rec.TokenHash, "totp_secret_enc": rec.SecretEnc,
		"revoked": "", "failed_attempts": 0, "locked_until": "", "last_totp_step": 0,
	})
	if err != nil {
		h.indexDelete(r.Context(), rec.TokenHash)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	h.indexDelete(r.Context(), oldHash)
	rec.Revoked, rec.FailedAttempts, rec.LockedUntil = "", 0, ""
	enr.Share = h.view(rec)
	writeJSON(w, enr)
}

func (h *Handler) revoke(w http.ResponseWriter, id string) {
	if _, ok := h.byID(w, id); !ok {
		return
	}
	if err := h.shares.Update(id, map[string]any{"revoked": "true"}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]string{"status": "ok"})
}

func (h *Handler) remove(w http.ResponseWriter, r *http.Request, id string) {
	rec, ok := h.byID(w, id)
	if !ok {
		return
	}
	if err := h.shares.Delete(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	h.indexDelete(r.Context(), rec.TokenHash)
	writeJSON(w, map[string]string{"status": "ok"})
}

func (h *Handler) accessLog(w http.ResponseWriter, id string) {
	logs, err := h.access.All()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	out := []models.ShareAccess{}
	for _, l := range logs {
		if l.ShareID == id {
			out = append(out, l)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].At > out[j].At })
	if len(out) > 50 {
		out = out[:50]
	}
	writeJSON(w, out)
}

func (h *Handler) issueCredentials(rec *models.ShareRecipient) (enrolment, error) {
	token, err := NewLinkToken()
	if err != nil {
		return enrolment{}, err
	}
	secret, err := NewTOTPSecret()
	if err != nil {
		return enrolment{}, err
	}
	enc, err := h.sealer.Seal(secret)
	if err != nil {
		return enrolment{}, err
	}
	rec.TokenHash = HashToken(token)
	rec.SecretEnc = enc
	return enrolment{
		LinkPath:   "/s#" + token,
		OTPAuthURI: ProvisioningURI(issuer, rec.Name, secret),
		Secret:     secret,
	}, nil
}

func (h *Handler) byID(w http.ResponseWriter, id string) (models.ShareRecipient, bool) {
	rows, err := h.shares.All()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return models.ShareRecipient{}, false
	}
	for _, s := range rows {
		if s.ID == id {
			return s, true
		}
	}
	http.Error(w, "share not found", http.StatusNotFound)
	return models.ShareRecipient{}, false
}

func (h *Handler) view(s models.ShareRecipient) shareView {
	v := shareView{ID: s.ID, Name: s.Name, Relation: s.Relation, Expires: s.Expires, Created: s.Created, Status: "active"}
	switch {
	case s.Revoked == "true":
		v.Status = "revoked"
	case h.expired(s):
		v.Status = "expired"
	case h.locked(s):
		v.Status = "locked"
	}
	return v
}

// ── Recipient API ────────────────────────────────────────────────────────────

// Recipient routes /api/share/{verify|report|export|logout}. Unauthenticated:
// access is granted by link token + TOTP code, then by the share cookie.
func (h *Handler) Recipient(w http.ResponseWriter, r *http.Request) {
	h, ok := h.scoped(w, r)
	if !ok {
		return
	}
	switch strings.TrimPrefix(r.URL.Path, "/api/share/") {
	case "verify":
		h.verify(w, r)
	case "report":
		h.report(w, r)
	case "export":
		h.export(w, r)
	case "logout":
		http.SetCookie(w, &http.Cookie{Name: cookieName, Value: "", Path: cookiePath, Secure: h.cookieSecure, HttpOnly: true, MaxAge: -1})
		writeJSON(w, map[string]string{"status": "ok"})
	default:
		http.Error(w, "not found", http.StatusNotFound)
	}
}

func (h *Handler) verify(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		Code string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, ErrGeneric, http.StatusUnauthorized)
		return
	}
	code := strings.ReplaceAll(strings.TrimSpace(body.Code), " ", "")

	h.mu.Lock()
	defer h.mu.Unlock()

	rec, found := h.byToken(r.Header.Get(TokenHeader))
	if !found {
		http.Error(w, ErrGeneric, http.StatusUnauthorized)
		return
	}
	if rec.Revoked == "true" || h.expired(rec) {
		h.logAccess(r, rec.ID, "denied")
		http.Error(w, ErrGeneric, http.StatusUnauthorized)
		return
	}
	if h.locked(rec) {
		h.logAccess(r, rec.ID, "locked")
		http.Error(w, lockedMessage(rec, h.now()), http.StatusTooManyRequests)
		return
	}

	secret, err := h.sealer.Open(rec.SecretEnc)
	if err != nil {
		log.Printf("⚠ share %s: decrypting TOTP secret: %v", rec.ID, err)
		http.Error(w, ErrGeneric, http.StatusUnauthorized)
		return
	}
	step, ok := VerifyTOTP(secret, code, h.now(), int64(rec.LastTOTPStep))
	if !ok {
		failed := rec.FailedAttempts + 1
		patch := map[string]any{"failed_attempts": failed}
		status, msg := http.StatusUnauthorized, ErrGeneric
		if failed >= maxAttempts {
			rec.LockedUntil = h.now().Add(lockout).UTC().Format(time.RFC3339)
			patch = map[string]any{"failed_attempts": 0, "locked_until": rec.LockedUntil}
			status, msg = http.StatusTooManyRequests, lockedMessage(rec, h.now())
		}
		if err := h.shares.Update(rec.ID, patch); err != nil {
			log.Printf("⚠ share %s: recording failed attempt: %v", rec.ID, err)
		}
		h.logAccess(r, rec.ID, "bad_code")
		http.Error(w, msg, status)
		return
	}

	// Persisting the step is what makes replay protection hold across
	// instances — fail closed if it can't be written.
	if err := h.shares.Update(rec.ID, map[string]any{"failed_attempts": 0, "locked_until": "", "last_totp_step": step}); err != nil {
		log.Printf("⚠ share %s: recording successful verify: %v", rec.ID, err)
		http.Error(w, "Couldn't verify right now — please try again.", http.StatusServiceUnavailable)
		return
	}
	h.logAccess(r, rec.ID, "verified")
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    h.session.SignFor(rec.ID, sessionTTL),
		Path:     cookiePath,
		HttpOnly: true,
		Secure:   h.cookieSecure,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   int(sessionTTL.Seconds()),
	})
	writeJSON(w, map[string]string{"name": rec.Name})
}

// sharedReport is the read-only family report sent to a recipient — built
// server-side from report.BuildTables, so the raw /api/data payload (and
// anything not in a report section) never reaches them.
type sharedReport struct {
	Recipient string         `json:"recipient"`
	Scope     string         `json:"scope"`
	Generated string         `json:"generated"`
	Tables    []report.Table `json:"tables"`
}

func (h *Handler) report(w http.ResponseWriter, r *http.Request) {
	rec, ok := h.authorize(w, r)
	if !ok {
		return
	}
	data, err := h.familyData(h.t)
	if err != nil {
		log.Printf("⚠ share %s: loading family data: %v", rec.ID, err)
		http.Error(w, "This report is temporarily unavailable. Please let the person who shared it know.", http.StatusServiceUnavailable)
		return
	}
	meta := report.BuildMeta(data, "", h.now())
	h.logAccess(r, rec.ID, "viewed")
	writeJSON(w, sharedReport{
		Recipient: rec.Name,
		Scope:     meta.Scope,
		Generated: meta.Generated,
		Tables:    report.BuildTables(data, "", report.ValidSections()),
	})
}

func (h *Handler) export(w http.ResponseWriter, r *http.Request) {
	rec, ok := h.authorize(w, r)
	if !ok {
		return
	}
	data, err := h.familyData(h.t)
	if err != nil {
		log.Printf("⚠ share %s: loading family data: %v", rec.ID, err)
		http.Error(w, "This report is temporarily unavailable. Please let the person who shared it know.", http.StatusServiceUnavailable)
		return
	}
	meta := report.BuildMeta(data, "", h.now())
	// Stamp the recipient's name on the PDF so a forwarded copy is traceable.
	meta.Scope += " · shared with " + rec.Name
	pdf, err := report.RenderPDF(report.BuildTables(data, "", report.ValidSections()), meta)
	if err != nil {
		http.Error(w, "generating report: "+err.Error(), http.StatusInternalServerError)
		return
	}
	h.logAccess(r, rec.ID, "pdf")
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="Kosh-Family-Report-%s.pdf"`, h.now().Format("2006-01-02")))
	w.Write(pdf)
}

// authorize checks that the request's share cookie belongs to the recipient
// whose link token is in the TokenHeader, and that the share is still active.
func (h *Handler) authorize(w http.ResponseWriter, r *http.Request) (models.ShareRecipient, bool) {
	deny := func() (models.ShareRecipient, bool) {
		http.Error(w, "verification required", http.StatusUnauthorized)
		return models.ShareRecipient{}, false
	}
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return models.ShareRecipient{}, false
	}
	c, err := r.Cookie(cookieName)
	if err != nil {
		return deny()
	}
	shareID, err := h.session.Verify(c.Value)
	if err != nil {
		return deny()
	}
	rec, found := h.byToken(r.Header.Get(TokenHeader))
	if !found || rec.ID != shareID || rec.Revoked == "true" || h.expired(rec) {
		return deny()
	}
	return rec, true
}

func (h *Handler) byToken(token string) (models.ShareRecipient, bool) {
	if token == "" {
		return models.ShareRecipient{}, false
	}
	rows, err := h.shares.All()
	if err != nil {
		log.Printf("⚠ reading Shares: %v", err)
		return models.ShareRecipient{}, false
	}
	want := HashToken(token)
	for _, s := range rows {
		if s.TokenHash == want {
			return s, true
		}
	}
	return models.ShareRecipient{}, false
}

func (h *Handler) expired(s models.ShareRecipient) bool {
	if s.Expires == "" {
		return false
	}
	exp, err := time.Parse("2006-01-02", s.Expires)
	if err != nil {
		return true // fail closed on a hand-edited, unparseable date
	}
	// The expiry date itself is still a valid day.
	return !h.now().Before(exp.AddDate(0, 0, 1))
}

func (h *Handler) locked(s models.ShareRecipient) bool {
	if s.LockedUntil == "" {
		return false
	}
	until, err := time.Parse(time.RFC3339, s.LockedUntil)
	return err == nil && h.now().Before(until)
}

func lockedMessage(s models.ShareRecipient, now time.Time) string {
	until, _ := time.Parse(time.RFC3339, s.LockedUntil)
	mins := int(until.Sub(now).Minutes()) + 1
	return fmt.Sprintf("Too many wrong codes. Please wait %d minute(s) and try again.", mins)
}

func (h *Handler) logAccess(r *http.Request, shareID, event string) {
	ua := r.UserAgent()
	if len(ua) > 160 {
		ua = ua[:160]
	}
	entry := models.ShareAccess{
		ID:        randomID("sa_"),
		ShareID:   shareID,
		At:        h.now().UTC().Format(time.RFC3339),
		Event:     event,
		IP:        clientIP(r),
		UserAgent: ua,
	}
	if _, err := h.access.Add(entry); err != nil {
		log.Printf("⚠ writing share access log: %v", err)
	}
}

// clientIP prefers the first X-Forwarded-For hop (Cloud Run / Netlify sit in
// front of the app); it's for the owner's information only, never for
// access decisions, so spoofability doesn't matter.
func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return strings.TrimSpace(strings.Split(xff, ",")[0])
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}
