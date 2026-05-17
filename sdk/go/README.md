# swm Go SDK

Go SDK for writing swm plugins. Import path: `github.com/kalbasit/swm/sdk/go`.

Plugins are external binaries that the swm host discovers and connects to over gRPC (via [HashiCorp go-plugin](https://github.com/hashicorp/go-plugin)). The SDK wraps the generated protobuf types and the plugin handshake so you can focus on implementing the capability logic.

## Capability interfaces

Each capability surface has its own sub-package with a `Plugin` interface and a `Serve` entry point.

| Sub-package | Import path                              | Capability            |
| ----------- | ---------------------------------------- | --------------------- |
| `session`   | `github.com/kalbasit/swm/sdk/go/session` | Terminal multiplexer  |
| `vcs`       | `github.com/kalbasit/swm/sdk/go/vcs`     | Version control       |
| `forge`     | `github.com/kalbasit/swm/sdk/go/forge`   | Code-hosting platform |
| `picker`    | `github.com/kalbasit/swm/sdk/go/picker`  | Interactive selection |

### Session

```go
import "github.com/kalbasit/swm/sdk/go/session"

type Plugin interface {
    Info(context.Context, *pluginv1.Empty) (*pluginv1.SessionInfo, error)
    OpenWorkspace(context.Context, *pluginv1.OpenWorkspaceRequest) (*pluginv1.Workspace, error)
    CloseWorkspace(context.Context, *pluginv1.CloseWorkspaceRequest) (*pluginv1.Empty, error)
    ListWorkspaces(context.Context, *pluginv1.Empty) ([]*pluginv1.Workspace, error)
    OpenPaneGroup(context.Context, *pluginv1.OpenPaneGroupRequest) (*pluginv1.PaneGroup, error)
    SwitchTo(context.Context, *pluginv1.SwitchToRequest) (*pluginv1.Empty, error)
    IsInsideWorkspace(context.Context, *pluginv1.Empty) (*pluginv1.BoolValue, error)
    CurrentContext(context.Context, *pluginv1.Empty) (*pluginv1.CurrentContextResponse, error)
}
```

### VCS

```go
import "github.com/kalbasit/swm/sdk/go/vcs"

type Plugin interface {
    Info(context.Context, *pluginv1.Empty) (*pluginv1.VCSInfo, error)
    Clone(context.Context, *pluginv1.CloneRequest) (*pluginv1.CloneResponse, error)
    ParseRemoteURL(context.Context, *pluginv1.ParseRemoteURLRequest) (*pluginv1.ProjectID, error)
    CreateWorktree(context.Context, *pluginv1.CreateWorktreeRequest) (*pluginv1.Empty, error)
    RemoveWorktree(context.Context, *pluginv1.RemoveWorktreeRequest) (*pluginv1.Empty, error)
    DetectProjectAtPath(context.Context, *pluginv1.DetectAtPathRequest) (*pluginv1.ProjectID, error)
    ListBranches(context.Context, *pluginv1.ListBranchesRequest, func(*pluginv1.Branch) error) error
}
```

### Forge

```go
import "github.com/kalbasit/swm/sdk/go/forge"

type Plugin interface {
    Info(context.Context, *pluginv1.Empty) (*pluginv1.ForgeInfo, error)
    ListPullRequests(context.Context, *pluginv1.ListPRsRequest, func(*pluginv1.PullRequest) error) error
    CreatePullRequest(context.Context, *pluginv1.CreatePRRequest) (*pluginv1.PullRequest, error)
    GetPullRequest(context.Context, *pluginv1.GetPRRequest) (*pluginv1.PullRequest, error)
}
```

### Picker

```go
import "github.com/kalbasit/swm/sdk/go/picker"

type Plugin interface {
    Info(context.Context, *pluginv1.Empty) (*pluginv1.PickerInfo, error)
    Pick(context.Context, func() (*pluginv1.PickItem, error), func(*pluginv1.PickResult) error) error
}
```

## Registering a plugin

Call `Serve` from `main()` with your implementation:

```go
package main

import (
    "github.com/kalbasit/swm/sdk/go/vcs"
)

func main() {
    if err := vcs.Serve(&myVCSPlugin{}); err != nil {
        panic(err)
    }
}
```

`Serve` blocks until the host closes the connection. It handles the gRPC server setup, the go-plugin handshake, and OS signal handling.

## Declaring capabilities via Info()

Every plugin implements `Info()` to advertise its identity and dependencies. The host validates the dependency graph at startup — missing required capabilities cause a startup error.

```go
func (p *myForgePlugin) Info(_ context.Context, _ *pluginv1.Empty) (*pluginv1.ForgeInfo, error) {
    return &pluginv1.ForgeInfo{
        PluginInfo: &pluginv1.PluginInfo{
            Name:    "my-forge",
            Version: version, // set via -ldflags at build time
            // Capabilities this plugin requires the host to have loaded.
            Requires: []*pluginv1.CapabilityDep{
                {Capability: pluginv1.Capability_VCS},
            },
            // Optional takes capability name strings (not CapabilityDep structs).
            // The host wires the capability if present; the plugin handles absence.
            Optional: []string{"picker"},
        },
        // Hostnames this forge plugin handles (for URL routing).
        ClaimedHosts: []string{"github.example.com"},
    }, nil
}
```

**`Requires`** — the host aborts startup if any required capability is missing.
**`Optional`** — the host wires the capability if available; the plugin must handle absence gracefully.

## Protobuf types

All request/response types live in `github.com/kalbasit/swm/proto` (module `github.com/kalbasit/swm/proto`), package `pluginv1`. The SDK re-exports common types; import the proto module directly if you need lower-level access.

## Example: minimal VCS plugin skeleton

```go
package main

import (
    "context"

    pluginv1 "github.com/kalbasit/swm/proto/swm/plugin/v1"
    "github.com/kalbasit/swm/sdk/go/vcs"
)

var version = "dev"

type myVCS struct{}

func (v *myVCS) Info(_ context.Context, _ *pluginv1.Empty) (*pluginv1.VCSInfo, error) {
    return &pluginv1.VCSInfo{
        PluginInfo: &pluginv1.PluginInfo{
            Name:    "my-vcs",
            Version: version,
        },
        ProjectMarkers: []string{".git"},
    }, nil
}

// ... implement remaining methods ...

func main() {
    if err := vcs.Serve(&myVCS{}); err != nil {
        panic(err)
    }
}
```

Build the binary and name it `swm-plugin-vcs-<name>`, then place it on `$PATH` or configure its path explicitly in `config.toml`.
