### Requirement: Story JSON persistence
The story store SHALL persist each story as a JSON file at `$XDG_DATA_HOME/swm/stories/<name>.json`. The file MUST contain: `name` (string), `branch_name` (string), `created_at` (RFC3339 timestamp), `vcs` (string plugin name), `projects` (array of project entries), and `metadata` (free-form object). Worktree paths SHALL be derived at runtime and MUST NOT be stored.

#### Scenario: Create persists JSON file
- **WHEN** `Store.Create(ctx, "feat-x", "feat/feat-x")` is called
- **THEN** a file at `$XDG_DATA_HOME/swm/stories/feat-x.json` is written with `name="feat-x"`, `branch_name="feat/feat-x"`, and `created_at` set to the current UTC time

#### Scenario: Create with duplicate name
- **WHEN** `Store.Create` is called with a name that already has a JSON file on disk
- **THEN** an error wrapping a sentinel `ErrStoryExists` is returned and no file is written

#### Scenario: Get reads existing story
- **WHEN** `Store.Get(ctx, "feat-x")` is called and `feat-x.json` exists
- **THEN** the deserialized `*Story` is returned with all fields matching the file content

#### Scenario: Get unknown story
- **WHEN** `Store.Get` is called for a name with no corresponding JSON file
- **THEN** an error wrapping `ErrStoryNotFound` is returned

#### Scenario: List returns all stories
- **WHEN** `Store.List(ctx)` is called and three JSON files exist in the stories directory
- **THEN** a slice of three `*Story` values is returned in lexical name order

#### Scenario: Delete removes file
- **WHEN** `Store.Delete(ctx, "feat-x")` is called and `feat-x.json` exists
- **THEN** the file is removed from disk and subsequent `Get` returns `ErrStoryNotFound`

### Requirement: Write locking via flock
The story store SHALL acquire an exclusive file-system lock (flock) on the story JSON path before any write operation (Create, Update, Delete). The lock MUST be released before the function returns.

#### Scenario: Concurrent writes serialize
- **WHEN** two goroutines call `Store.Create` for different story names simultaneously
- **THEN** both succeed without data corruption (each acquires its own per-file lock)

#### Scenario: Lock released on error
- **WHEN** a write operation acquires the flock but encounters a serialization error
- **THEN** the flock is released before the error is returned

### Requirement: Default story bootstrapping
The story store SHALL ensure a `_default.json` story file exists whenever the stories directory is initialized (i.e., on first access). The `_default` story represents the baseline state with no active story.

#### Scenario: First access creates default story
- **WHEN** the stories directory does not yet exist and `Store.List` is called
- **THEN** the directory is created, `_default.json` is written with `name="_default"`, and the default story is included in the returned list

### Requirement: Project attachment
The story store SHALL allow attaching a project (identified by a `ProjectID` with `host` and `segments`) to an existing story via `Store.Update`. Projects are appended to `projects[]` with an `attached_at` timestamp. Duplicate attachments (same host+segments) MUST be rejected.

#### Scenario: Attach new project
- **WHEN** `Store.Update` is called with a story that has a new project added to its `projects` slice
- **THEN** the JSON file is rewritten with the new project entry and `attached_at` set to the current UTC time

#### Scenario: Duplicate project rejected
- **WHEN** `Store.Update` is called with a project already present in `projects[]`
- **THEN** an error wrapping `ErrProjectAlreadyAttached` is returned and the file is not modified
