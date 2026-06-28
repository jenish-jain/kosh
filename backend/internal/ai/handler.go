package ai

import (
	"encoding/json"
	"net/http"

	"kosh/internal/models"
)

// DataLoader fetches the financial data appropriate for this request
// (live data, demo data, or dev data — resolved by the caller).
type DataLoader func(r *http.Request) *models.Data

// Handler serves POST /api/chat.
type Handler struct {
	provider   Provider
	loadData   DataLoader
	isDemo     func(*http.Request) bool
}

func NewHandler(provider Provider, loadData DataLoader, isDemo func(*http.Request) bool) *Handler {
	return &Handler{provider: provider, loadData: loadData, isDemo: isDemo}
}

func (h *Handler) Handle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Return a friendly message for demo sessions instead of hitting the real provider.
	if h.isDemo != nil && h.isDemo(r) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"reply": "Ask Kosh is not available in demo mode. To enable it, deploy your own instance with AI_ENABLED=true and your preferred AI provider configured (Ollama for free local inference, or an Anthropic API key for cloud). Your questions will then be answered using your actual financial data.",
		})
		return
	}

	var req struct {
		Messages []Message `json:"messages"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || len(req.Messages) == 0 {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Load financial data and prepend it as a system message.
	data := h.loadData(r)
	systemMsg := Message{Role: "system", Content: BuildSystemPrompt(data)}
	messages := append([]Message{systemMsg}, req.Messages...)

	reply, err := h.provider.Chat(r.Context(), messages)
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusBadGateway)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"reply": reply})
}
