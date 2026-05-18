## Context

`swm story remove <name>` currently declares `cobra.ExactArgs(1)`, making the story name a mandatory positional argument. Users running the command from within an active story workspace must manually type or expand `$SWM_STORY` — an environment variable the host already injects into every story workspace session.

Other commands in the same CLI (`swm workspace open`, `swm pr list`, `swm pr create`) already read `$SWM_STORY` as a fallback when no explicit story name is given. This change brings `swm story remove` in line with that established pattern.

The change touches a single file (`cmd/swm/internal/cli/story/remove.go`) and a thin layer of tests.

## Goals / Non-Goals

**Goals:**
- Allow `swm story remove` to be invoked with zero positional arguments, resolving the story name from `$SWM_STORY`.
- Produce a clear error when the argument is omitted and `$SWM_STORY` is unset or empty.
- Keep all existing invocations (explicit `<name>`) working identically.

**Non-Goals:**
- Adding env-var fallback to `swm story create` or `swm story list`.
- Changing the confirmation-prompt logic or the `--force` flag.
- Adding a config-level default for `story remove`.

## Decisions

### D1: `cobra.MaximumNArgs(1)` instead of `cobra.ExactArgs(1)`

Cobra's built-in arg validators are the idiomatic way to control arity. Changing to `MaximumNArgs(1)` lets Cobra still reject 2+ args with a helpful error, while allowing 0. The name resolution block runs inside `RunE`, after Cobra validates arity.

_Alternative considered_: a custom `Args` validator that also checks `$SWM_STORY` — rejected because it conflates arity validation with business logic and produces confusing error messages.

### D2: Resolution precedence matches other commands

```
1. Positional <name> argument (if provided)
2. $SWM_STORY environment variable (if set and non-empty)
3. Error — cannot proceed without a story name
```

This is the same precedence used by `swm pr list` and `swm pr create` (`--story` flag → `$SWM_STORY` → error). Consistency lowers the cognitive load for users and readers of the code.

### D3: No interactive picker fallback

`swm workspace open` falls back to an interactive story picker when both the arg and `$SWM_STORY` are absent. `story remove` is a destructive operation; silently presenting a picker for a destructive command would be surprising. If neither source provides a name, the command errors out immediately.

## Risks / Trade-offs

[Accidental removal] A user who forgets they are inside a story workspace might run `swm story remove` intending to remove a different story, only to find the current one removed. → Mitigation: the existing confirmation prompt (listing all attached projects) guards against this in non-`--force` invocations. The `--force` path remains as explicit as before.

[No migration needed] All existing call-sites that pass an explicit `<name>` continue to work — `MaximumNArgs(1)` accepts 0 or 1 args, so the change is purely additive for callers.
