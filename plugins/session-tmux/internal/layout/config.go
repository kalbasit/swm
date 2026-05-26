// Package layout loads and validates per-repo / global tmux layout configs.
package layout

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"github.com/pelletier/go-toml/v2"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// FlexDirection controls how sibling panes are split.
type FlexDirection string

const (
	// FlexDirectionColumn splits panes vertically (top/bottom stacked). This is the default.
	FlexDirectionColumn FlexDirection = "column"
	// FlexDirectionRow splits panes horizontally (left/right side by side).
	FlexDirectionRow FlexDirection = "row"
)

// TemplateVars holds variables injected into config values before TOML parsing.
type TemplateVars struct {
	WorktreePath string
	StoryName    string
	ProjectID    string
	TmuxSocket   string
}

// Command is an executable with optional arguments (used for startup commands).
type Command struct {
	Command string   `toml:"command"`
	Args    []string `toml:"args"`
}

// Pane describes one tmux pane. Panes are recursive — a Pane may contain
// child Panes forming a nested split tree.
type Pane struct {
	Flex          *int              `toml:"flex"`
	FlexDirection FlexDirection     `toml:"flex_direction"`
	Path          string            `toml:"path"`
	Shell         string            `toml:"shell"`
	Commands      []string          `toml:"commands"`
	Env           map[string]string `toml:"env"`
	Focus         bool              `toml:"focus"`
	Zoom          bool              `toml:"zoom"`
	Panes         []Pane            `toml:"panes"`
}

// effectiveFlex returns the pane's flex weight, defaulting to 1 when unset.
func (p *Pane) effectiveFlex() int {
	if p.Flex == nil {
		return 1
	}

	return *p.Flex
}

// Window describes a named tmux window and its pane layout tree.
type Window struct {
	Name          string        `toml:"name"`
	Path          string        `toml:"path"`
	FlexDirection FlexDirection `toml:"flex_direction"`
	Panes         []Pane        `toml:"panes"`
}

// Config is the parsed layout configuration.
type Config struct {
	Path         string            `toml:"path"`
	Shell        string            `toml:"shell"`
	PaneCmdDelay int               `toml:"pane_cmd_delay"`
	Env          map[string]string `toml:"env"`
	Startup      []Command         `toml:"startup"`
	Windows      []Window          `toml:"windows"`
}

// LoadConfig resolves a layout config using a two-tier lookup (per-repo wins over global):
//
//  1. <worktreePath>/.swm/session-tmux.toml
//  2. <xdgConfigHome>/swm/session-tmux.toml
//
// Returns nil, nil when no config file is found at either tier.
func LoadConfig(worktreePath, xdgConfigHome string, vars TemplateVars) (*Config, error) {
	candidates := []string{
		filepath.Join(worktreePath, ".swm", "session-tmux.toml"),
	}
	if xdgConfigHome != "" {
		candidates = append(candidates, filepath.Join(xdgConfigHome, "swm", "session-tmux.toml"))
	}

	for _, p := range candidates {
		cfg, err := loadFile(p, vars)
		if err != nil {
			return nil, err
		}

		if cfg != nil {
			return cfg, nil
		}
	}

	return nil, nil //nolint:nilnil // nil Config means "no config file found" — not an error condition
}

func loadFile(path string, vars TemplateVars) (*Config, error) {
	raw, err := os.ReadFile(path) //nolint:gosec // path is composed from trusted config directories
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil //nolint:nilnil // nil Config means "file not found" — not an error condition
		}

		return nil, fmt.Errorf("reading layout config %s: %w", path, err)
	}

	tmpl, err := template.New("layout").Parse(string(raw))
	if err != nil {
		return nil, fmt.Errorf("parsing layout config template %s: %w", path, err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, vars); err != nil {
		return nil, fmt.Errorf("rendering layout config template %s: %w", path, err)
	}

	var cfg Config
	if err := toml.Unmarshal(buf.Bytes(), &cfg); err != nil {
		return nil, fmt.Errorf("parsing layout config TOML %s: %w", path, err)
	}

	if err := validateConfig(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func validateConfig(cfg *Config) error {
	if len(cfg.Windows) == 0 {
		return status.Errorf(codes.InvalidArgument, "layout config must define at least one window")
	}

	for i, w := range cfg.Windows {
		if w.Name == "" {
			return status.Errorf(codes.InvalidArgument, "layout: window[%d] must have a non-empty name", i)
		}

		if err := validateFlexDirection(w.FlexDirection, fmt.Sprintf("window[%d](%q)", i, w.Name)); err != nil {
			return err
		}

		var focusCount, zoomCount int
		if err := validatePanes(w.Panes, fmt.Sprintf("window[%d](%q)", i, w.Name), &focusCount, &zoomCount); err != nil {
			return err
		}
	}

	return nil
}

func validateFlexDirection(dir FlexDirection, ctx string) error {
	switch dir {
	case "", FlexDirectionColumn, FlexDirectionRow:
		return nil
	default:
		return status.Errorf(codes.InvalidArgument,
			"layout: %s has an invalid flex_direction %q (must be \"row\" or \"column\")", ctx, dir)
	}
}

func validatePanes(panes []Pane, ctx string, focusCount, zoomCount *int) error {
	for i, p := range panes {
		paneCtx := fmt.Sprintf("%s.panes[%d]", ctx, i)

		if err := validateFlexDirection(p.FlexDirection, paneCtx); err != nil {
			return err
		}

		if p.Flex != nil && *p.Flex < 1 {
			return status.Errorf(codes.InvalidArgument, "layout: %s has a pane with flex=%d (must be ≥ 1)", ctx, *p.Flex)
		}

		if p.Focus {
			*focusCount++
			if *focusCount > 1 {
				return status.Errorf(codes.InvalidArgument, "layout: %s has more than one pane with focus=true", ctx)
			}
		}

		if p.Zoom {
			*zoomCount++
			if *zoomCount > 1 {
				return status.Errorf(codes.InvalidArgument, "layout: %s has more than one pane with zoom=true", ctx)
			}
		}

		if err := validatePanes(p.Panes, paneCtx, focusCount, zoomCount); err != nil {
			return err
		}
	}

	return nil
}
