## Context

`swm workspace open [story]` and `swm story remove [story]` each accept an optional story name positional argument. Currently no `ValidArgsFunction` is registered on either command, so shells offer no dynamic completions for that argument. Both commands already receive a `coreStory.Store` parameter, so the story list is available without any new wiring.

The existing `shell-completion` spec covers static script generation (bash/zsh/fish) and Nix installation. It does not address dynamic argument completions. Cobra's shell completion infrastructure supports dynamic completions via `ValidArgsFunction` on a `*cobra.Command`; the generated scripts already call back to `swm __complete` at tab time, so no changes to the generated scripts or Nix installation are needed.

## Goals / Non-Goals

**Goals:**
- Register a `ValidArgsFunction` on `workspace open` that returns all story names from the store.
- Register a `ValidArgsFunction` on `story remove` that returns all story names from the store.
- Completions return `cobra.ShellCompDirectiveNoFileComp` so the shell does not fall back to filename completion.

**Non-Goals:**
- Adding completion to `story create` (new name, not an existing one).
- Changing the static completion scripts or Nix installation.
- Filtering out `_default` from completion results (user may want to open it explicitly).

## Decisions

### Use `ValidArgsFunction` on each command directly

Each command constructor (`NewOpenCmd`, `NewRemoveCmd`) already takes `store coreStory.Store`. Setting `cmd.ValidArgsFunction` inline in the constructor keeps the completion logic co-located with the command, requires no new types or packages, and is the idiomatic cobra pattern.

Alternative considered: a shared helper `StoryNameCompletionFunc(store)` extracted to a shared package. Rejected — two call sites don't justify an abstraction, and the helper would need to import `coreStory` anyway, creating no real decoupling.

### Return all stories; let the shell filter by prefix

`ValidArgsFunction` returns the full list; the shell handles prefix filtering. This is the standard cobra approach and avoids re-implementing prefix matching.

### Errors in `ValidArgsFunction` are silent

If `store.List` fails, return an empty slice with `cobra.ShellCompDirectiveError`. This degrades gracefully (no completion, no crash) and matches cobra convention. No logging — completion runs in a short-lived subprocess where log output would corrupt completion output.

## Risks / Trade-offs

- **Store I/O on every tab press**: `store.List` reads the story JSON store from disk. For typical story counts (< 100) this is negligible. No caching needed.
- **Completion subprocess context**: cobra spawns a child process for `__complete`. The store path comes from the same config resolution as the main command, so it is correct.

## Migration Plan

No migration needed. Additive change; existing scripts continue to work unchanged.

## Open Questions

None.
