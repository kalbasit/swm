# swm v2 — Technical Design Document

**Status:** Draft v1
**Author:** kalbasit (with Claude)
**Date:** May 2026

---

## 1. Background and motivation

`swm` (Story-based Workflow Manager) is a CLI that organises code into per-story git worktrees and launches isolated terminal-multiplexer environments for each story. The original implementation (Go, ~2017–2020, at `github.com/kalbasit/swm`) works but is tightly coupled to tmux, fzf, git, and GitHub. Adding zellij, jujutsu, or GitLab support today would require forking and rewriting core packages.

This document describes a rewrite of swm as a **plugin-host** architecture: a small core that owns the story/worktree/filesystem model, and external plugin binaries that provide all integrations with terminal multiplexers, VCS tools, code hosts, pickers, and lifecycle hooks.

### Goals

- Preserve the user-visible workflow of the existing tool (story = unit of work, workspace per story, project per worktree)
- Make every external integration a swappable plugin
- Allow plugins to be written in any language with gRPC support (Go, Rust, Python, etc.)
- Ship a useful default install (git + tmux + github + fzf) without forcing those choices
- Keep day-one Go developers productive; the SDK should make trivial plugins trivial

### Non-goals

- Backwards-compatible config or on-disk format with swm v1 (we expect a migration tool, not transparent compatibility)
- Network-distributed operation (swm is a local CLI; plugins are local subprocesses)
- A GUI

---

## 2. Concepts and terminology

| Term           | Definition                                                                                                                                                                                                                                                     |
| -------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Code root**  | The top directory swm manages (was `$CODE_HOME`, e.g. `~/code`). Contains `repositories/` and `stories/`.                                                                                                                                                      |
| **Project**    | A single source repository, identified by a host-derived path tuple: `(host, segments...)`. For git: `("github.com", ["kalbasit", "swm"])`. For keybase: `("keybase", ["team", "stowix.infra", "fly-secrets"])`. The on-disk layout always mirrors this tuple. |
| **Story**      | A named unit of work. Has a name, a branch name, a creation timestamp, and an associated set of project worktrees.                                                                                                                                             |
| **Repository** | The canonical clone of a project on disk at `<code-root>/repositories/<project-id>`.                                                                                                                                                                           |
| **Worktree**   | A per-story checkout of a project at `<code-root>/stories/<story>/<project-id>`. For git, this is a `git worktree`. For jj, the plugin chooses.                                                                                                                |
| **Workspace**  | swm's abstraction for an isolated multiplexer container. Maps to one story. The session plugin chooses the native primitive (tmux: socket; zellij: session; screen: socket).                                                                                   |
| **Pane group** | swm's abstraction for a project's working environment inside a workspace. Maps to one project's worktree. (tmux: session; zellij: tab; etc.)                                                                                                                   |
| **Plugin**     | An external executable that implements one or more swm capability interfaces over gRPC.                                                                                                                                                                        |
| **Capability** | A typed interface a plugin can implement: `session`, `vcs`, `forge`, `picker`, `hook`.                                                                                                                                                                         |
| **Hook**       | A lifecycle event (`pre-worktree-create`, `post-story-create`, etc.). Hook plugins are a lighter-weight subprocess flavour — see §6.6.                                                                                                                         |

---

## 3. High-level architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                         swm CLI (host)                          │
│  ┌───────────────────────────────────────────────────────────┐  │
│  │  cmd/  (cobra commands: story, project, workspace, ...)   │  │
│  └──────────────────────────┬────────────────────────────────┘  │
│                             │                                   │
│  ┌──────────────────────────▼────────────────────────────────┐  │
│  │  core/  domain logic                                      │  │
│  │  - story store    - project store   - layout resolver     │  │
│  │  - orchestrator (wires capabilities together)             │  │
│  └──────────────────────────┬────────────────────────────────┘  │
│                             │                                   │
│  ┌──────────────────────────▼────────────────────────────────┐  │
│  │  plugin/  plugin manager (HashiCorp go-plugin client)     │  │
│  │  - discovery   - lifecycle   - capability registry        │  │
│  │  - host services (callbacks from plugins back to swm)     │  │
│  └──────────────────────────┬────────────────────────────────┘  │
└─────────────────────────────┼───────────────────────────────────┘
                              │ gRPC over local socket
        ┌─────────────────────┼─────────────────────┐
        │                     │                     │
   ┌────▼─────┐         ┌─────▼────┐          ┌─────▼─────┐
   │ session  │         │   vcs    │   ...    │   forge   │
   │  (tmux)  │         │  (git)   │          │ (github)  │
   └──────────┘         └──────────┘          └───────────┘
