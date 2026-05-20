// Command swm is the Story-based Workflow Manager host.
package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/adrg/xdg"

	"github.com/kalbasit/swm/cmd/swm/internal/cli"
	"github.com/kalbasit/swm/cmd/swm/internal/cli/workspace"
	"github.com/kalbasit/swm/cmd/swm/internal/config"
	"github.com/kalbasit/swm/cmd/swm/internal/core/layout"
	"github.com/kalbasit/swm/cmd/swm/internal/core/story"
	"github.com/kalbasit/swm/cmd/swm/internal/hostsvc"
	"github.com/kalbasit/swm/cmd/swm/internal/pluginmgr"
)

var version = "v2.0.0-dev"

func main() {
	cfgPath := config.ResolveConfigPath(os.Getenv("SWM_CONFIG"), xdg.ConfigHome)

	cfg, err := config.Load(cfgPath)
	if err != nil && !errors.Is(err, config.ErrConfigNotFound) {
		fmt.Fprintf(os.Stderr, "swm: loading config: %v\n", err)
		os.Exit(1)
	}

	if cfg == nil {
		cfg = config.Defaults()
	}

	storiesDir := filepath.Join(xdg.DataHome, "swm", "stories")
	store := story.NewJSONStore(storiesDir)
	resolver := layout.NewResolver(cfg.CodeRoot, cfg.DefaultStory)

	hostSrv, err := hostsvc.NewServer(cfg, resolver, store)
	if err != nil {
		fmt.Fprintf(os.Stderr, "swm: starting host service: %v\n", err)
		os.Exit(1)
	}
	defer hostSrv.Stop()

	mgr := pluginmgr.New(cfg, hostSrv.SocketPath())
	defer mgr.Close() //nolint:errcheck // best-effort close on exit

	root := cli.NewRootCmd(cfgPath, cfg, mgr, store, resolver, workspace.WithProjectLister(hostSrv))
	root.Version = version

	if err := root.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
