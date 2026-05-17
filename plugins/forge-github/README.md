# swm-plugin-forge-github

GitHub forge plugin for swm. Handles pull request listing and creation for repositories hosted on `github.com`.

## Purpose

Implements the `forge` capability surface for GitHub. It claims the `github.com` host, meaning any project whose remote URL resolves to `github.com` will use this plugin for forge operations (`swm pr list`, `swm pr create`).

## Requirements

- A GitHub personal access token with `repo` scope (or a fine-grained token with pull-request read/write permissions on the target repositories).
- Network access to `api.github.com`.

## Configuration

Configure under `[plugins.config.forge-github]` in `config.toml`:

| Key          | Type   | Default                      | Description                                                    |
| ------------ | ------ | ---------------------------- | -------------------------------------------------------------- |
| `token_path` | string | `~/.config/swm/github_token` | Path to a file containing the GitHub token (newline-stripped). |

### Example

```toml
[plugins]
forges = ["github"]

[plugins.config.forge-github]
token_path = "~/.config/swm/github_token"
```

Create the token file:

```sh
echo "ghp_yourtoken" > ~/.config/swm/github_token
chmod 600 ~/.config/swm/github_token
```

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
- The plugin reads a static token from a file; OAuth device flow is not supported yet.
- At most one forge plugin can claim a given host; if you need multiple GitHub identities, use separate swm installations.
