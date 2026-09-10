## 1. story list writes to stdout

- [x] 1.1 Print story names to stdout rather than through cobra's Print family; verify by capturing the process's own stdout with stderr discarded, since a test that sets the command's out writer cannot tell the streams apart
- [x] 1.2 Verify the new test fails against the unfixed code, because the existing ones could not

## 2. The contract carries tags

- [x] 2.1 Add `tags` to `OpenPaneRequest` and to `Pane` in the proto, documented as opaque to swm and belonging to the pane rather than the process; verify `task proto:lint` and generation are clean
- [x] 2.2 Verify the field is additive: a provider that ignores tags still satisfies the contract, and a pane opened without them reports none

## 3. The tmux provider stores them

- [x] 3.1 Set tags as pane-scoped options on the pane OpenPane creates, in deterministic order; verify the commands issued against the fake tmux
- [x] 3.2 Report tags from ListPanes; verify a pane opened with tags lists with them and one opened without lists with none
- [x] 3.3 Verify tags survive the pane's program exiting and being replaced, which is the case `--env` cannot serve and the reason for the whole change

## 4. The CLI surface

- [x] 4.1 Add `--tag KEY=VALUE`, repeatable, parsed exactly as `--env`; verify a malformed entry exits non-zero before any plugin call
- [x] 4.2 Report tags in `pane list --json`, omitted when a pane has none; verify both
- [x] 4.3 Document `--tag` in the host CLI README next to `--env`, saying plainly that a tag belongs to the pane and an env entry to the process

## 5. Proof on this machine

- [x] 5.1 Open a tagged pane in a real workspace, verify `pane list --json` reports the tag, then kill the program in that pane and verify the tag is still reported
- [x] 5.2 Verify `swm story list` survives capture: `test -n "$(swm story list 2>/dev/null)"`
