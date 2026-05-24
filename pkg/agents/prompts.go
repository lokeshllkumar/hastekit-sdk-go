package agents

import (
	"context"
)

// DeferredToolInfo is the projection of a deferred Tool that prompt
// providers actually consume — just the schema's name and description.
// Kept as a plain struct (not the Tool interface) so Dependencies can
// JSON-roundtrip across Temporal activity boundaries; the full Tool
// interface carries closure state (workflow ctx, broker handles, etc.)
// that doesn't deserialize on the worker side.
type DeferredToolInfo struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

type Dependencies struct {
	RunContext    map[string]any
	Handoffs      []*Handoff
	DeferredTools []DeferredToolInfo
}

type SystemPromptProvider interface {
	GetPrompt(ctx context.Context, data *Dependencies) (string, error)
}
