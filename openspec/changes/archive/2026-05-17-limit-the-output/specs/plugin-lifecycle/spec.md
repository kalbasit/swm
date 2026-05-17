## ADDED Requirements

### Requirement: Plugin manager logger level propagation
The plugin manager SHALL configure go-plugin's internal `hclog` logger to the same level as the host's active log level (`--log-level` flag). When no level is explicitly configured, the hclog logger SHALL default to `warn`. At no point SHALL the hclog logger use a level lower than the host's configured level (i.e., it MUST NOT emit DEBUG or TRACE lines when the host is at `warn` or higher).

#### Scenario: Default log level suppresses DEBUG output
- **WHEN** `swm` is invoked without `--log-level` (default is `warn`)
- **THEN** no `[DEBUG]` or `[TRACE]` lines from go-plugin appear on stderr during plugin launch

#### Scenario: Debug log level shows go-plugin internal lines
- **WHEN** `swm` is invoked with `--log-level debug`
- **THEN** go-plugin `[DEBUG]` lines for plugin startup, handshake, and RPC address appear on stderr

#### Scenario: Warn level preserves plugin error visibility
- **WHEN** `swm` is invoked without `--log-level` and a plugin process crashes
- **THEN** go-plugin `[WARN]` or `[ERROR]` lines for the crash appear on stderr
