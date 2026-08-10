## 1. Command scaffolding (cmd/swm)

- [x] 1.1 (cmd/swm) Add `cmd/swm/internal/cli/story/attach.go` with a cobra command `attach [<story-name>]`, wired into the existing `story` command group; accept 0–1 positional args. No logic yet — return a not-implemented error so wiring compiles.
- [x] 1.2 (cmd/swm) Wire `<story-name>` shell completion to the same story-name completion helper used by `story remove`; add a completion test asserting story names are listed.

## 2. Story + project resolution (cmd/swm)

- [x] 2.1 (cmd/swm) TDD: write table tests for story resolution precedence (arg → `$SWM_STORY` → error) and "story not found" error, following the pattern in `story/remove_test.go`.
- [x] 2.2 (cmd/swm) Implement story resolution to satisfy 2.1: explicit arg wins, else `$SWM_STORY`, else exit non-zero before any work; error if the resolved story is absent from the store.
- [x] 2.3 (cmd/swm) TDD: write tests (with a `stubVCSClient`) that the command calls `vcs.DetectProjectAtPath` with the current working directory and surfaces a descriptive error when it returns `NotFound`.
- [x] 2.4 (cmd/swm) Implement project resolution via `vcs.DetectProjectAtPath(cwd)`; compose `RepoPath = resolver.CanonicalPath(pid)` and `WorktreePath = resolver.WorktreePath(story, pid)`.

## 3. Attach logic — three states (cmd/swm)

- [x] 3.1 (cmd/swm) TDD: test the "already attached" no-op — project already in `story.Projects` ⇒ no hooks, no `CreateWorktree`, no store write, exit 0.
- [x] 3.2 (cmd/swm) TDD: test the "create" path — unattached project, worktree absent ⇒ `pre-worktree-create` hook → `vcs.CreateWorktree` (with `story.BranchName`) → store attach → `post-worktree-create` hook, exit 0. Assert hook `RunConfig` fields (`ProjectHost`, `ProjectPath`, `WorktreePath`, `RepoPath`) match `workspace open`.
- [x] 3.3 (cmd/swm) TDD: test the "reconcile" path — unattached project, worktree already present on disk ⇒ no `CreateWorktree`, no create hooks, store attach only, exit 0.
- [x] 3.4 (cmd/swm) TDD: test default-story handling — resolved story is `default_story` ⇒ `CreateWorktree` skipped, hooks still run, project attached.
- [x] 3.5 (cmd/swm) TDD: test hook failure semantics — `pre-worktree-create` non-zero aborts (no `CreateWorktree`, no attach, non-zero exit); `post-worktree-create` non-zero is logged and exit stays 0.
- [x] 3.6 (cmd/swm) TDD: test the concurrency race — `store.Update` returns `ErrProjectAlreadyAttached` ⇒ treated as success (exit 0).
- [x] 3.7 (cmd/swm) TDD: test that no session plugin is loaded and `OpenWorkspace`/`OpenPaneGroup`/`SwitchTo` are never called on any path.
- [x] 3.8 (cmd/swm) Implement the attach logic to satisfy 3.1–3.7, reusing the worktree-create + hook sequence from `workspace/open.go` (steps 6a–6d) minus all session calls; add a filesystem stat helper to detect an existing worktree at `WorktreePath`.

## 4. Integration test (cmd/swm)

- [x] 4.1 (cmd/swm) Add an integration test under `cmd/swm/tests/integration/` using real plugin binaries against a temp `$CODE_ROOT`: `swm story attach` from a canonical clone creates the worktree and records the project in the story JSON; a second invocation is a clean no-op.

## 5. Contrib rule file (docs)

- [x] 5.1 (docs) Add `contrib/claude-rules/swm-story-confinement.md` — a copyable version of the confinement rule, with the worktree-bootstrap section rewritten to instruct agents to run `swm story attach` instead of raw `git worktree add`, and the bookkeeping caveat removed.
- [x] 5.2 (docs) Add a short pointer to the contrib file from the repo's contributor docs/README (where other contrib assets are referenced), so others can discover and copy it.

## 6. Verification

- [x] 6.1 (cmd/swm) Run `task fmt`, `task lint`, and `task test`; confirm each exits 0. (No `proto/` changes ⇒ nix vendorHash task is not required.)
