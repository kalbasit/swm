# Design: workspace-open-positional-argz

## Context

`workspace open` is the primary daily-driver command in swm. The current interface requires `--story <name>` which is more verbose than the idiomatic positional form. The entire change is contained in one file (`cmd/swm/internal/cli/workspace/open.go`) plus integration tests and the spec delta.

## Goals / Non-Goals

**Goals:**
- Replace `--story` flag with an optional positional argument
- Preserve the existing resolution fallback chain: positional arg → `$SWM_STORY` → `cfg.DefaultStory`

**Non-Goals:**
- Changing fallback behaviour or priority order
- Deprecating or aliasing the old flag (hard removal)

## Decisions

### Use `cobra.MaximumNArgs(1)` + `args[0]` instead of `StringVar`

Replace `cmd.Flags().StringVar(&storyName, "story", ...)` with `Args: cobra.MaximumNArgs(1)` and read `args[0]` inside `RunE`. This is the standard cobra idiom for optional positional arguments. No helper libraries needed.

**Alternatives considered:**
- Keep `--story` as an alias: adds maintenance burden and contradicts the goal of simplifying the interface.
- Accept both positional and `--story` with mutual exclusion: unnecessary complexity for a single optional arg.

## Risks / Trade-offs

- **Breaking change**: existing scripts/aliases using `--story` will break. Mitigation: this is intentional and the proposal explicitly marks it breaking.
- **Autocomplete**: cobra's completion for positional args is less ergonomic than flag completion for some shells. Mitigation: out of scope for this change; story name completion can be added separately.

## Migration Plan

1. Remove `--story` flag registration from `NewOpenCmd`.
2. Change `RunE` signature to use `args []string` and extract `storyName` from `args[0]` when present.
3. Update integration tests that pass `--story`.
4. No rollback required — this is a CLI-only change with no persisted state.