```

The host owns: the on-disk path layout (URL-derived, never plugin-chosen), the story JSON files, the plugin lifecycle, the CLI surface. The plugins own: how to talk to tmux/git/github/etc.

---

## 4. Repository layout (monorepo)

```
swm/                              # github.com/kalbasit/swm
├── go.work                       # Go workspaces — one module per buildable artifact
├── README.md
├── docs/
│   ├── tdd/
│   │   ├── 00001-plugin-host-architecture.md    # this file
│   │   └── ...                                  # future TDDs
│   ├── plugin-author-guide.md
│   └── proto-versioning.md
├── proto/
│   └── swm/plugin/v1/            # versioned protobuf definitions
│       ├── common.proto
│       ├── session.proto
│       ├── vcs.proto
│       ├── forge.proto
│       ├── picker.proto
│       └── host.proto            # host services plugins can call back into
├── sdk/
│   ├── go/                       # Go SDK — wraps go-plugin boilerplate
│   │   ├── go.mod
│   │   ├── session/              # base types + serve helper for session plugins
│   │   ├── vcs/
│   │   └── ...
│   └── (future: sdk/rust, sdk/python — out of scope for v2.0)
├── cmd/swm/                      # the swm binary
│   ├── go.mod
│   ├── main.go
│   └── internal/
│       ├── cli/                  # cobra commands
│       ├── core/                 # domain (story, project, layout)
│       ├── pluginmgr/            # discovery, lifecycle, registry
│       └── hostsvc/              # implementations of host services
└── plugins/                      # bundled plugins, each its own module
    ├── session-tmux/
    │   ├── go.mod
    │   ├── main.go
    │   └── README.md
    ├── vcs-git/
    │   ├── go.mod
    │   └── main.go
    ├── forge-github/
    └── picker-fzf/
```

Each plugin is an independent Go module with its own `go.mod`, releaseable independently with its own version tag (`plugins/session-tmux/v0.2.1`). The SDK and proto packages are imported by every plugin; using Go workspaces (`go.work`) keeps local development frictionless. CI builds and tags each plugin independently. Users who install via package managers get `swm`, `swm-plugin-session-tmux`, `swm-plugin-vcs-git`, etc. as separate artefacts — see §6.2 for the binary naming convention.

The monorepo gives us:

- One PR can touch a proto + the host + every plugin that uses it (atomic cross-cutting changes)
- Per-plugin tags and changelogs (plugins evolve at their own pace)
- One CI config, one issue tracker

---

## 5. Domain model and on-disk layout

### 5.1 Filesystem layout

The on-disk layout is a **host invariant**, not a plugin choice. swm derives paths from the remote URL the same way Go's pre-modules `go get` derived `$GOPATH/src/<host>/<owner>/<repo>` — the layout tells users at a glance where every repository came from.

```
$CODE_ROOT/                       # e.g. ~/code
├── repositories/
│   └── <host>/<seg1>/<seg2>/.../<segN>/   # canonical clone
│       └── .git/                          # (or VCS-equivalent)
└── stories/
    └── <story-name>/
        └── <host>/<seg1>/<seg2>/.../<segN>/   # worktree
            └── .git                            # file pointing at canonical
```

Examples:

| Remote URL                                | Project ID                                           | On-disk path                                                 |
| ----------------------------------------- | ---------------------------------------------------- | ------------------------------------------------------------ |
| `git@github.com:kalbasit/swm.git`         | `("github.com", ["kalbasit","swm"])`                 | `~/code/repositories/github.com/kalbasit/swm/`               |
| `https://gitlab.com/foo/bar/baz.git`      | `("gitlab.com", ["foo","bar","baz"])`                | `~/code/repositories/gitlab.com/foo/bar/baz/`                |
| `keybase://team/stowix.infra/fly-secrets` | `("keybase", ["team","stowix.infra","fly-secrets"])` | `~/code/repositories/keybase/team/stowix.infra/fly-secrets/` |

The VCS plugin's role in path resolution is limited to `ParseRemoteURL(url) → (host, segments[])`. The host composes the rest. The plugin cannot choose to flatten, hash, or rearrange — preserving the user-visible discoverability that's been part of swm's design since day one.

