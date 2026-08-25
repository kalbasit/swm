## Why

`swm clone org-1470@git.entreprise.com:Team/Project.git` fails with `InvalidArgument: cannot parse remote URL`. The `vcs-git` plugin's scp-style matcher hardcodes the `git` user (`^git@([^:]+):(.+?)(?:\.git)?$`), so any remote whose SSH user is not literally `git` — self-hosted Forgejo (`forgejo@`), Azure DevOps org aliases (`org-1470@`), per-deploy-key SSH aliases — falls through to `url.Parse`, which cannot parse scp syntax and yields an empty host. This makes swm unusable against several common enterprise git hosts.

Two adjacent defects in the same function surface while fixing it: a URL with an explicit port (`ssh://git@host:2222/a/b`) returns `host:2222` as the `ProjectID` host, producing a directory name containing a colon; and a trailing or doubled slash (`https://host/a/b/`) yields an empty path segment, producing an empty directory component. Both corrupt the host-composed on-disk layout (TDD §5.1).

## What Changes

- Replace the `git`-only scp-style regex with one accepting any (or no) SSH user: `[user@]host:path`. `org-1470@git.entreprise.com:Team/Project.git` and `git.entreprise.com:Team/Project.git` both resolve to `ProjectID{host: "git.entreprise.com", segments: ["Team", "Project"]}`.
- Disambiguate scp syntax from schemed URLs by checking for `://` first, so `https://` / `ssh://` / `file://` keep their existing path and are never treated as scp.
- Strip the port from the host for schemed URLs, so `ssh://git@host:2222/a/b` yields host `host`.
- Reject URLs that produce an empty path segment with `InvalidArgument` instead of returning a `ProjectID` containing an empty segment.
- Existing behaviour is preserved for `git@`, HTTPS, `file://`, absolute local paths, and unparseable input (`not-a-url` still errors).

## Non-goals

- The duplicated error output (`Error: …` from cobra followed by `error: …` from `main`) shown in the bug report. That is a CLI error-reporting concern in `cmd/swm`, not URL parsing, and belongs in its own change.
- Supporting non-git VCS URL schemes, credential-bearing HTTPS URLs, or `~user` scp paths.
- Any change to how the host composes paths from a `ProjectID` — the plugin still returns the tuple only.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `vcs-git`: the "ParseRemoteURL returns ProjectID" requirement is amended to specify arbitrary/absent SSH users in scp-style remotes, port stripping for schemed URLs, and rejection of empty path segments.

## Impact

- **Capability surface**: `vcs`. No proto changes; `ParseRemoteURLRequest`/`ProjectID` are unchanged.
- **Code**: `plugins/vcs-git/internal/vcs/git.go` (`sshURLRe`, `parseURL`) and `plugins/vcs-git/internal/vcs/git_test.go`.
- **Behavioural**: previously-failing clones now succeed. Two inputs that previously returned a `ProjectID` change: a schemed URL with an explicit port loses the `:port` suffix from its host, and a URL with a trailing or doubled slash is now rejected instead of returning a `ProjectID` with an empty segment. Both previously produced an unusable on-disk path (a directory named `host:2222`, or an empty path component), so no migration is needed. Every other currently-succeeding URL keeps its exact `ProjectID`.
