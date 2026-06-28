package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// OllamaProvider calls a locally-running Ollama instance via its
// OpenAI-compatible /v1/chat/completions endpoint.
// Install: brew install ollama && ollama pull llama3.1
type OllamaProvider struct {
	baseURL string
	model   string
	client  *http.Client
}

func NewOllamaProvider(baseURL, model string, client *http.Client) *OllamaProvider {
	if client == nil {
		client = http.DefaultClient
	}
	return &OllamaProvider{baseURL: baseURL, model: model, client: client}
}

func (p *OllamaProvider) Chat(ctx context.Context, messages []Message) (string, error) {
	body, _ := json.Marshal(map[string]any{
		"model":    p.model,
		"messages": messages,
		"stream":   false,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		p.baseURL+"/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("ollama: %w — is Ollama running? (ollama serve)", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("ollama: HTTP %d — model %q may not be pulled yet (ollama pull %s)", resp.StatusCode, p.model, p.model)
	}

	var out struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", fmt.Errorf("ollama: decode: %w", err)
	}
	if len(out.Choices) == 0 {
		return "", fmt.Errorf("ollama: no choices in response")
	}
	return out.Choices[0].Message.Content, nil
}
