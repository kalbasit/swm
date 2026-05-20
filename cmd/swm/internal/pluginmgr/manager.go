// Package pluginmgr handles discovery, launch, and lifecycle of swm plugins.
package pluginmgr

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"sync"
	"sync/atomic"

	"github.com/adrg/xdg"

	hclog "github.com/hashicorp/go-hclog"
	goplugin "github.com/hashicorp/go-plugin"
	pluginv1 "github.com/kalbasit/swm/proto/swm/plugin/v1"
	sdkforge "github.com/kalbasit/swm/sdk/go/forge"
	sdkpicker "github.com/kalbasit/swm/sdk/go/picker"
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
	capabilityForge   = "forge"
	capabilityPicker  = "picker"
	capabilitySession = "session"
	capabilityVCS     = "vcs"
)

// Sentinel errors for plugin capability configuration.
var (
	errInvalidForgePlugin = errors.New("forge plugin did not return a ForgeClient")
	errNoForgePlugin      = errors.New("no forge plugin configured for hostname")
	errNoPickerPlugin     = errors.New("no picker plugin configured")
	errNoSessionPlugin    = errors.New("no session plugin configured")
	errNoVCSPlugin        = errors.New("no vcs plugin configured")
	errPluginNotFound     = errors.New("plugin binary not found")
	errPluginMissingDep   = errors.New("plugin missing required capability")
	errUnknownCapability  = errors.New("unknown capability")
	errUnsupported        = errors.New("unsupported capability")
)

// launchOnce holds the result of a single plugin launch attempt.
// once.Do ensures the launch runs exactly once; the result is cached permanently
// (including errors — a failed launch is not retried).
// done is set to true after once.Do's function returns; Close checks this instead of
// calling once.Do(func(){}) (which would steal the Once if launch hasn't started yet).
type launchOnce struct {
	once   sync.Once
	done   atomic.Bool
	client *goplugin.Client
	raw    any
	err    error
}

type forgeEntry struct {
	client    *goplugin.Client
	forge     pluginv1.ForgeClient
	hostnames []string
}

// Option configures a Manager.
type Option func(*Manager)

// WithStderr sets the writer that receives the raw stderr output of plugin processes.
// The provided writer must be thread-safe as it may be shared by multiple concurrent plugins.
// Defaults to os.Stderr when not specified.
func WithStderr(w io.Writer) Option {
	return func(m *Manager) {
		m.stderr = w
	}
}

// Manager discovers, launches, and provides typed access to swm plugins.
type Manager struct {
	cfg        *config.Config
	hostSocket string
	stderr     io.Writer

	// launched stores *launchOnce per capability, enabling per-capability locking
	// so concurrent Get/Warm calls for different capabilities do not serialize.
	launched sync.Map

	mu           sync.Mutex // guards forge state only
	forgeClients []*forgeEntry
	forgesLoaded bool
}

// New returns a Manager. Plugins are not launched until Get is called.
func New(cfg *config.Config, hostSocket string, opts ...Option) *Manager {
	m := &Manager{
		cfg:        cfg,
		hostSocket: hostSocket,
		stderr:     os.Stderr,
	}

	for _, o := range opts {
		o(m)
	}

	return m
}

// Close terminates all launched plugin processes.
func (m *Manager) Close() error {
	m.launched.Range(func(k, v any) bool {
		lo, ok := v.(*launchOnce)
		if !ok {
			return true
		}

		// Only kill if launch has fully completed; done is set after once.Do's function
		// returns, ensuring client is visible without a data race.
		// A launch that is in-progress or never started is skipped — the OS cleans up.
		if lo.done.Load() && lo.client != nil {
			lo.client.Kill()
		}

		m.launched.Delete(k)

		return true
	})

	m.mu.Lock()
	defer m.mu.Unlock()

	for _, fe := range m.forgeClients {
		fe.client.Kill()
	}

	m.forgeClients = nil
	m.forgesLoaded = false

	return nil
}

// Get returns the client for the configured plugin of the given capability.
// The plugin is lazily launched on the first call and cached for subsequent calls.
// A failed launch is also cached — the same error is returned on every subsequent call.
func (m *Manager) Get(ctx context.Context, capability string) (any, error) {
	stored, _ := m.launched.LoadOrStore(capability, &launchOnce{})

	entry, ok := stored.(*launchOnce)
	if !ok {
		return nil, fmt.Errorf("%w: launched map contains unexpected type %T", errUnknownCapability, stored)
	}

	entry.once.Do(func() {
		entry.client, entry.raw, entry.err = m.launch(ctx, capability)
		entry.done.Store(true)
	})

	return entry.raw, entry.err
}

