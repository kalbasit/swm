// Package pluginmgr handles discovery, launch, and lifecycle of swm plugins.
package pluginmgr

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"

	"github.com/adrg/xdg"

	goplugin "github.com/hashicorp/go-plugin"
	pluginv1 "github.com/kalbasit/swm/proto/swm/plugin/v1"
	sdksession "github.com/kalbasit/swm/sdk/go/session"
	sdkvcs "github.com/kalbasit/swm/sdk/go/vcs"

	"github.com/kalbasit/swm/cmd/swm/internal/config"
	"github.com/kalbasit/swm/sdk/go/handshake"
)

// VCSClient wraps pluginv1.VCSClient as a named type for type assertions.
type VCSClient = pluginv1.VCSClient

// SessionClient wraps pluginv1.SessionClient as a named type for type assertions.
type SessionClient = pluginv1.SessionClient

const (
	capabilityPicker  = "picker"
	capabilitySession = "session"
	capabilityVCS     = "vcs"
)

// Sentinel errors for plugin capability configuration.
var (
	errNoPickerPlugin    = errors.New("no picker plugin configured")
	errNoSessionPlugin   = errors.New("no session plugin configured")
	errNoVCSPlugin       = errors.New("no vcs plugin configured")
	errPluginNotFound    = errors.New("plugin binary not found")
	errPluginMissingDep  = errors.New("plugin missing required capability")
	errUnknownCapability = errors.New("unknown capability")
	errUnsupported       = errors.New("unsupported capability")
)

type entry struct {
	client *goplugin.Client
	raw    any
}

// Manager discovers, launches, and provides typed access to swm plugins.
type Manager struct {
	cfg        *config.Config
	hostSocket string

	mu       sync.Mutex
	launched map[string]*entry
}

// New returns a Manager. Plugins are not launched until Get is called.
func New(cfg *config.Config, hostSocket string) *Manager {
	return &Manager{
		cfg:        cfg,
		hostSocket: hostSocket,
		launched:   make(map[string]*entry),
	}
}

// Close terminates all launched plugin processes.
func (m *Manager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var errs []error

	for _, e := range m.launched {
		e.client.Kill()
	}

	m.launched = make(map[string]*entry)

	return errors.Join(errs...)
}

// Get returns the client for the configured plugin of the given capability.
// The plugin is lazily launched on the first call and cached for subsequent calls.
func (m *Manager) Get(ctx context.Context, capability string) (any, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if e, ok := m.launched[capability]; ok {
		return e.raw, nil
	}

	// Look up the plugin name from config.
	name, err := m.capabilityName(capability)
	if err != nil {
		return nil, err
	}

	binary, err := m.discover(capability, name)
	if err != nil {
		return nil, err
	}

	set := pluginSet(capability)
	if len(set) == 0 {
		return nil, fmt.Errorf("%w: %s", errUnsupported, capability)
	}

	// Pre-populate Cmd.Env with the host socket address; go-plugin will append
	// os.Environ() (since SkipHostEnv defaults to false), so this var stays first.
	pluginCmd := exec.Command(binary) //nolint:gosec // binary is discovered from trusted sources
	if m.hostSocket != "" {
		pluginCmd.Env = []string{"SWM_HOST_SOCKET=" + m.hostSocket}
	}

	client := goplugin.NewClient(&goplugin.ClientConfig{
		HandshakeConfig: handshake.Config,
		Plugins:         set,
		Cmd:             pluginCmd,
		AllowedProtocols: []goplugin.Protocol{
			goplugin.ProtocolGRPC,
		},
	})

	rpcClient, err := client.Client()
	if err != nil {
		client.Kill()

		return nil, fmt.Errorf("connecting to plugin %s: %w", binary, err)
	}

	raw, err := rpcClient.Dispense(capability)
	if err != nil {
		client.Kill()

		return nil, fmt.Errorf("dispensing capability %s: %w", capability, err)
	}

	if err := m.validateDeps(ctx, capability, raw); err != nil {
		client.Kill()

		return nil, err
	}

	m.launched[capability] = &entry{client: client, raw: raw}

	return raw, nil
}

// capabilityName returns the plugin name from config for the given capability.
func (m *Manager) capabilityName(capability string) (string, error) {
	switch capability {
	case capabilitySession:
		if m.cfg.Plugins.Session == "" {
			return "", errNoSessionPlugin
		}

		return m.cfg.Plugins.Session, nil
	case capabilityVCS:
		if m.cfg.Plugins.VCS == "" {
			return "", errNoVCSPlugin
		}

		return m.cfg.Plugins.VCS, nil
	case capabilityPicker:
		if m.cfg.Plugins.Picker == "" {
			return "", errNoPickerPlugin
		}

		return m.cfg.Plugins.Picker, nil
	default:
		return "", fmt.Errorf("%w: %s", errUnknownCapability, capability)
	}
}

// discover finds the binary for the plugin providing the given capability with the given name.
// Search order: (1) explicit config path, (2) XDG plugins dir, (3) PATH.
func (m *Manager) discover(capability, name string) (string, error) {
	binary := "swm-plugin-" + capability + "-" + name

	// 1. Explicit config path.
	if explicit, ok := m.cfg.Plugins.Paths[name]; ok {
		if _, err := os.Stat(explicit); err == nil {
			return explicit, nil
		}
	}

	// 2. XDG data dir: $XDG_DATA_HOME/swm/plugins/<name>/<binary>.
	xdgPath := filepath.Join(xdg.DataHome, "swm", "plugins", name, binary)
	if _, err := os.Stat(xdgPath); err == nil {
		return xdgPath, nil
	}

	// 3. PATH lookup.
	if path, err := exec.LookPath(binary); err == nil {
		return path, nil
	}

	return "", fmt.Errorf("%w: %q not in config paths, %s, or PATH", errPluginNotFound, binary, xdgPath)
}

// validateDeps calls Info() on the plugin and checks required capability deps.
func (m *Manager) validateDeps(ctx context.Context, capability string, raw any) error {
	var info *pluginv1.PluginInfo

	switch capability {
	case capabilityVCS:
		if c, ok := raw.(pluginv1.VCSClient); ok {
			resp, err := c.Info(ctx, &pluginv1.Empty{})
			if err != nil {
				return fmt.Errorf("calling Info on vcs plugin: %w", err)
			}

			info = resp.GetPluginInfo()
		}
	case capabilitySession:
		if c, ok := raw.(pluginv1.SessionClient); ok {
			resp, err := c.Info(ctx, &pluginv1.Empty{})
			if err != nil {
				return fmt.Errorf("calling Info on session plugin: %w", err)
			}

			info = resp.GetPluginInfo()
		}
	}

	if info == nil {
		return nil
	}

	for _, dep := range info.GetRequires() {
		depCap := dep.GetCapability().String()
		if _, configured := m.capabilityName(depCap); configured != nil {
			return fmt.Errorf("%w: %q requires %q", errPluginMissingDep, info.GetName(), depCap)
		}
	}

	return nil
}

// pluginSet returns the go-plugin PluginSet for the given capability.
func pluginSet(capability string) goplugin.PluginSet {
	switch capability {
	case capabilitySession:
		return goplugin.PluginSet{capabilitySession: &sdksession.GRPCPlugin{}}
	case capabilityVCS:
		return goplugin.PluginSet{capabilityVCS: &sdkvcs.GRPCPlugin{}}
	default:
		return goplugin.PluginSet{}
	}
}
