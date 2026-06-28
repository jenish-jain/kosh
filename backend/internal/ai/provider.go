package ai

import "context"

// Message is a single turn in a conversation.
type Message struct {
	Role    string `json:"role"`    // "system" | "user" | "assistant"
	Content string `json:"content"`
}

// Provider is the interface that all LLM backends implement.
type Provider interface {
	Chat(ctx context.Context, messages []Message) (string, error)
}
