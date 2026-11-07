package billing

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"

	"kosh/internal/controlplane"
)

// Operator serves /api/operator/* — the service operator's settings — to
// the accounts listed in OPERATOR_EMAILS. Everyone else gets a 404, so the
// endpoints aren't advertised.
type Operator struct {
	policy       *Policy
	emails       map[string]bool
	sessionEmail func(*http.Request) (string, bool)
	isDemo       func(*http.Request) bool
}

func NewOperator(policy *Policy, emails []string, sessionEmail func(*http.Request) (string, bool), isDemo func(*http.Request) bool) *Operator {
	o := &Operator{policy: policy, emails: map[string]bool{}, sessionEmail: sessionEmail, isDemo: isDemo}
	for _, e := range emails {
		if e = controlplane.NormalizeEmail(e); e != "" {
			o.emails[e] = true
		}
	}
	return o
}

// Count is how many operator accounts are configured.
func (o *Operator) Count() int { return len(o.emails) }

// Is reports whether email is an operator.
func (o *Operator) Is(email string) bool { return o.emails[controlplane.NormalizeEmail(email)] }

func (o *Operator) Routes(w http.ResponseWriter, r *http.Request) {
	email, ok := o.sessionEmail(r)
	if !ok || o.isDemo(r) || !o.Is(email) {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	switch path := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/operator/"), "/"); {
	case path == "features" && r.Method == http.MethodGet:
		o.getFeatures(w, r)
	case path == "features" && r.Method == http.MethodPut:
		o.putFeatures(w, r, controlplane.NormalizeEmail(email))
	default:
		http.Error(w, "not found", http.StatusNotFound)
	}
}

type featuresJSON struct {
	Tiers    []TierInfo                 `json:"tiers"`
	Features []FeatureInfo              `json:"features"`
	Defaults map[string]map[string]bool `json:"defaults"`
	PolicyView
}

func (o *Operator) getFeatures(w http.ResponseWriter, r *http.Request) {
	v, err := o.policy.Current(r.Context())
	if err != nil {
		serverError(w, "loading the feature policy", err)
		return
	}
	writeJSON(w, featuresJSON{Tiers: Tiers, Features: Features, Defaults: DefaultPolicy(), PolicyView: v})
}

func (o *Operator) putFeatures(w http.ResponseWriter, r *http.Request, email string) {
	var body struct {
		Policy map[string]map[string]bool `json:"policy"`
	}
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16<<10))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&body); err != nil || body.Policy == nil {
		http.Error(w, "Send {\"policy\": {tier: {feature: true|false}}}.", http.StatusBadRequest)
		return
	}
	v, err := o.policy.Set(r.Context(), body.Policy, email)
	if errors.Is(err, ErrInvalidPolicy) {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err != nil {
		serverError(w, "saving the feature policy", err)
		return
	}
	log.Printf("billing: %s changed the feature policy: %v", email, v.Tiers)
	writeJSON(w, featuresJSON{Tiers: Tiers, Features: Features, Defaults: DefaultPolicy(), PolicyView: v})
}
