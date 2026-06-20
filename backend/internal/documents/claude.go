package documents

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// Parser extracts structured fields from a document using an AI model.
type Parser interface {
	Parse(ctx context.Context, docType, mimeType string, data []byte) (map[string]any, error)
}

// promptFiles maps document type keys to their prompt file names.
var promptFiles = map[string]string{
	"fd":        "fd.md",
	"insurance": "insurance.md",
	"metals":    "metals.md",
	"nps":       "nps.md",
}

// ClaudeParser calls the Anthropic API to extract structured fields from a document.
type ClaudeParser struct {
	apiKey     string
	promptsDir string
	client     *http.Client
}

// NewClaudeParser returns a ClaudeParser using the given API key, prompts directory,
// and HTTP client.
func NewClaudeParser(apiKey, promptsDir string, client *http.Client) *ClaudeParser {
	return &ClaudeParser{apiKey: apiKey, promptsDir: promptsDir, client: client}
}

// Parse sends the document to Claude and returns the extracted fields as a map.
func (p *ClaudeParser) Parse(ctx context.Context, docType, mimeType string, data []byte) (map[string]any, error) {
	promptFile, ok := promptFiles[docType]
	if !ok {
		return nil, fmt.Errorf("unknown document type %q", docType)
	}
	promptPath := filepath.Join(p.promptsDir, promptFile)
	promptBytes, err := os.ReadFile(promptPath)
	if err != nil {
		return nil, fmt.Errorf("reading prompt %s: %w", promptPath, err)
	}

	encoded := base64.StdEncoding.EncodeToString(data)

	// PDFs use the "document" block type; images use "image".
	var mediaBlock map[string]interface{}
	if mimeType == "application/pdf" {
		mediaBlock = map[string]interface{}{
			"type": "document",
			"source": map[string]interface{}{
				"type":       "base64",
				"media_type": mimeType,
				"data":       encoded,
			},
		}
	} else {
		mediaBlock = map[string]interface{}{
			"type": "image",
			"source": map[string]interface{}{
				"type":       "base64",
				"media_type": mimeType,
				"data":       encoded,
			},
		}
	}

	body, _ := json.Marshal(map[string]interface{}{
		"model":      "claude-haiku-4-5-20251001",
		"max_tokens": 1024,
		"messages": []map[string]interface{}{{
			"role": "user",
			"content": []interface{}{
				mediaBlock,
				map[string]interface{}{
					"type": "text",
					"text": string(promptBytes),
				},
			},
		}},
	})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.anthropic.com/v1/messages", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("building Anthropic request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", p.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("calling Anthropic API: %w", err)
	}
	defer resp.Body.Close()
	respBytes, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Anthropic returned %d: %s", resp.StatusCode, string(respBytes))
	}

	var anthResp struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(respBytes, &anthResp); err != nil || len(anthResp.Content) == 0 {
		return nil, fmt.Errorf("unexpected Anthropic response shape: %s", string(respBytes))
	}

	text := strings.TrimSpace(anthResp.Content[0].Text)
	// Strip markdown code fences the model sometimes adds despite instructions.
	if i := strings.Index(text, "{"); i > 0 {
		text = text[i:]
	}
	if i := strings.LastIndex(text, "}"); i >= 0 {
		text = text[:i+1]
	}

	var fields map[string]interface{}
	if err := json.Unmarshal([]byte(text), &fields); err != nil {
		return nil, fmt.Errorf("parsing extracted JSON (%w) — raw: %s", err, text)
	}
	return fields, nil
}
