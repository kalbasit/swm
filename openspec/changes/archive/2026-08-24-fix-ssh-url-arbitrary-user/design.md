## Context

All remote-URL parsing lives in one unexported function, `parseURL` in `plugins/vcs-git/internal/vcs/git.go`, reached from both `ParseRemoteURL` and `DetectProjectAtPath`. It dispatches over four shapes in order: scp-style SSH (via `sshURLRe`), absolute local path, `file://`, and everything else via `net/url.Parse`.

`net/url.Parse` accepts scp syntax without erroring — it parses `org-1470@git.entreprise.com:Team/Project.git` as `Opaque`-ish input with an empty `Host` — so a scp-style URL that misses the regex does not fail loudly at parse time; it fails at the `u.Host == ""` guard several lines later. Any fix must therefore decide scp-vs-URL *before* handing the string to `net/url`, not by falling back on a parse error.

See proposal.md for motivation.

## Goals / Non-Goals

**Goals:**

- Keep all shape dispatch inside `parseURL`; callers and the gRPC surface are untouched.
- Make the scp/URL decision explicit and total, so no input silently lands in the wrong branch.
- Preserve every currently-passing input's `ProjectID` byte-for-byte, apart from the declared port-stripping change.

**Non-Goals:**

- Reimplementing git's full URL grammar (`~user` paths, `[]`-bracketed IPv6 scp hosts, credential-bearing HTTPS).
- Reusing a third-party git-URL parsing library; the surface is small enough that a dependency is not worth the supply-chain cost.

## Decisions

**Detect scp-style by the absence of `://`, not by a `git@` prefix.** Git's own rule (`connect.c`) is: a URL with `://` is schemed; otherwise, if a `:` appears before any `/`, it is scp-style. Mirroring that gives a total, order-independent classification.

- Alternative — widen the regex to `^([^@]+)@([^:]+):(.+)$`, keeping the mandatory `@`: rejected because it still fails on the userless form `host:path`, which git accepts.
- Alternative — try `url.Parse` first and fall back to scp on error: rejected because `url.Parse` does not error on scp input, so there is nothing to fall back from.

**Match scp with `^(?:[^/@]+@)?([^/:]+):(.+)$`, discarding the user.** `[^/@]+@` is optional and non-capturing — the user is parsed only to be dropped, since it is an access detail, not identity. `[^/:]+` for the host prevents a leading `/` or a second `:` from being swallowed, so absolute paths and schemed URLs cannot match even if the branch order changes.

**Order the branches: absolute path → scp → schemed.** The absolute-path check moves ahead of the scp check. Both regex and ordering independently exclude `/tmp/x`, but relying on either alone makes the other a silent load-bearing detail.

**Strip `.git` with `strings.TrimSuffix` on a greedy capture rather than an optional regex group.** The existing `(.+?)(?:\.git)?$` is correct but reads as if it could be, and every other branch already uses `TrimSuffix`. Uniformity here is worth more than saving a line.

**Use `u.Hostname()` instead of `u.Host` for schemed URLs.** `Hostname()` drops the port (and unwraps IPv6 brackets). A port is a transport detail; it must not become a directory name.

**Validate segments once, at the end.** All four branches converge on a `strings.Split` result, so a single `slices.Contains(segments, "")` guard before returning covers trailing slashes, doubled slashes, and empty paths uniformly, replacing the per-branch `path == ""` checks.

## Risks / Trade-offs

- **A bare `host:path` now parses where it previously errored** → An input like `foo:bar` becomes `ProjectID{host: "foo", segments: ["bar"]}` instead of `InvalidArgument`. This matches what `git clone foo:bar` does, so accepting it is more correct than the old rejection, and a bad host surfaces immediately as a clone failure.
- **Port stripping changes the `ProjectID` for `ssh://host:port/...` remotes** → Such a project would have been stored under a directory containing a colon. If any user has one on disk, swm will no longer find it at the old path and will re-clone. No migration is written because the old path shape was already broken.
- **Two hosts differing only by port collapse to one directory** → Accepted. The layout is host-keyed by design (TDD §5.1) and multi-port git hosting on one hostname is vanishingly rare.
- **Regression risk to the four existing shapes** → Mitigated by keeping every existing `TestParseRemoteURL_*` assertion unchanged and adding the new cases alongside them, so a behaviour change in an old shape fails an old test.