Project listing (e.g. for the picker) is therefore a **host** operation: walk `repositories/`, treat any directory containing a marker (`.git`, `.jj`, `.hg`, `.swm-project`) as a project. The VCS plugin advertises which markers it recognises via `Info()`.

### 5.2 State storage

Stories as JSON files at `$XDG_DATA_HOME/swm/stories/<name>.json` (your call from earlier). Schema:

```json
{
  "name": "STORY-123",
  "branch_name": "feat/STORY-123",
  "created_at": "2026-05-15T10:30:00Z",
  "vcs": "git",
  "projects": [
    {
      "host": "github.com",
      "segments": ["kalbasit", "swm"],
      "vcs": "git",
      "attached_at": "2026-05-15T10:35:00Z"
    }
  ],
  "metadata": {}
}
```

`projects[]` records which projects are explicitly attached to this story. Worktree paths are derived (not stored) — `$CODE_ROOT/stories/<name>/<host>/<seg1>/.../<segN>`.

`metadata` is a free-form object plugins can use for their own data, namespaced by plugin name (`metadata["forge-github"] = {"pr_url": "..."}`). The host never reads it.

A default story file always exists at `$XDG_DATA_HOME/swm/stories/_default.json`, created on `swm init` — this represents "no story" / the working state on the canonical repository clone.

### 5.3 Config

`$XDG_CONFIG_HOME/swm/config.toml`:

```toml
code_root = "~/code"
default_story = "_default"

[plugins]
session = "tmux"
vcs = "git"
picker = "fzf"
forges = ["github"]    # forges are a list — a story may touch repos from multiple hosts

# Per-plugin config is passed through opaquely to the plugin
[plugins.config.session-tmux]
default_window_layout = "vim-and-shell"
attach_on_create = true
# Optional: override layout with a custom command (legacy; prefer session-tmux.toml)
# pane_group_command = "my-layout --socket '{{.TmuxSocket}}' --path '{{.WorktreePath}}'"

[plugins.config.forge-github]
token_path = "~/.github_token"

# Global hooks; per-repo and per-story hooks are discovered automatically. See §6.6.
[hooks]
# (No paths needed here — swm reads $XDG_CONFIG_HOME/swm/hooks/<event>.d/ by default.
#  This section can be used to add additional paths if needed.)
```

TOML (not YAML) because it's less ambiguous for users hand-editing config, and the Go ecosystem has good support (`pelletier/go-toml/v2`).

---

## 6. Plugin system

### 6.1 Mechanism: HashiCorp `go-plugin` over gRPC

For each plugin invocation:

1. Host launches plugin binary as subprocess with a magic env var (`SWM_PLUGIN_MAGIC_COOKIE`) — without this the binary refuses to start in plugin mode, preventing accidental direct execution
2. Plugin starts a gRPC server on a Unix socket in `$XDG_RUNTIME_DIR/swm/plugins/<pid>.sock` (Unix only; Windows uses named pipes or local TCP)
3. Plugin prints handshake line: `1|<proto-version>|unix|<socket-path>|grpc\n` then closes stdout
4. Host parses handshake, dials the socket, performs capability negotiation
5. Host makes typed calls; plugin can make typed callbacks (host services)
6. On host exit, plugin process is killed via SIGTERM then SIGKILL (go-plugin handles this)

Plugins are launched **on first use within a command**, not at startup, to keep CLI latency low. A plugin pool keeps started plugins alive for the duration of a single CLI invocation.

We use **gRPC** (not the older net/rpc) for: streaming RPCs (picker streams candidates in, host streams selection out), proto-based schema versioning, and language-agnosticism.

### 6.2 Discovery and binary naming

**Binary naming convention:** every plugin binary is named `swm-plugin-<capability>-<name>`. Examples: `swm-plugin-session-tmux`, `swm-plugin-vcs-git`, `swm-plugin-forge-github`, `swm-plugin-picker-fzf`. The `-plugin-` segment keeps the swm namespace unambiguous in `$PATH` (so a user's personal `swm-vcs-helper` wrapper script doesn't collide with anything swm looks for) and gives clean tab-completion behaviour.

The monorepo directory `plugins/session-tmux/` produces a binary `swm-plugin-session-tmux`.

**Discovery order** (first match wins per capability):

