## 1. Update session-tmux README

- [x] 1.1 Add a "Template variables" sub-section to the Configuration section listing `{{worktree_path}}`, `{{story_name}}`, `{{project_id}}`, and `{{tmux_socket}}` with descriptions
- [x] 1.2 Add a "Laio integration" section with the per-project config example (`{{worktree_path}}/.swm/laio.yaml`)
- [x] 1.3 Add the global config example to the "Laio integration" section (`~/.config/swm/laio.yaml` + `--var path={{worktree_path}}`)
- [x] 1.4 Add an explanation of why `--skip-attach` is required

## 2. Add sample laio.yaml

- [x] 2.1 Create `plugins/session-tmux/examples/laio.yaml` with a multi-window layout using `path: "{{ path }}"` and compatible with swm's socket model (no attach, `force_new_windows: true`)
- [x] 2.2 Link to `examples/laio.yaml` from the "Laio integration" section of the README
