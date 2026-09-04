package pane

import (
	"encoding/json"
	"fmt"
	"io"

	pluginv1 "github.com/kalbasit/swm/proto/swm/plugin/v1"
)

// paneView is the JSON shape of a Pane.
//
// It is hand-written rather than produced by protojson because protojson would
// emit lowerCamelCase keys, and every other machine-facing name in this project
// is snake_case — including the proto field names themselves.
//
// Every field is always emitted, including the descriptive ones a provider may
// leave empty. A caller can then index into the object without an existence
// check; an absent key would be a less clear way of saying "this provider does
// not report that" than a present empty one.
type paneView struct {
	PaneID         string `json:"pane_id"`
	PaneGroupID    string `json:"pane_group_id"`
	WorkspaceID    string `json:"workspace_id"`
	Title          string `json:"title"`
	CurrentCommand string `json:"current_command"`
	CurrentPath    string `json:"current_path"`
	Focused        bool   `json:"focused"`
}

// newPaneView projects a Pane onto its JSON shape.
func newPaneView(p *pluginv1.Pane) paneView {
	return paneView{
		PaneID:         p.GetPaneId(),
		PaneGroupID:    p.GetPaneGroupId(),
		WorkspaceID:    p.GetWorkspaceId(),
		Title:          p.GetTitle(),
		CurrentCommand: p.GetCurrentCommand(),
		CurrentPath:    p.GetCurrentPath(),
		Focused:        p.GetFocused(),
	}
}

// writeJSON encodes v to w, indented. Indentation costs a consumer nothing —
// no JSON parser cares — and makes the output readable when a person runs the
// command by hand to see what a caller will receive.
func writeJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")

	if err := enc.Encode(v); err != nil {
		return fmt.Errorf("encoding JSON output: %w", err)
	}

	return nil
}
