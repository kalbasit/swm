## 1. Proto: Update VCS service definition

- [x] 1.1 Edit `proto/swm/plugin/v1/vcs.proto` (`proto`): remove `CloneResponse`; add `CloneProgressEvent` message with `oneof event { string progress_line = 1; ProjectID project_id = 2; }`; change `Clone` RPC signature to `rpc Clone(CloneRequest) returns (stream CloneProgressEvent)`
- [x] 1.2 Run `task proto` to regenerate `proto/swm/plugin/v1/vcs.pb.go` and `proto/swm/plugin/v1/vcs_grpc.pb.go` (`proto`)

## 2. VCS Plugin: Implement streaming Clone

- [x] 2.1 Write failing test in `plugins/vcs-git/internal/vcs/git_test.go` (`plugins/vcs-git`): test that `Clone` streams at least one `progress_line` event and a terminal `project_id` event, with separate cases for already-cloned (`AlreadyExists`) and git failure (`Internal`)
- [x] 2.2 Update `plugins/vcs-git/internal/vcs/git.go` (`plugins/vcs-git`): change `Clone` to the server-streaming signature; pass `--progress` to git; switch from `cmd.Output()` to `cmd.StderrPipe()` with a `bufio.Scanner` splitting on `\r` and `\n`; stream each segment as `CloneProgressEvent{ProgressLine: seg}`; on success resolve the `ProjectID` and send a terminal `CloneProgressEvent{ProjectId: id}` before returning nil

## 3. Host CLI: Consume streaming Clone

- [x] 3.1 Write failing test in `cmd/swm/internal/cli/clone_test.go` (`cmd/swm`): update `stubVCS.Clone` to match the new `grpc.ServerStreamingClient[pluginv1.CloneProgressEvent]` return type; add a test case asserting progress lines are written to stderr; update existing success/error test cases for the streaming interface
- [x] 3.2 Update `cmd/swm/internal/cli/clone.go` (`cmd/swm`): replace the unary `vcs.Clone()` call with a streaming call; loop over stream events writing `progress_line` events to `cmd.ErrOrStderr()`; capture the `project_id` from the terminal event for use in the post-clone hook and final "cloned to" message

## 4. Nix: Update vendor hashes

- [x] 4.1 Run `task update-nix-vendor-hashes` (`nix`) to patch the `vendorHash` in all five Nix packages (`swm`, `swm-plugin-forge-github`, `swm-plugin-picker-fzf`, `swm-plugin-session-tmux`, `swm-plugin-vcs-git`) after the proto regeneration

## 5. Verification

- [x] 5.1 Run `task fmt lint test` across all modules and confirm all exit 0 (`all modules`)
