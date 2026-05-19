# Spec: config-command

CLI subcommands for reading and writing swm configuration without manually editing the TOML file.

---

## Requirement: swm config get

`swm config get <key>` prints the effective value of a registered config key to stdout, followed by a newline. The effective value is the file-configured value if present, otherwise the compiled-in default.

### Scenario: Get a key that has a default and is not in the config file
- **GIVEN** no config file exists (or the key is absent from the file)
- **WHEN** `swm config get code_root` is run
- **THEN** the command exits zero and prints the default value (`~/code`)

### Scenario: Get a key that is set in the config file
- **GIVEN** `config.toml` contains `code_root = "/workspace/code"`
- **WHEN** `swm config get code_root` is run
- **THEN** the command exits zero and prints `/workspace/code`

### Scenario: Get a nested key
- **GIVEN** `config.toml` contains `[plugins]\nsession = "tmux"`
- **WHEN** `swm config get plugins.session` is run
- **THEN** the command exits zero and prints `tmux`

### Scenario: Get a map sub-key
- **GIVEN** `config.toml` contains `[plugins.paths]\nvcs-git = "/usr/local/bin/swm-plugin-vcs-git"`
- **WHEN** `swm config get plugins.paths.vcs-git` is run
- **THEN** the command exits zero and prints `/usr/local/bin/swm-plugin-vcs-git`

### Scenario: Get an unknown key
- **WHEN** `swm config get does.not.exist` is run
- **THEN** the command exits non-zero and prints an error message listing valid keys to stderr

### Scenario: Get with missing argument
- **WHEN** `swm config get` is run with no arguments
- **THEN** the command exits non-zero and prints usage to stderr

---

## Requirement: swm config set

`swm config set <key> <value>` writes a new scalar value for a registered writable key to the config file. If the config file does not exist, it is created (including parent directories). Comments in an existing file are not preserved after a write.

### Scenario: Set a top-level key in an existing config file
- **GIVEN** `config.toml` exists with `code_root = "~/code"`
- **WHEN** `swm config set code_root /workspace` is run
- **THEN** the command exits zero, and `config.toml` contains `code_root = "/workspace"`

### Scenario: Set creates the config file when it does not exist
- **GIVEN** no config file exists
- **WHEN** `swm config set default_story main` is run
- **THEN** the command exits zero, the config file is created, and it contains `default_story = "main"`

### Scenario: Set a nested key
- **WHEN** `swm config set plugins.session tmux` is run
- **THEN** the command exits zero and `config.toml` contains `session = "tmux"` under `[plugins]`

### Scenario: Set a map sub-key
- **WHEN** `swm config set plugins.paths.vcs-git /usr/local/bin/swm-plugin-vcs-git` is run
- **THEN** the command exits zero and `config.toml` contains the path under `[plugins.paths]`

### Scenario: Set a non-writable key
- **WHEN** `swm config set plugins.forges "github"` is run
- **THEN** the command exits non-zero and prints an error message indicating the key is not writable via `set` in this version

### Scenario: Set an unknown key
- **WHEN** `swm config set does.not.exist value` is run
- **THEN** the command exits non-zero and prints an error message listing valid keys to stderr

### Scenario: Set with wrong argument count
- **WHEN** `swm config set plugins.session` is run with only one argument
- **THEN** the command exits non-zero and prints usage to stderr

---

## Requirement: swm config list

`swm config list` prints only the keys that are explicitly present in the config file, in `key = value` format (one per line). If no config file exists, it prints nothing and exits zero.

### Scenario: No config file
- **GIVEN** no config file exists
- **WHEN** `swm config list` is run
- **THEN** the command exits zero and prints nothing to stdout

### Scenario: Config file with some keys set
- **GIVEN** `config.toml` contains `code_root = "/workspace"` and `[plugins]\nsession = "tmux"`
- **WHEN** `swm config list` is run
- **THEN** the command exits zero and prints exactly:
  ```
  code_root = /workspace
  plugins.session = tmux
  ```

### Scenario: Array key displayed inline
- **GIVEN** `config.toml` contains `[plugins]\nforges = ["github"]`
- **WHEN** `swm config list` is run
- **THEN** the command exits zero and the output includes `plugins.forges = ["github"]`

---

## Requirement: swm config list --all

`swm config list --all` prints every registered key with its effective value (configured or default), in key registry definition order.

### Scenario: Fresh install with no config file
- **GIVEN** no config file exists
- **WHEN** `swm config list --all` is run
- **THEN** the command exits zero and all registered keys are printed with their default values

### Scenario: Some keys configured
- **GIVEN** `config.toml` contains `code_root = "/workspace"`
- **WHEN** `swm config list --all` is run
- **THEN** the command exits zero, `code_root` shows `/workspace`, and all other keys show their defaults

### Scenario: Output order is stable
- **WHEN** `swm config list --all` is run twice
- **THEN** both outputs are identical (keys appear in registry definition order, not map iteration order)
