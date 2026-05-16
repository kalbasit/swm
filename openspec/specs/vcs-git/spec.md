### Requirement: ParseRemoteURL returns ProjectID
`vcs-git` SHALL implement `VCS.ParseRemoteURL(url)` by parsing the remote URL into a `ProjectID(host, segments[])`. It MUST handle SSH (`git@github.com:owner/repo.git`), HTTPS (`https://github.com/owner/repo.git`), and git+ssh formats. The `.git` suffix SHALL be stripped. The host MUST compose all on-disk paths; the plugin MUST NOT return any path information.

#### Scenario: Parse SSH URL
- **WHEN** `ParseRemoteURL("git@github.com:kalbasit/swm.git")` is called
- **THEN** `ProjectID{host: "github.com", segments: ["kalbasit", "swm"]}` is returned

#### Scenario: Parse HTTPS URL
- **WHEN** `ParseRemoteURL("https://gitlab.com/foo/bar/baz.git")` is called
- **THEN** `ProjectID{host: "gitlab.com", segments: ["foo", "bar", "baz"]}` is returned

#### Scenario: Unparseable URL
- **WHEN** `ParseRemoteURL("not-a-url")` is called
- **THEN** a gRPC status error with code `InvalidArgument` is returned

### Requirement: Clone to canonical path
`vcs-git` SHALL implement `VCS.Clone(url, canonical_path)` by running `git clone <url> <canonical_path>`. The `canonical_path` is provided by the host (derived from `ParseRemoteURL` output). The plugin MUST NOT compute or choose the destination path. Clone SHALL fail with `AlreadyExists` gRPC status if the canonical path already contains a `.git` directory.

#### Scenario: Successful clone
- **WHEN** `Clone({url: "git@github.com:kalbasit/swm.git", canonical_path: "/tmp/code/repositories/github.com/kalbasit/swm"})` is called
- **THEN** `git clone` is executed and the repository is present at `canonical_path`

#### Scenario: Already cloned
- **WHEN** `Clone` is called with a `canonical_path` that already contains a `.git` directory
- **THEN** a gRPC `AlreadyExists` status error is returned without running `git clone`

#### Scenario: git clone failure
- **WHEN** `git clone` exits non-zero (e.g., repository not found, auth error)
- **THEN** a gRPC `Internal` status error is returned with stderr captured in the message

### Requirement: CreateWorktree for a story
`vcs-git` SHALL implement `VCS.CreateWorktree({canonical_path, worktree_path, branch_name})` by running `git -C <canonical_path> worktree add <worktree_path> <branch_name>`. If the branch does not exist, it SHALL be created (`--orphan` is not used; `git worktree add -b <branch>` creates it). The worktree directory's parent MUST be created if absent.

#### Scenario: Create worktree with existing branch
- **WHEN** `CreateWorktree({canonical_path: "/code/repositories/github.com/k/s", worktree_path: "/code/stories/feat-x/github.com/k/s", branch_name: "feat/feat-x"})` is called and the branch exists
- **THEN** `git worktree add <worktree_path> feat/feat-x` is run and the directory is populated

#### Scenario: Create worktree with new branch
- **WHEN** `CreateWorktree` is called with a `branch_name` that does not exist in the repository
- **THEN** `git worktree add -b <branch_name> <worktree_path>` creates the branch and worktree

#### Scenario: Parent directory creation
- **WHEN** `CreateWorktree` is called and the parent of `worktree_path` does not exist
- **THEN** the parent directories are created before running `git worktree add`

### Requirement: RemoveWorktree for a story
`vcs-git` SHALL implement `VCS.RemoveWorktree({canonical_path, worktree_path})` by running `git -C <canonical_path> worktree remove --force <worktree_path>`. After removal, `git worktree prune` SHALL be run to clean up stale worktree metadata.

#### Scenario: Successful removal
- **WHEN** `RemoveWorktree({canonical_path: "/code/repositories/github.com/k/s", worktree_path: "/code/stories/feat-x/github.com/k/s"})` is called
- **THEN** the worktree directory is removed and `git worktree list` no longer shows it

#### Scenario: Worktree not found
- **WHEN** `RemoveWorktree` is called with a `worktree_path` that is not registered
- **THEN** a gRPC `NotFound` status error is returned

### Requirement: DetectProjectAtPath
`vcs-git` SHALL implement `VCS.DetectProjectAtPath({path})` by running `git -C <path> remote get-url origin` and passing the result through `ParseRemoteURL`. If the path is not a git repository, a gRPC `NotFound` error is returned.

#### Scenario: Detect inside repo
- **WHEN** `DetectProjectAtPath({path: "/code/repositories/github.com/kalbasit/swm"})` is called and `origin` points to `git@github.com:kalbasit/swm.git`
- **THEN** `ProjectID{host: "github.com", segments: ["kalbasit", "swm"]}` is returned

#### Scenario: Not a git repo
- **WHEN** `DetectProjectAtPath({path: "/tmp"})` is called
- **THEN** a gRPC `NotFound` status error is returned

### Requirement: VCS plugin Info
`vcs-git` SHALL implement `VCS.Info()` returning a `VCSInfo` with `plugin_info.name = "git"`, the plugin version, and `project_markers = [".git"]`. The `project_markers` field is used by the host to walk `repositories/` and identify git repositories.

#### Scenario: Info response
- **WHEN** `Info()` is called
- **THEN** a `VCSInfo` with `project_markers: [".git"]` and non-empty `plugin_info.version` is returned
