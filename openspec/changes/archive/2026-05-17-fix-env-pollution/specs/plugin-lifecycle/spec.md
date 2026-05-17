## ADDED Requirements

### Requirement: Plugin-internal environment variable contract

The plugin manager defines a fixed set of environment variables that are classified as plugin-internal. These variables MUST be present in every plugin subprocess environment and MUST NOT appear in any process that is not a designated plugin subprocess.

Plugin-internal variables:

| Variable | Purpose |
|---|---|
| `SWM_HOST_SOCKET` | gRPC socket address for plugin-to-host callbacks |
| `SWM_LOG_LEVEL` | host log level propagated to plugin logging |
| `SWM_PLUGIN_MAGIC_COOKIE` | go-plugin handshake token |

#### Scenario: Plugin subprocess receives all plugin-internal vars
- **WHEN** the plugin manager launches any plugin subprocess
- **THEN** `SWM_HOST_SOCKET`, `SWM_LOG_LEVEL`, and `SWM_PLUGIN_MAGIC_COOKIE` are all present in that process's environment

#### Scenario: Plugin-internal vars are not required in non-plugin processes
- **WHEN** the session plugin launches a terminal multiplexer process
- **THEN** `SWM_HOST_SOCKET`, `SWM_LOG_LEVEL`, and `SWM_PLUGIN_MAGIC_COOKIE` are absent from that process's environment

#### Scenario: Plugin-internal vars are not required in hook processes
- **WHEN** the hook executor launches a hook binary
- **THEN** `SWM_HOST_SOCKET`, `SWM_LOG_LEVEL`, and `SWM_PLUGIN_MAGIC_COOKIE` are absent from that process's environment
