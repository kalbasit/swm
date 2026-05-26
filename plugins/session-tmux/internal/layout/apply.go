package layout

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// RunFunc executes a tmux subcommand and returns its stdout.
// It mirrors the signature of (*Tmux).run so callers can pass it directly.
type RunFunc func(ctx context.Context, args ...string) (string, error)

// paneResult holds the ID and config of a leaf pane after layout application.
type paneResult struct {
	paneID string
	pane   Pane
}

// Apply applies the layout defined in cfg to an already-created tmux session.
// sock is the tmux socket path; sessionName is the pane group session name.
func Apply(ctx context.Context, run RunFunc, sock, sessionName string, cfg *Config) error {
	sessionPath := cfg.Path

	// Apply session-level env vars before creating any panes.
	for k, v := range cfg.Env {
		if _, err := run(ctx, "-S", sock, "setenv", "-t", sessionName, k, v); err != nil {
			return fmt.Errorf("setting session env %s: %w", k, err)
		}
	}

	// Apply startup commands to the initial pane before any layout steps.
	if len(cfg.Startup) > 0 {
		initialPaneID, err := getPaneID(ctx, run, sock, sessionName+":0")
		if err != nil {
			return err
		}

		for _, cmd := range cfg.Startup {
			if err := sendKeys(ctx, run, sock, initialPaneID, cmdString(cmd), cfg.PaneCmdDelay); err != nil {
				return err
			}
		}
	}

	for i, w := range cfg.Windows {
		winPath := resolvePath(w.Path, sessionPath)

		if i == 0 {
			// Rename the existing default window.
			if _, err := run(ctx, "-S", sock, "rename-window", "-t", sessionName+":0", w.Name); err != nil {
				return fmt.Errorf("renaming first window to %q: %w", w.Name, err)
			}
		} else {
			// Create subsequent windows.
			args := []string{"-S", sock, "new-window", "-t", sessionName, "-n", w.Name}
			if winPath != "" {
				args = append(args, "-c", winPath)
			}

			if _, err := run(ctx, args...); err != nil {
				return fmt.Errorf("creating window %q: %w", w.Name, err)
			}
		}

		if len(w.Panes) == 0 {
			continue
		}

		winTarget := fmt.Sprintf("%s:%d", sessionName, i)

		initialPaneID, err := getPaneID(ctx, run, sock, winTarget)
		if err != nil {
			return fmt.Errorf("getting initial pane ID for window %d: %w", i, err)
		}

		leafPanes, err := layoutPanes(ctx, run, sock, w.Panes, initialPaneID, w.FlexDirection, winPath, cfg.PaneCmdDelay)
		if err != nil {
			return fmt.Errorf("laying out window %q: %w", w.Name, err)
		}

		// Apply focus then zoom (focus must come before zoom per spec).
		for _, r := range leafPanes {
			if r.pane.Focus {
				if _, err := run(ctx, "-S", sock, "select-pane", "-t", r.paneID); err != nil {
					return fmt.Errorf("select-pane %s: %w", r.paneID, err)
				}
			}
		}

		for _, r := range leafPanes {
			if r.pane.Zoom {
				if _, err := run(ctx, "-S", sock, "resize-pane", "-Z", "-t", r.paneID); err != nil {
					return fmt.Errorf("resize-pane -Z %s: %w", r.paneID, err)
				}
			}
		}
	}

	return nil
}

