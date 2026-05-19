# Delta Spec: plugin-lifecycle

## MODIFIED Requirements

### Requirement: Plugin discovery
The plugin manager SHALL discover plugin binaries in the following priority order,
stopping at the first match per capability:
(0) each directory listed in `$SWM_PLUGIN_PATH` (colon-separated, evaluated
left-to-right) — non-existent or non-directory entries MUST be silently skipped,
(1) explicit paths from `config.toml [plugins.paths]`,
(2) `$XDG_DATA_HOME/swm/plugins/<name>/swm-plugin-<capability>-<name>`,
(3) `$PATH` lookup for `swm-plugin-<capability>-<name>`.
The binary naming convention MUST be `swm-plugin-<capability>-<name>`.

#### Scenario: PATH discovery
- **WHEN** no explicit path or XDG path is configured for capability `vcs` and `swm-plugin-vcs-git` is in `$PATH`
- **THEN** the manager discovers `swm-plugin-vcs-git` as the vcs plugin

#### Scenario: Explicit config overrides PATH
- **WHEN** `config.toml` sets an explicit path for `vcs` and `swm-plugin-vcs-git` is also in `$PATH`
- **THEN** the manager uses the explicitly configured binary, not the PATH one

#### Scenario: Missing required plugin
- **WHEN** `config.toml` specifies `vcs = "git"` and no `swm-plugin-vcs-git` is found in any search location
- **THEN** `Manager.Get("vcs")` returns an error describing which capability binary was not found

#### Scenario: SWM_PLUGIN_PATH takes precedence over all other search locations
- **WHEN** `SWM_PLUGIN_PATH=/dev/bin`, `swm-plugin-vcs-git` exists in `/dev/bin`, and `swm-plugin-vcs-git` is also discoverable via `$PATH`
- **THEN** the manager uses `/dev/bin/swm-plugin-vcs-git`

#### Scenario: SWM_PLUGIN_PATH takes precedence over explicit config paths
- **WHEN** `SWM_PLUGIN_PATH=/dev/bin`, `swm-plugin-vcs-git` exists in `/dev/bin`, and `config.toml` sets an explicit path for `vcs` pointing elsewhere
- **THEN** the manager uses `/dev/bin/swm-plugin-vcs-git`

#### Scenario: SWM_PLUGIN_PATH colon-separated list searched left-to-right
- **WHEN** `SWM_PLUGIN_PATH=/dir1:/dir2` and `swm-plugin-vcs-git` exists only in `/dir2`
- **THEN** the manager discovers `/dir2/swm-plugin-vcs-git`

#### Scenario: Non-existent SWM_PLUGIN_PATH entries are silently skipped
- **WHEN** `SWM_PLUGIN_PATH=/nonexistent:/dir2` and `swm-plugin-vcs-git` exists in `/dir2`
- **THEN** `/nonexistent` is silently skipped and the manager discovers `/dir2/swm-plugin-vcs-git`

#### Scenario: Unset SWM_PLUGIN_PATH leaves discovery unchanged
- **WHEN** `SWM_PLUGIN_PATH` is not set and `swm-plugin-vcs-git` is in `$PATH`
- **THEN** the manager discovers `swm-plugin-vcs-git` via PATH as before
