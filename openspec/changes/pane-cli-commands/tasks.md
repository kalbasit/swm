## 1. Exit status plumbing

- [ ] 1.1 (`cmd/swm`) Write tests for a new `internal/exitcode` package: a nil
      error resolves to 0, a plain error to 1, an error carrying a code to that
      code, and a wrapped error still to its carried code.
- [ ] 1.2 (`cmd/swm`) Add `internal/exitcode` with `Coder`, `Error`, and
      `From`. No `os.Exit` in the package — it only computes the status.
- [ ] 1.3 (`cmd/swm`) Teach `main` to exit with `exitcode.From(err)` instead of
      a hard-coded 1.

## 2. Pane JSON encoding

- [ ] 2.1 (`cmd/swm`) Write tests for the pane JSON view: snake_case keys, all
      fields present when the descriptive ones are empty, and an empty list
      encoding as an empty array rather than null.
- [ ] 2.2 (`cmd/swm`) Add `internal/cli/pane` with the JSON view struct and its
      encoding helpers.

## 3. swm pane open

- [ ] 3.1 (`cmd/swm`) Write tests: argv passed through verbatim including an
      element with spaces, empty argv, `--cwd`, repeated `--env` including a
      value containing `=`, malformed `--env` rejected before any plugin call,
      default output is the bare pane id, `--json` output, missing required
      flag, and an `OpenPane` error.
- [ ] 3.2 (`cmd/swm`) Implement `pane open`.

## 4. swm pane list

- [ ] 4.1 (`cmd/swm`) Write tests: no filters, both filters forwarded, JSON
      array output, empty stream yields `[]` in JSON mode and no output in
      table mode, focused flag preserved, stream error surfaced.
- [ ] 4.2 (`cmd/swm`) Implement `pane list`, including the table renderer.

## 5. swm pane send

- [ ] 5.1 (`cmd/swm`) Write tests: text delivered, submit-only, nothing-to-send
      rejected locally, `--delay-ms` forwarded, text not interpreted,
      `--allow-focused` forwarded, `FAILED_PRECONDITION` resolving to exit
      status 3 with `--allow-focused` named in the message, and every other
      failure resolving to 1.
- [ ] 5.2 (`cmd/swm`) Implement `pane send`.

## 6. swm pane close

- [ ] 6.1 (`cmd/swm`) Write tests: close forwards workspace and pane, an
      already-closed pane succeeds, an error is surfaced.
- [ ] 6.2 (`cmd/swm`) Implement `pane close`.

## 7. Wiring and end-to-end coverage

- [ ] 7.1 (`cmd/swm`) Register the `pane` group in `internal/cli/root.go`.
- [ ] 7.2 (`cmd/swm`) Add integration tests in `cmd/swm/tests/integration`
      driving the real `swm-plugin-session-tmux` against `swm-test-faketmux`:
      open a pane and capture its id, list it back as JSON, send text into an
      unfocused pane, and confirm a seeded focused pane refuses with exit
      status 3.
- [ ] 7.3 (root) Run `task fmt`, `task lint`, and `task test`; all must exit 0.
      No `task proto:gen` and no `task update-nix-vendor-hashes` — this change
      does not touch `proto/`.
