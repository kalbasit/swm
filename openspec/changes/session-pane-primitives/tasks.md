## 1. Proto contract

- [ ] 1.1 (`proto`) Add `Pane`, `OpenPaneRequest`, `ListPanesRequest`,
      `SendTextRequest`, and `ClosePaneRequest` messages plus the four RPCs to
      `proto/swm/plugin/v1/session.proto`. Document `pane_id` opacity, the
      best-effort descriptive fields, the focused-pane hazard, and `delay_ms`'s
      relationship to `pane_cmd_delay`.
- [ ] 1.2 (`proto`) Run `task proto:gen` and commit the regenerated
      `session.pb.go` / `session_grpc.pb.go`. Verify `task proto:lint` and
      `task proto:build` pass.
- [ ] 1.3 (root) Run `task update-nix-vendor-hashes` — required after any
      `proto/` change. Downstream Nix packages to rebuild:
      `swm-plugin-session-tmux`, `swm-test-faketmux`, `swm`, `swm-full`.
      No `sdk/go` change is needed: `session.Plugin` is a type alias for
      `pluginv1.SessionServer`, so the new methods flow through.

## 2. Shell quoting helper

- [ ] 2.1 (`plugins/session-tmux`) Write tests for a new `internal/shellquote`
      package covering: bare word unchanged, empty string, spaces, embedded
      single quotes, shell metacharacters, and argv joining.
- [ ] 2.2 (`plugins/session-tmux`) Move `shellQuote` out of `internal/layout`
      into `internal/shellquote` as `Arg`, add `Argv`, and update
      `layout.cmdString` to call it. Existing layout tests must still pass
      unchanged.

## 3. faketmux support

- [ ] 3.1 (`plugins/session-tmux`) Extend `internal/session/testdata/faketmux`
      to handle `new-window` (mint a `%N` pane ID, print it, record a pane row
      against the socket, honour a failure-injection env var) and `list-panes`
      (emit the recorded rows). Keep every existing behaviour intact.

## 4. ListPanes and the shared pane query

- [ ] 4.1 (`plugins/session-tmux`) Write tests: all panes across two live
      workspaces; filtered by workspace; filtered by pane group; focus flag
      derived from attached/active columns; unknown workspace → `NOT_FOUND`;
      dead socket contributes nothing; deterministic ordering.
- [ ] 4.2 (`plugins/session-tmux`) Extract the live-socket scan out of
      `ListWorkspaces` into a shared helper and implement `panes(ctx, sock)`
      over `list-panes -a -F`, plus `Tmux.ListPanes` on top of both.

## 5. OpenPane

- [ ] 5.1 (`plugins/session-tmux`) Write tests: argv with a spaced element is
      quoted, not split; empty argv starts a shell; `cwd` becomes `-c`; env is
      emitted in sorted order; pane group targeted exactly; missing
      workspace/pane-group → `INVALID_ARGUMENT` with no tmux invocation;
      nonexistent pane group → `NOT_FOUND`.
- [ ] 5.2 (`plugins/session-tmux`) Implement `Tmux.OpenPane`.

## 6. SendText

- [ ] 6.1 (`plugins/session-tmux`) Write tests: literal text plus separate
      submit; text that looks like a key name; leading-dash text; submit-only;
      nothing-to-send → `INVALID_ARGUMENT`; unknown pane → `NOT_FOUND`; focused
      pane refused with `FAILED_PRECONDITION` naming `allow_focused`; focused
      pane delivered when `allow_focused` is set; unattached session not treated
      as focused; `delay_ms` observed before delivery; cancelled context during
      the delay returns the context error.
- [ ] 6.2 (`plugins/session-tmux`) Implement `Tmux.SendText` using
      `send-keys -l --` for the text and a separate `send-keys Enter` for
      submit, gated on the shared pane query.

## 7. ClosePane

- [ ] 7.1 (`plugins/session-tmux`) Write tests: live pane killed; already-gone
      pane succeeds; missing identifiers → `INVALID_ARGUMENT`.
- [ ] 7.2 (`plugins/session-tmux`) Implement `Tmux.ClosePane`, generalising
      `isKillPaneNotFound` into a shared target-not-found predicate also used by
      `killOriginPane`.

## 8. Documentation and verification

- [ ] 8.1 (`plugins/session-tmux`) Document the four RPCs in the plugin README,
      including the focused-pane refusal and the `allow_focused` opt-out.
- [ ] 8.2 (root) Run `task fmt`, `task lint`, and `task test`; all three must
      exit 0.