// layoutPanes recursively splits parentPaneID into len(panes) regions according
// to each pane's flex weight, then applies content to leaf panes.
// Returns all leaf pane results so the caller can apply focus/zoom.
func layoutPanes(ctx context.Context, run RunFunc, sock string, panes []Pane, parentPaneID string, dir FlexDirection, parentPath string, paneCmdDelay int) ([]paneResult, error) {
	if len(panes) == 0 {
		return nil, nil
	}

	type allocation struct {
		pane   Pane
		paneID string
	}

	allocs := make([]allocation, 0, len(panes))
	currentID := parentPaneID
	remainingFlex := sumFlex(panes)

	for i, p := range panes {
		eFlex := p.effectiveFlex()
		allocs = append(allocs, allocation{pane: p, paneID: currentID})
		remainingFlex -= eFlex

		if i < len(panes)-1 {
			// split-window -p N: current pane keeps (100-N)%, new pane gets N%.
			pct := splitPercent(eFlex, remainingFlex)
			nextPath := resolvePath(panes[i+1].Path, parentPath)
			newID, err := run(ctx, buildSplitArgs(sock, pct, currentID, dir, nextPath)...)
			if err != nil {
				return nil, fmt.Errorf("splitting pane %s: %w", currentID, err)
			}

			currentID = strings.TrimSpace(newID)
		}
	}

	var results []paneResult

	for _, a := range allocs {
		effectivePath := resolvePath(a.pane.Path, parentPath)

		if len(a.pane.Panes) > 0 {
			children, err := layoutPanes(ctx, run, sock, a.pane.Panes, a.paneID, a.pane.FlexDirection, effectivePath, paneCmdDelay)
			if err != nil {
				return nil, err
			}

			results = append(results, children...)
		} else {
			if err := applyPaneContent(ctx, run, sock, a.pane, a.paneID, effectivePath, paneCmdDelay); err != nil {
				return nil, err
			}

			results = append(results, paneResult{paneID: a.paneID, pane: a.pane})
		}
	}

	return results, nil
}

func applyPaneContent(ctx context.Context, run RunFunc, sock string, p Pane, paneID, path string, paneCmdDelay int) error {
	for k, v := range p.Env {
		if err := sendKeys(ctx, run, sock, paneID, "export "+k+"="+v, paneCmdDelay); err != nil {
			return fmt.Errorf("exporting env %s in pane %s: %w", k, paneID, err)
		}
	}

	if p.Path != "" && path != "" {
		if err := sendKeys(ctx, run, sock, paneID, "cd "+path, paneCmdDelay); err != nil {
			return fmt.Errorf("cd %s in pane %s: %w", path, paneID, err)
		}
	}

	for _, cmd := range p.Commands {
		if err := sendKeys(ctx, run, sock, paneID, cmd, paneCmdDelay); err != nil {
			return err
		}
	}

	return nil
}

func sendKeys(ctx context.Context, run RunFunc, sock, paneID, cmd string, delayMs int) error {
	if delayMs > 0 {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Duration(delayMs) * time.Millisecond):
		}
	}

	if _, err := run(ctx, "-S", sock, "send-keys", "-t", paneID, cmd, "Enter"); err != nil {
		return fmt.Errorf("send-keys %q to pane %s: %w", cmd, paneID, err)
	}

	return nil
}

func getPaneID(ctx context.Context, run RunFunc, sock, target string) (string, error) {
	id, err := run(ctx, "-S", sock, "display-message", "-t", target, "-p", "#{pane_id}")
	if err != nil {
		return "", fmt.Errorf("getting pane ID for %s: %w", target, err)
	}

	return strings.TrimSpace(id), nil
}

func buildSplitArgs(sock string, pct int, targetPaneID string, dir FlexDirection, path string) []string {
	args := []string{"-S", sock, "split-window", "-P", "-F", "#{pane_id}", "-p", strconv.Itoa(pct)}
	if dir == FlexDirectionRow {
		args = append(args, "-h")
	}

	if path != "" {
		args = append(args, "-c", path)
	}

	return append(args, "-t", targetPaneID)
}

// resolvePath resolves path relative to basePath.
// Absolute paths and ~ paths are returned as-is. Empty path → basePath.
func resolvePath(path, basePath string) string {
	if path == "" {
		return basePath
	}

	if path[0] == '/' || path[0] == '~' {
		return path
	}

	if basePath == "" {
		return path
	}

	return basePath + "/" + path
}

func cmdString(c Command) string {
	if len(c.Args) == 0 {
		return c.Command
	}

	quoted := make([]string, len(c.Args))
	for i, arg := range c.Args {
		quoted[i] = shellQuote(arg)
	}

	return c.Command + " " + strings.Join(quoted, " ")
}

// shellQuote wraps arg in single quotes if it contains shell metacharacters.
// Single quotes inside the arg are escaped using the ”\” idiom.
func shellQuote(arg string) string {
	if arg == "" {
		return "''"
	}

	if !strings.ContainsAny(arg, " \t\n&*;<>|'\"()$[]?~`{}!\\") {
		return arg
	}

	return "'" + strings.ReplaceAll(arg, "'", `'\''`) + "'"
}
