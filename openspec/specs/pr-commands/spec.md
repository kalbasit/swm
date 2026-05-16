### Requirement: swm pr list
`swm pr list [--story <name>]` SHALL list open pull requests for all projects attached to the given story (resolved from `--story` flag, `$SWM_STORY`, or `_default`). For each attached project it SHALL look up the forge plugin for the project's host and call `forge.ListPullRequests`. Output SHALL be one line per PR formatted as: `#<number>\t<title>\t<url>`. If no forge plugin is registered for a project's host, that project is silently skipped. If no PRs exist across all projects, the command prints nothing and exits 0.

#### Scenario: Open PRs listed for all attached projects
- **WHEN** `swm pr list --story feat-x` is run and `feat-x` has two attached projects each with open PRs
- **THEN** one line per PR is printed to stdout with the PR number, title, and URL

#### Scenario: No open PRs
- **WHEN** `swm pr list --story feat-x` is run and no attached project has open PRs
- **THEN** the command prints nothing and exits 0

#### Scenario: No forge configured for project host
- **WHEN** `swm pr list` is run and one project's host has no forge plugin registered
- **THEN** that project is silently skipped; other projects' PRs are still listed

#### Scenario: Story not found
- **WHEN** `swm pr list --story nonexistent` is run
- **THEN** the command exits non-zero with an error indicating the story was not found

### Requirement: swm pr create
`swm pr create [--story <name>] --title <title> [--body <body>] [--base <base>] [--draft]` SHALL create a pull request for the project in the current working directory (detected via `vcs.DetectProjectAtPath`). It SHALL look up the forge plugin for the project's host and call `forge.CreatePullRequest` with: `project_id` derived from detection, `title` from `--title`, `body` from `--body` (default: empty), `head_branch` from the story's `branch_name`, `base_branch` from `--base` (default: `main`), `draft` from `--draft` flag. On success it SHALL print the PR URL to stdout.

#### Scenario: Successful PR creation
- **WHEN** `swm pr create --story feat-x --title "My PR"` is run in a project directory
- **THEN** `forge.CreatePullRequest` is called with the correct fields and the PR URL is printed to stdout

#### Scenario: Draft PR creation
- **WHEN** `swm pr create --title "My PR" --draft` is run
- **THEN** `forge.CreatePullRequest` is called with `draft = true`

#### Scenario: Custom base branch
- **WHEN** `swm pr create --title "My PR" --base develop` is run
- **THEN** `forge.CreatePullRequest` is called with `base_branch = "develop"`

#### Scenario: No forge configured for current project host
- **WHEN** `swm pr create` is run in a directory whose host has no forge plugin registered
- **THEN** the command exits non-zero with an error indicating no forge plugin for the host

#### Scenario: --title flag required
- **WHEN** `swm pr create` is run without `--title`
- **THEN** the command exits non-zero with a usage error indicating `--title` is required