// GetForge returns the ForgeClient for the plugin claiming the given hostname.
// All configured forge plugins are lazily launched on the first call.
func (m *Manager) GetForge(ctx context.Context, hostname string) (pluginv1.ForgeClient, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.forgesLoaded {
		if err := m.loadForges(ctx); err != nil {
			return nil, err
		}

		m.forgesLoaded = true
	}

	for _, fe := range m.forgeClients {
		if slices.Contains(fe.hostnames, hostname) {
			return fe.forge, nil
		}
	}

	return nil, fmt.Errorf("%w: %q", errNoForgePlugin, hostname)
}

// Warm pre-starts the listed capabilities in background goroutines and returns
// immediately. Errors from plugin startup are not returned here; they are
// surfaced by the first Get call for the failing capability.
// Capabilities already running are reused without relaunching.
func (m *Manager) Warm(ctx context.Context, capabilities ...string) error {
	// Strip cancellation so goroutines outlive the caller's context (e.g. PreRunE).
	bgCtx := context.WithoutCancel(ctx)

	for _, c := range capabilities {
		stored, loaded := m.launched.LoadOrStore(c, &launchOnce{})
		if loaded {
			continue
		}

		lo, ok := stored.(*launchOnce)
		if !ok {
			continue
		}

		go func(capability string) {
			lo.once.Do(func() {
				lo.client, lo.raw, lo.err = m.launch(bgCtx, capability)
				lo.done.Store(true)
			})
		}(c)
	}

	return nil
}

// hclogLevelFromSlog maps a slog logger's effective level to the corresponding hclog level.
// go-plugin uses hclog for its internal lifecycle logging; this keeps it consistent with swm's --log-level.
func hclogLevelFromSlog(ctx context.Context, logger *slog.Logger) hclog.Level {
	switch {
	case logger.Enabled(ctx, slog.LevelDebug):
		return hclog.Debug
	case logger.Enabled(ctx, slog.LevelInfo):
		return hclog.Info
	case logger.Enabled(ctx, slog.LevelWarn):
		return hclog.Warn
	case logger.Enabled(ctx, slog.LevelError):
		return hclog.Error
	default:
		return hclog.Warn
	}
}

