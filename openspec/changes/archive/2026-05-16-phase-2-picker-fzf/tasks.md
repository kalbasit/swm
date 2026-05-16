## 1. SDK — Picker GRPCPlugin (sdk/go)

- [x] 1.1 Replace the stub `Serve` in `sdk/go/picker/plugin.go` with a real `GRPCPlugin` struct (embedding `goplugin.NetRPCUnsupportedPlugin`, with `GRPCClient` returning `pluginv1.NewPickerClient(conn)` and `GRPCServer` registering `pluginv1.RegisterPickerServer`); update `Serve(impl Plugin)` to call `goplugin.Serve` with the registered plugin map; add `NewClient(conn *grpc.ClientConn) Plugin` constructor; remove `ErrNotImplemented` (`sdk/go` module)
- [x] 1.2 Write unit tests in `sdk/go/picker/plugin_test.go` verifying the handshake config fields match `sdk/go/handshake` constants (`sdk/go` module)
- [x] 1.3 Run `task sdk:test` → confirm exits 0

## 2. Plugin Manager — Wire Picker Capability (cmd/swm)

- [x] 2.1 Add `sdkpicker "github.com/kalbasit/swm/sdk/go/picker"` import to `cmd/swm/internal/pluginmgr/manager.go`; update `pluginSet` to handle `capabilityPicker` → `goplugin.PluginSet{capabilityPicker: &sdkpicker.GRPCPlugin{}}`; update `validateDeps` to handle `capabilityPicker` (call `Info()` and validate `plugin_info`) (`cmd/swm` module)
- [x] 2.2 Run `task swm:test` → confirm exits 0

## 3. picker-fzf Plugin (plugins/picker-fzf)

- [x] 3.1 Add deps to `plugins/picker-fzf/go.mod`: `github.com/kalbasit/swm/sdk/go`, `github.com/kalbasit/swm/proto`, `google.golang.org/grpc`, `github.com/hashicorp/go-plugin`; run `go mod tidy`; add `replace` directives for local modules; add `plugins/picker-fzf` to `go.work` (`plugins/picker-fzf` module)
- [x] 3.2 Create `plugins/picker-fzf/internal/picker/fzf.go`: `Fzf` struct implementing `pluginv1.PickerServer`; `New() *Fzf` constructor; implement `Info` returning `PickerInfo{plugin_info: {name: "fzf", version: buildVersion}}` (`plugins/picker-fzf` module)
- [x] 3.3 Implement `Fzf.Pick(stream grpc.BidiStreamingServer[PickItem, PickResult]) error`: (1) receive all `PickItem` messages from the host stream, accumulating keys and display strings; (2) open `/dev/tty` for both stdin and stdout — if `/dev/tty` cannot be opened, return gRPC `FailedPrecondition`; (3) pipe accumulated candidates as `<key>\t<display>\n` to fzf's stdin with `--with-nth=2` so display is shown; (4) after fzf exits, parse its stdout for the selected key; (5) if fzf exits non-zero (user cancelled), return gRPC `Aborted`; (6) stream one `PickResult{key: selectedKey}` and close (`plugins/picker-fzf` module)
- [x] 3.4 Update `plugins/picker-fzf/main.go`: import `sdkpicker` and `internal/picker`; call `sdkpicker.Serve(&picker.Fzf{})` instead of `fmt.Println` stub; set `buildVersion` via ldflags (`plugins/picker-fzf` module)
- [x] 3.5 Create `plugins/picker-fzf/internal/picker/fzf_test.go`: write tests using a fake `fzf` binary compiled from `testdata/fakefzf/`; cover: successful single selection, user cancellation (fake fzf exits 1), no-TTY error path (`plugins/picker-fzf` module)
- [x] 3.6 Run `task picker-fzf:test` → confirm exits 0

## 4. session-tmux — pane_group_command (plugins/session-tmux)

