# swm-plugin-forge-github

GitHub forge plugin for swm. Handles pull request listing and creation for repositories hosted on `github.com`.

## Purpose

Implements the `forge` capability surface for GitHub. It claims the `github.com` host, meaning any project whose remote URL resolves to `github.com` will use this plugin for forge operations (`swm pr list`, `swm pr create`).

## Requirements

- Network access to `api.github.com`.
- A GitHub credential available via one of the methods below.

## Authentication

The plugin resolves a GitHub token using the following priority order:

1. **`token_path` config key** — if set, the token is read from that file. Missing or empty file is a hard error (no fallback).
2. **`gh auth token`** — if the [GitHub CLI](https://cli.github.com/) is on `$PATH` and authenticated, its token is used automatically. This is the default for most developer machines.
3. **`~/.github_token` file** — last-resort fallback for environments without `gh`.

### Recommended: use the GitHub CLI

```sh
# Install gh (https://cli.github.com/), then:
gh auth login
```

No swm configuration needed — the plugin picks up `gh`'s credentials automatically.

### Alternative: token file

```sh
echo "ghp_yourtoken" > ~/.github_token
chmod 600 ~/.github_token
```

Or point to a custom path via config:

```toml
[plugins.config.forge-github]
token_path = "~/.config/swm/github_token"
```

## Configuration

Configure under `[plugins.config.forge-github]` in `config.toml`:

| Key          | Type   | Default | Description                                                                                          |
| ------------ | ------ | ------- | ---------------------------------------------------------------------------------------------------- |
| `token_path` | string | —       | Path to a file containing the GitHub token. Takes priority over `gh auth`. Tilde (`~/`) is expanded. |

`token_path` is optional. When absent, the plugin uses `gh auth token` or `~/.github_token`.

## Usage

Once configured, forge operations work through the swm CLI:

```sh
# List open pull requests for the current story
swm pr list

# Create a pull request
swm pr create --title "feat: add thing" --body "Closes #123" --draft
```

## Limitations

- Only `github.com` is claimed. GitHub Enterprise Server is not supported in this release.
- OAuth device flow is not supported; use `gh auth login` or a personal access token.
- At most one forge plugin can claim a given host; if you need multiple GitHub identities, use separate swm installations.
