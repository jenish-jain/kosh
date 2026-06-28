package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// AnthropicProvider calls the Anthropic Messages API.
// Uses claude-haiku-4-5 by default — cheapest model, ~₹0.04/question.
type AnthropicProvider struct {
	apiKey string
	model  string
	client *http.Client
}

func NewAnthropicProvider(apiKey, model string, client *http.Client) *AnthropicProvider {
	if client == nil {
		client = http.DefaultClient
	}
	if model == "" {
		model = "claude-haiku-4-5-20251001"
	}
	return &AnthropicProvider{apiKey: apiKey, model: model, client: client}
}

func (p *AnthropicProvider) Chat(ctx context.Context, messages []Message) (string, error) {
	// Anthropic takes system as a top-level field, not as a message.
	var system string
	var msgs []map[string]string
	for _, m := range messages {
		if m.Role == "system" {
			system = m.Content
		} else {
			msgs = append(msgs, map[string]string{"role": m.Role, "content": m.Content})
		}
	}

	payload := map[string]any{
		"model":      p.model,
		"max_tokens": 1024,
		"messages":   msgs,
	}
	if system != "" {
		payload["system"] = system
	}

	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://api.anthropic.com/v1/messages", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", p.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := p.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("anthropic: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errBody struct{ Error struct{ Message string } `json:"error"` }
		json.NewDecoder(resp.Body).Decode(&errBody)
		return "", fmt.Errorf("anthropic: HTTP %d — %s", resp.StatusCode, errBody.Error.Message)
	}

	var out struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", fmt.Errorf("anthropic: decode: %w", err)
	}
	if len(out.Content) == 0 {
		return "", fmt.Errorf("anthropic: empty response")
	}
	return out.Content[0].Text, nil
}