- [x] 4.1 Add `hostClient pluginv1.HostClient` field to `Tmux` struct; update `New()` to read `SWM_HOST_SOCKET` env var, dial the Unix socket via `grpc.NewClient("unix://"+socketPath, grpc.WithTransportCredentials(insecure.NewCredentials()))`, and store `pluginv1.NewHostClient(conn)` on the struct; store the `*grpc.ClientConn` for cleanup too; add `Close() error` method to `Tmux` that closes the gRPC connection (`plugins/session-tmux` module)
- [x] 4.2 Update `OpenPaneGroup`: if `hostClient` is non-nil, call `hostClient.GetConfig(ctx, &pluginv1.GetConfigRequest{PluginName: "tmux"})`; unmarshal the returned TOML bytes into a struct with a `PaneGroupCommand string \`toml:"pane_group_command"\`` field; if `pane_group_command` is set, substitute `{{worktree_path}}`, `{{story_name}}`, and `{{project_id}}` via `strings.ReplaceAll` and pass the result as the `-c` argument to `tmux new-session`; if not set, default to `$SHELL` (`plugins/session-tmux` module)
- [x] 4.3 Update `plugins/session-tmux/main.go`: call `t.Close()` on process exit (defer after `session.New()` succeeds) (`plugins/session-tmux` module)
- [x] 4.4 Add tests in `tmux_test.go` for `OpenPaneGroup` with `pane_group_command` set: inject a fake host client that returns TOML config with `pane_group_command = "laio start --config {{worktree_path}}/.swm/laio.yaml"` and verify the faketmux receives the substituted command (`plugins/session-tmux` module)
- [x] 4.5 Run `task session-tmux:test` → confirm exits 0

## 5. workspace open — Picker Integration and Lazy Worktree (cmd/swm)

- [x] 5.1 Refactor `open.go` to extract a `listAllProjects(codeRoot string) ([]*pluginv1.ProjectID, error)` helper that walks `<codeRoot>/repositories/` and returns a `ProjectID` for each discovered repository directory (same logic as `hostsvc.ListProjects`) (`cmd/swm` module)
- [x] 5.2 In `open.go` `RunE`, after resolving the story, attempt `mgr.Get(ctx, "picker")` — if it succeeds and the returned client is a `pluginv1.PickerClient`, assign it; if `mgr.Get` returns `errNoPickerPlugin` or a type-assertion failure, leave the picker nil and fall through to the Phase 1 path (`cmd/swm` module)
- [x] 5.3 Implement picker path in `RunE`: (1) call `listAllProjects(cfg.CodeRoot)` and merge with already-attached projects to form a deduplicated candidate list; (2) open a `picker.Pick` stream; (3) send each candidate as a `PickItem{key: projectIDString, display: projectIDString}`; (4) close the send side; (5) receive one `PickResult` — if the stream returns `Aborted`, print nothing and return nil; if any other error, return it wrapped (`cmd/swm` module)
- [x] 5.4 After receiving a `PickResult`, derive the `ProjectID` from the key; check if the project is already attached to the story; if not: call `vcs.CreateWorktree` for the project (using the story's branch name and the resolver-derived paths), then call `store.Update` to attach the project to the story (`cmd/swm` module)
- [x] 5.5 After lazy-attach (or if already attached), call `sess.OpenPaneGroup(ctx, &pluginv1.OpenPaneGroupRequest{WorkspaceId: ws.WorkspaceId, ProjectId: pid, WorktreePath: worktreePath})` for the selected project; then call `sess.SwitchTo` to bring it into focus (`cmd/swm` module)
- [x] 5.6 Update `open_test.go`: add test cases for picker path (stub picker returning a fixed `PickResult`), lazy worktree creation path (stub vcs + store), and picker-absent fallback (Phase 1 path still works) (`cmd/swm` module)

## 6. Integration and Final Verification

- [x] 6.1 Add `plugins/picker-fzf` binary compilation to `cmd/swm/tests/integration/integration_test.go` `TestMain`; write `TestWorkspaceOpenWithPicker` using a fake fzf binary that echoes the first candidate, verifying `OpenPaneGroup` is called with correct worktree path (`cmd/swm` module)
- [x] 6.2 Run `task fmt` → confirm exits 0
- [x] 6.3 Run `task lint` → confirm exits 0
- [x] 6.4 Run `task test` → confirm exits 0