// buildClientConfig constructs a goplugin.ClientConfig with the hclog logger level
// derived from the current slog default, so go-plugin respects swm's --log-level flag.
// It also sets SWM_LOG_LEVEL on the plugin process so the plugin-side hclog matches.
func (m *Manager) buildClientConfig(
	ctx context.Context,
	pluginCmd *exec.Cmd,
	set goplugin.PluginSet,
) *goplugin.ClientConfig {
	level := hclogLevelFromSlog(ctx, slog.Default())
	pluginCmd.Env = append(pluginCmd.Env, "SWM_LOG_LEVEL="+level.String())

	return &goplugin.ClientConfig{
		HandshakeConfig: handshake.Config,
		Plugins:         set,
		Cmd:             pluginCmd,
		Stderr:          newLevelFilterWriter(m.stderr, level),
		Logger: hclog.New(&hclog.LoggerOptions{
			Level:  level,
			Output: m.stderr,
		}),
		AllowedProtocols: []goplugin.Protocol{
			goplugin.ProtocolGRPC,
		},
	}
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
// Search order: (0) SWM_PLUGIN_PATH dirs, (1) explicit config path, (2) XDG plugins dir, (3) PATH.
func (m *Manager) discover(capability, name string) (string, error) {
	binary := "swm-plugin-" + capability + "-" + name

	// 0. SWM_PLUGIN_PATH: platform-specific path list, searched left-to-right.
	// Non-existent or non-directory entries are silently skipped.
	for _, dir := range filepath.SplitList(os.Getenv("SWM_PLUGIN_PATH")) {
		candidate := filepath.Join(dir, binary)
		if _, err := os.Stat(candidate); err == nil { //nolint:gosec // SWM_PLUGIN_PATH is user-owned
			return candidate, nil
		}
	}

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

// launch performs the actual plugin binary discovery, exec, and gRPC handshake.
// It is called inside launchOnce.once.Do and must not hold any Manager-level locks.
func (m *Manager) launch(ctx context.Context, capability string) (*goplugin.Client, any, error) {
	name, err := m.capabilityName(capability)
	if err != nil {
		return nil, nil, err
	}

	binary, err := m.discover(capability, name)
	if err != nil {
		return nil, nil, err
	}

	set := pluginSet(capability)
	if len(set) == 0 {
		return nil, nil, fmt.Errorf("%w: %s", errUnsupported, capability)
	}

	// Pre-populate Cmd.Env with the host socket address; go-plugin will append
	// os.Environ() (since SkipHostEnv defaults to false), so this var stays first.
	pluginCmd := exec.Command(binary) //nolint:gosec // binary is discovered from trusted sources
	if m.hostSocket != "" {
		pluginCmd.Env = []string{"SWM_HOST_SOCKET=" + m.hostSocket}
	}

	client := goplugin.NewClient(m.buildClientConfig(ctx, pluginCmd, set))

	rpcClient, err := client.Client()
	if err != nil {
		client.Kill()

		return nil, nil, fmt.Errorf("connecting to plugin %s: %w", binary, err)
	}

	raw, err := rpcClient.Dispense(capability)
	if err != nil {
		client.Kill()

		return nil, nil, fmt.Errorf("dispensing capability %s: %w", capability, err)
	}

	if err := m.validateDeps(ctx, capability, raw); err != nil {
		client.Kill()

		return nil, nil, err
	}

	return client, raw, nil
}

// loadForges launches all configured forge plugins and populates m.forgeClients.
// Must be called with m.mu held.
func (m *Manager) loadForges(ctx context.Context) error {
	for _, name := range m.cfg.Plugins.Forges {
		binary, err := m.discover(capabilityForge, name)
		if err != nil {
			return err
		}

		pluginCmd := exec.Command(binary) //nolint:gosec // binary is discovered from trusted sources
		if m.hostSocket != "" {
			pluginCmd.Env = []string{"SWM_HOST_SOCKET=" + m.hostSocket}
		}

		set := goplugin.PluginSet{capabilityForge: &sdkforge.GRPCPlugin{}}

		client := goplugin.NewClient(m.buildClientConfig(ctx, pluginCmd, set))

		rpcClient, err := client.Client()
		if err != nil {
			client.Kill()

			return fmt.Errorf("connecting to forge plugin %s: %w", binary, err)
		}

		raw, err := rpcClient.Dispense(capabilityForge)
		if err != nil {
			client.Kill()

			return fmt.Errorf("dispensing forge capability from %s: %w", binary, err)
		}

		fc, ok := raw.(pluginv1.ForgeClient)
		if !ok {
			client.Kill()

			return fmt.Errorf("%w: %s", errInvalidForgePlugin, binary)
		}

		info, err := fc.Info(ctx, &pluginv1.Empty{})
		if err != nil {
			client.Kill()

			return fmt.Errorf("calling Info on forge plugin %s: %w", binary, err)
		}

		m.forgeClients = append(m.forgeClients, &forgeEntry{
			client:    client,
			forge:     fc,
			hostnames: info.GetClaimedHosts(),
		})
	}

	return nil
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
	case capabilityPicker:
		if c, ok := raw.(pluginv1.PickerClient); ok {
			resp, err := c.Info(ctx, &pluginv1.Empty{})
			if err != nil {
				return fmt.Errorf("calling Info on picker plugin: %w", err)
			}

			info = resp.GetPluginInfo()
		}
	case capabilityForge:
		if c, ok := raw.(pluginv1.ForgeClient); ok {
			resp, err := c.Info(ctx, &pluginv1.Empty{})
			if err != nil {
				return fmt.Errorf("calling Info on forge plugin: %w", err)
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
	case capabilityForge:
		return goplugin.PluginSet{capabilityForge: &sdkforge.GRPCPlugin{}}
	case capabilityPicker:
		return goplugin.PluginSet{capabilityPicker: &sdkpicker.GRPCPlugin{}}
	case capabilitySession:
		return goplugin.PluginSet{capabilitySession: &sdksession.GRPCPlugin{}}
	case capabilityVCS:
		return goplugin.PluginSet{capabilityVCS: &sdkvcs.GRPCPlugin{}}
	default:
		return goplugin.PluginSet{}
	}
}
