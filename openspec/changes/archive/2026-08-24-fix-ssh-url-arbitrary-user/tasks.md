## 1. Failing tests (red)

- [x] 1.1 Convert the four `TestParseRemoteURL_*` cases in `plugins/vcs-git/internal/vcs/git_test.go` into one table-driven test, preserving each existing input and expected `ProjectID` verbatim; confirm it still passes before adding new rows.
- [x] 1.2 Add a row for `org-1470@git.entreprise.com:Team/Project.git` expecting `{git.entreprise.com, [Team, Project]}`; confirm it fails with `InvalidArgument`.
- [x] 1.3 Add a row for the userless scp form `git.entreprise.com:Team/Project.git` expecting the same `ProjectID`.
- [x] 1.4 Add rows for `ssh://git@example.com/owner/repo.git` and `ssh://git@example.com:2222/owner/repo.git`, both expecting `{example.com, [owner, repo]}`; the second must assert the host contains no `:`.
- [x] 1.5 Add a row for an absolute local path (`/srv/repos/acme/widget.git`) expecting `{localhost, [srv, repos, acme, widget]}` (currently untested, guards the branch reorder).
- [x] 1.6 Add error rows for `https://gitlab.com/foo//baz.git` and `https://gitlab.com/foo/bar/`, each expecting `codes.InvalidArgument`.
- [x] 1.7 Assert the gRPC status *code* is `InvalidArgument` on every error row, not just that an error is non-nil.

## 2. Implementation (green)

- [x] 2.1 Replace `sshURLRe` with `^(?:[^/@]+@)?([^/:]+):(.+)$` and rename it to reflect that it matches scp-style URLs with any or no user; update its doc comment.
- [x] 2.2 In `parseURL`, move the absolute-local-path branch ahead of the scp branch.
- [x] 2.3 Guard the scp branch on `!strings.Contains(raw, "://")` so schemed URLs cannot reach it.
- [x] 2.4 In the scp branch, discard the user, `strings.TrimSuffix` the `.git` from the path capture, and split into segments.
- [x] 2.5 Switch the schemed-URL branch from `u.Host` to `u.Hostname()`, keeping the existing empty-host `InvalidArgument` guard.
- [x] 2.6 Add a single empty-segment guard covering all branches, returning `InvalidArgument`, and remove the now-redundant per-branch `path == ""` checks.
- [x] 2.7 Run `task test` and confirm every row from section 1 passes.

## 3. Cleanup (refactor)

- [x] 3.1 Re-read `parseURL` end to end and confirm the four branches read as one ordered dispatch with no duplicated suffix/segment handling.
- [x] 3.2 Verify no `panic`, `log.Fatal`, or `os.Exit` was introduced and that any `//nolint` carries an explanatory comment.

## 4. Verification

- [x] 4.1 Run `task fmt` and confirm it exits 0 with no residual diff.
- [x] 4.2 Run `task lint` and confirm it exits 0.
- [x] 4.3 Run `task test` across the workspace (not just the `vcs-git` module) and confirm it exits 0.
- [x] 4.4 Confirm `proto/` is untouched, so `task update-nix-vendor-hashes` is not required.
- [x] 4.5 Build the plugin and manually confirm `swm clone --help`-level wiring is unaffected by running `ParseRemoteURL` against the reported URL via the test suite.