1. **Explicit config** — paths listed in `config.toml` under `[plugins.paths]`
2. **XDG plugins dir** — `$XDG_DATA_HOME/swm/plugins/<name>/swm-plugin-<capability>-<name>` (built binary, typically populated by `swm plugin install`)
3. **PATH** — executables matching `swm-plugin-<capability>-<name>`

PATH lookup is the primary mechanism — it's how git, kubectl, and gh do it, and users understand it. The XDG dir is for users who installed via `swm plugin install <git-url>`, which clones + builds + installs into that directory. We do **not** auto-build at swm startup — see §6.7.

### 6.3 Capability interfaces

Each plugin advertises one or more capabilities at handshake time. The host queries `GetCapabilities()` and routes calls accordingly. A single binary can implement multiple capabilities (e.g. `swm-vcs-git` could conceivably also implement a `forge` capability for self-hosted plain git, though we wouldn't ship it that way).

Capability surfaces (all in `proto/swm/plugin/v1/`, simplified here — actual `.proto` files will have full message definitions):

#### `Session` (terminal multiplexer)

```protobuf
service Session {
  rpc Info(Empty) returns (SessionInfo);                     // name, version, supported features
  rpc OpenWorkspace(OpenWorkspaceRequest) returns (Workspace);  // create+attach OR attach if exists
  rpc CloseWorkspace(CloseWorkspaceRequest) returns (Empty);
  rpc ListWorkspaces(Empty) returns (stream Workspace);
  rpc OpenPaneGroup(OpenPaneGroupRequest) returns (PaneGroup);  // for a project in a workspace
  rpc SwitchTo(SwitchToRequest) returns (Empty);             // jump to a pane group
  rpc IsInsideWorkspace(Empty) returns (BoolValue);          // are we currently in a session this plugin owns?
  rpc CurrentContext(Empty) returns (CurrentContextResponse); // current workspace + pane group
}
```

`OpenWorkspaceRequest` carries the story name + the list of project worktree paths. The plugin maps `workspace=story` and `pane-group=project` to its native primitives. For tmux specifically the plugin uses socket-per-workspace, session-per-pane-group (your current model, preserved).

#### `VCS`

```protobuf
service VCS {
  rpc Info(Empty) returns (VCSInfo);                         // includes project_markers, e.g. [".git"]
  rpc Clone(CloneRequest) returns (CloneResponse);           // -> ProjectID
  rpc ParseRemoteURL(ParseRemoteURLRequest) returns (ProjectID);  // url -> (host, segments[])
  rpc CreateWorktree(CreateWorktreeRequest) returns (Empty); // for a story
  rpc RemoveWorktree(RemoveWorktreeRequest) returns (Empty);
  rpc DetectProjectAtPath(DetectAtPathRequest) returns (ProjectID); // pwd -> (host, segments[])
  rpc ListBranches(ListBranchesRequest) returns (stream Branch);
}

message ProjectID {
  string host = 1;            // e.g. "github.com", "keybase"
  repeated string segments = 2; // e.g. ["kalbasit", "swm"]
}
```

The host composes the on-disk path from `(host, segments)` — see §5.1. The plugin does not choose the layout.

#### `Forge`

```protobuf
service Forge {
  rpc Info(Empty) returns (ForgeInfo);                       // hostnames it claims, e.g. ["github.com"]
  rpc ListPullRequests(ListPRsRequest) returns (stream PullRequest);
  rpc CreatePullRequest(CreatePRRequest) returns (PullRequest);
  rpc GetPullRequest(GetPRRequest) returns (PullRequest);
  // (more: issues, CI status, releases — defined incrementally)
}
```

A forge plugin advertises the hostnames it handles. When swm needs a forge operation for `github.com/foo/bar`, it picks the plugin claiming `github.com`. Multiple forge plugins can coexist; conflicts on a hostname are config errors.

#### `Picker`

```protobuf
service Picker {
  rpc Info(Empty) returns (PickerInfo);
  rpc Pick(stream PickItem) returns (stream PickResult);     // bidirectional streaming
}
```

Host streams candidates in (`{key, display, preview?}`), plugin returns the selection(s). Multi-select is supported via streaming responses.

#### No `Editor` capability — session plugin handles it

Earlier drafts had an `Editor` capability. We removed it: launching an editor in a pane is just a shell command, and modelling it as a capability added surface area without enough value. Instead:

- **Default behavior:** the session plugin opens a pane group running `$EDITOR` (or `vim` as fallback) in the first window/pane and a shell in the second. This matches v1's behaviour exactly.
- **Custom layouts:** users place a `session-tmux.toml` file at `<worktree>/.swm/session-tmux.toml` (per-repo) or `$XDG_CONFIG_HOME/swm/session-tmux.toml` (global). The file is a TOML layout config with windows, panes, flex weights, and commands. Template variables (`{{.WorktreePath}}`, `{{.StoryName}}`, `{{.TmuxSocket}}`) are expanded before parsing. A legacy `pane_group_command` escape hatch is also available for users who need to shell out to an external layout tool.

### 6.4 Host services (plugin → host callbacks)

Plugins receive a connection back to the host and can call:

```protobuf
service Host {
  rpc GetConfig(GetConfigRequest) returns (Config);          // scoped to plugin's config namespace
  rpc GetCodeRoot(Empty) returns (PathResponse);
  rpc ListProjects(ListProjectsRequest) returns (stream Project);
  rpc GetCurrentStory(Empty) returns (Story);
  rpc Log(LogRequest) returns (Empty);                       // structured logging into host's logger
  rpc CallCapability(CallCapabilityRequest) returns (CallCapabilityResponse);
}
```

`CallCapability` is how plugin-to-plugin coordination happens. The session plugin doesn't talk to the VCS plugin directly — it asks the host to make the call. The host enforces dependency declarations (§6.5) and can substitute, mock, or log calls. No plugin has gRPC credentials for any other plugin.

Example: when the tmux plugin opens a pane group, it needs the working directory. That's the worktree path, which the VCS plugin knows. Flow:

1. Host calls `tmux.OpenPaneGroup({story: "X", project_id: "Y"})`
2. tmux plugin calls `host.CallCapability({capability: "vcs", method: "WorktreePathFor", args: {story:"X", project:"Y"}})`
3. Host routes to git plugin, returns the path
4. tmux plugin uses it as the session's starting directory

### 6.5 Plugin dependencies

Each plugin's `Info()` response declares dependencies:

```protobuf
message PluginInfo {
  string name = 1;
  string version = 2;
  repeated Capability provides = 3;
  repeated CapabilityDep requires = 4;  // capability name + min version
  repeated string optional = 5;          // capability names that enhance behavior if present
}
```

At startup the host builds a dependency graph and refuses to load a configuration with unsatisfiable deps, printing a clear error: "plugin `session-tmux` requires capability `vcs`, but no vcs plugin is configured."

Optional deps degrade gracefully — `session-tmux` works without a forge plugin, but if one's present it might show PR status in the status bar.

### 6.6 Hooks (the lighter plugin flavour)

Hooks are plain executables, not gRPC plugins. Reason: hooks are the right escape hatch for "I want to run a bash script when X happens," and forcing those scripts to speak gRPC defeats the purpose.

A hook executable is invoked with:

- Environment variables: `SWM_HOOK=pre-worktree-create`, `SWM_STORY=feat-x`, `SWM_PROJECT_HOST=github.com`, `SWM_PROJECT_PATH=kalbasit/swm`, `SWM_WORKTREE_PATH=/home/u/code/stories/feat-x/...`, `SWM_REPO_PATH=/home/u/code/repositories/...`
- Stdin: a JSON document with the same fields plus the full project ID tuple, for richer consumers

**Scoped resolution.** When event E fires for project P in story S, swm runs hooks from three tiers, **in order**, with results composed (not overridden):

1. **Global hooks:** `$XDG_CONFIG_HOME/swm/hooks/<event>.d/*`
2. **Per-repository hooks:** `<canonical-repo-path>/.swm/hooks/<event>.d/*` — lives in `.swm/` alongside `.git/` in the canonical clone (not the worktree), so it survives `git worktree remove` and doesn't pollute working trees
3. **Per-story hooks:** `$XDG_CONFIG_HOME/swm/stories/<story-name>/hooks/<event>.d/*` — rare but useful for "this one story needs special treatment"

Within each tier, files run in lexical order. All applicable tiers run for every event; a user with a global `direnv allow` hook and a per-repo `npm install` hook gets both.

The per-repo `.swm/hooks/` location is quietly powerful: project authors can commit `.swm/hooks/post-worktree-create.d/setup-env` to their repo, giving every contributor a working dev environment on first checkout. Whether to commit `.swm/` or gitignore it is the project author's choice.

**Exit codes:**

- `pre-*` hook returning non-zero **aborts** the operation. The first failing hook within a tier stops that tier; later tiers do not run. swm reports which hook failed and exits non-zero itself.
- `post-*` hook failures are logged but never roll back. swm continues to the next hook.

**Hook events** (initial set):

- `pre-story-create`, `post-story-create`
- `pre-story-remove`, `post-story-remove`
- `pre-worktree-create`, `post-worktree-create`
- `pre-worktree-remove`, `post-worktree-remove`
- `pre-clone`, `post-clone`
- `pre-workspace-open`, `post-workspace-open`

This subsumes v1's `~/.config/swm/hooks/coder/{pre-hook,post-hook}/` mechanism (which only covered worktree creation) with a richer event vocabulary, scoped resolution, and richer arg passing.

### 6.7 `swm plugin install` (no auto-build at startup)

```
swm plugin install github.com/foo/swm-session-zellij
swm plugin install ./local/path/to/plugin
swm plugin list
swm plugin remove <name>
swm plugin upgrade <name>
```

`install` clones (or copies) into `$XDG_DATA_HOME/swm/plugins/<name>/`, reads `swm-plugin.toml` from the plugin root, runs its declared build command (default `go build -o swm-plugin-<capability>-<name> .`), and symlinks the built binary into `$XDG_DATA_HOME/swm/bin/` (which users add to PATH, or `swm` finds via the XDG plugins dir lookup in §6.2).

We do **not** auto-build at `swm` startup because (a) latency, (b) it locks plugins to languages with predictable toolchains, (c) it conflates plugin development with plugin use. Build happens explicitly at install/upgrade time.

`swm-plugin.toml` example:

```toml
[plugin]
name = "session-zellij"
capability = "session"
version = "0.1.0"

[build]
command = ["go", "build", "-o", "swm-plugin-session-zellij", "."]
```

---

## 7. Core orchestration flows

Walking through the main user-facing commands to validate the architecture.

### 7.1 `swm story create <name> [--branch <branch>]`

1. CLI parses args, loads config
2. Core validates: name non-empty, not already existing
3. Run `pre-story-create` hooks
4. Write `$XDG_DATA_HOME/swm/stories/<name>.json`
5. Run `post-story-create` hooks
6. No worktrees created yet (lazy)

### 7.2 `swm clone <url>`

1. CLI calls VCS plugin's `ParseRemoteURL(url)` → project_id
2. Core checks: does `repositories/<project-id>` already exist?
3. Run `pre-clone` hooks
4. Core calls VCS plugin's `Clone(url)` — plugin handles the clone-to-temp-then-rename dance internally
5. Run `post-clone` hooks
6. Project is now available; not attached to any story yet

### 7.3 `swm story attach <project-id-or-url>` (new in v2, the explicit form)

1. Resolve to canonical project_id (via VCS plugin if it's a URL)
2. If repository doesn't exist locally, clone it first (calls 7.2 flow)
3. Run `pre-worktree-create` hooks
4. VCS plugin's `CreateWorktree({story, project_id, branch})`
5. Append project entry to the story JSON
6. Run `post-worktree-create` hooks

### 7.4 `swm workspace open [--story <name>]` (replaces `swm tmux switch-client`)

This is the big one. Flow:

1. CLI resolves story (flag, `$SWM_STORY`, or default)
2. Core asks session plugin: `IsInsideWorkspace()`. If yes, get current workspace name; if it matches the target story, we're already inside and can stream directly into `SwitchTo`. If it doesn't match, we'll need to detach (or open a nested session — plugin's call).
3. Core lists all projects: union of (projects attached to story) + (all repositories on disk)
4. Core calls picker plugin's `Pick(stream of project candidates)` → user selects one
5. If selected project isn't attached to the story yet: run 7.3 flow (lazy creation, preserved from v1)
6. Core calls session plugin's `OpenWorkspace({story, projects: [...]})` if workspace doesn't exist, else `OpenPaneGroup({story, project})` to add to existing
7. Session plugin handles the rest (creating tmux socket, sessions, attaching, etc.)

This flow preserves your v1 behaviour exactly while moving every tool-specific detail behind a plugin.

### 7.5 `swm story remove <name> [--force]`

1. Confirmation prompt unless `--force`
2. Run `pre-story-remove` hooks
3. For each attached project, call VCS plugin's `RemoveWorktree` and run `pre/post-worktree-remove` hooks
4. Tell session plugin to close the workspace if one exists
5. Delete the story JSON
6. Run `post-story-remove` hooks

---

## 8. Versioning and compatibility

- **swm host** versioned via semver, tag `vX.Y.Z` at repo root
- **Each plugin** versioned independently, tags `plugins/<name>/vX.Y.Z`
- **Proto definitions** versioned in directory paths (`proto/swm/plugin/v1/`). When breaking changes are needed, we create `v2/` alongside `v1/`. The host can speak multiple proto versions; plugins declare which they speak in their handshake. We commit to supporting `v1` for at least one full host major version after `v2` ships.
- **SDK** versioned alongside protos. Patch-level SDK changes don't require plugin rebuilds; minor/major do.

Compatibility matrix is published at `docs/compatibility.md` and tested in CI.

---

## 9. Testing strategy

- **Unit tests** in every package (core, pluginmgr, each plugin), `go test ./...`
- **Integration tests** in `tests/integration/` using real plugin binaries against a real temp `$CODE_ROOT`. Run on every push.
- **Plugin contract tests** in `sdk/go/contract/` — a test harness any plugin can import that exercises the full capability surface against the plugin. Ensures third-party plugins conform.
- **End-to-end tests** for the headline flows (clone → story create → workspace open) using a mock session plugin to avoid requiring tmux in CI.

---

## 10. Implementation plan

Suggested phasing — each phase produces a usable artefact.

### Phase 0 — Foundation (week 1)

- Monorepo skeleton, `go.work`, CI scaffolding
- Proto definitions for all six capabilities (compile-only, no impl)
- Go SDK skeleton (one package per capability, with stub `Serve` helpers)
- Document the plugin protocol

### Phase 1 — Vertical slice: git + tmux (weeks 2–3)

- Host core: story store, config loading, plugin manager
- `vcs-git` plugin (Clone, ParseRemoteURL, CreateWorktree, RemoveWorktree)
- `session-tmux` plugin (OpenWorkspace, OpenPaneGroup, SwitchTo, ListWorkspaces)
- `swm story create`, `swm clone`, `swm workspace open`, `swm story remove`
- End-to-end test: real git, real tmux, real filesystem

### Phase 2 — Picker (week 4)

- `picker-fzf` plugin
- Wire picker into `workspace open` flow
- Session plugin's built-in layout engine (`session-tmux.toml`) and `pane_group_command` escape hatch

### Phase 3 — Forges and hooks (week 5)

- `forge-github` plugin (ListPullRequests, CreatePullRequest, GetPullRequest)
- Hook executor in host
- `swm pr list`, `swm pr create`

### Phase 4 — Polish and release (week 6)

- `swm plugin install/list/remove/upgrade`
- Migration tool from v1 (`swm migrate-v1`): reads v1 story JSON files, rewrites to v2 schema
- Shell completion (cobra-generated, same as v1)
- Documentation site
- Release `v2.0.0`

### Stretch / post-v2.0

- `vcs-jj` plugin (jujutsu)
- `session-zellij` plugin
- `forge-gitlab` plugin
- Rust SDK

---

## 11. Open questions

These should be resolved before or during Phase 0:

1. **Plugin sandboxing.** Should plugins run as unprivileged subprocesses with limited filesystem access (e.g. only the code root)? Adds complexity; might be worth deferring.
2. **Plugin install security.** `swm plugin install <git-url>` runs arbitrary build commands from a third-party manifest. We should at least show the build command and prompt before first run, and probably maintain a community registry of vetted plugins long-term.
3. **Concurrent operations.** Multiple swm invocations against the same code root — do we need file locking on story JSONs? Probably yes, via `flock` on the story file during writes.
4. **Telemetry.** None by default. Mention in docs that all data stays local.

---

## 12. Migration from v1

A `swm migrate-v1` command will:

1. Locate `$XDG_DATA_HOME/swm/stories/*.json` (v1 schema)
2. For each, rewrite to v2 schema: add `vcs: "git"`, scan `$CODE_ROOT/stories/<name>/` for project directories and populate `projects[]`, add `created_at` from file mtime if missing
3. Move `~/.config/swm/hooks/coder/{pre-hook,post-hook}/` to `~/.config/swm/hooks/{pre-worktree-create,post-worktree-create}/` and update arg-passing if needed
4. Translate `~/.config/swm/config.yaml` (viper) to `~/.config/swm/config.toml` with default plugin selections (`session = "tmux"`, `vcs = "git"`)

The on-disk repository/worktree layout is unchanged, so no data movement is needed — just metadata.
