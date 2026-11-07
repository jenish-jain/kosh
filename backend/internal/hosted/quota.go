package hosted

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"kosh/internal/controlplane"
)

// MonthlyQuotas are the AI uses each household gets per calendar month
// (UTC). AI calls cost real money per request; everything else is
// unmetered.
var MonthlyQuotas = map[string]int{"ask": 200, "tax": 20, "parse": 30}

var quotaNoun = map[string]string{"ask": "questions to Ask Kosh", "tax": "tax recommendation runs", "parse": "document scans"}

// quotaKind maps a request to the metered AI feature it uses, if any.
func quotaKind(r *http.Request) string {
	if r.Method != http.MethodPost {
		return ""
	}
	switch p := r.URL.Path; {
	case p == "/api/chat":
		return "ask"
	case p == "/api/tax/recommendations/generate":
		return "tax"
	case strings.HasPrefix(p, "/api/upload/"):
		return "parse"
	}
	return ""
}

func period(now time.Time) string { return now.UTC().Format("2006-01") }

func nextReset(now time.Time) time.Time {
	y, m, _ := now.UTC().Date()
	return time.Date(y, m+1, 1, 0, 0, 0, 0, time.UTC)
}

// Quotas meters AI features per household. A use is consumed before the
// call and given back if the call fails server-side, so customers aren't
// charged for errors.
func (h *Handler) Quotas(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		kind := quotaKind(r)
		if kind == "" || h.isDemo(r) {
			next.ServeHTTP(w, r)
			return
		}
		email, ok := h.sessionEmail(r)
		if !ok {
			next.ServeHTTP(w, r)
			return
		}
		m, err := h.store.GetMember(r.Context(), email)
		if err != nil {
			next.ServeHTTP(w, r) // no household: the resolver answers
			return
		}
		now := h.now()
		limit := MonthlyQuotas[kind]
		if _, err := h.store.ConsumeQuota(r.Context(), m.HouseholdID, kind, period(now), limit); errors.Is(err, controlplane.ErrQuotaExceeded) {
			http.Error(w, fmt.Sprintf("You've used this month's %d %s. More on %s.", limit, quotaNoun[kind], nextReset(now).Format("2 January")), http.StatusTooManyRequests)
			return
		} else if err != nil {
			serverError(w, "checking your monthly allowance", err)
			return
		}
		sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(sw, r)
		if sw.status >= 500 {
			h.store.ReleaseQuota(r.Context(), m.HouseholdID, kind, period(now))
		}
	})
}

type statusWriter struct {
	http.ResponseWriter
	status int
	wrote  bool
}

func (s *statusWriter) WriteHeader(code int) {
	if !s.wrote {
		s.status, s.wrote = code, true
	}
	s.ResponseWriter.WriteHeader(code)
}

func (s *statusWriter) Write(b []byte) (int, error) {
	s.wrote = true
	return s.ResponseWriter.Write(b)
}

// Flush keeps streaming responses (Ask Kosh) streaming.
func (s *statusWriter) Flush() {
	if f, ok := s.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}
