## 1. Verify Cobra completion subcommand (cmd/swm)

- [x] 1.1 Run `swm completion bash` against a locally built binary and confirm it exits 0 with non-empty output
- [x] 1.2 If the subcommand is absent (e.g. `DisableDefaultCmd` is set in `root.go`), call `root.InitDefaultCompletionCmd()` or remove the option to re-enable it

## 2. Tests (cmd/swm)

- [x] 2.1 Write a table-driven unit test in `cmd/swm/internal/cli/` (e.g. `completion_test.go`) that invokes the root command with `completion <shell>` for each of `bash`, `zsh`, `fish`, `powershell` and asserts exit code 0 and non-empty stdout
- [x] 2.2 Run `task test` in `cmd/swm` and confirm the new tests pass

## 3. Nix packaging (nix/packages/swm)

- [x] 3.1 Add `pkgs.installShellFiles` to `nativeBuildInputs` in `nix/packages/swm/default.nix` (alongside `pkgs.git`)
- [x] 3.2 Add a `postInstall` hook that generates and installs completions:
  ```nix
  postInstall = ''
    installShellCompletion --cmd swm \
      --bash <($out/bin/swm completion bash) \
      --zsh  <($out/bin/swm completion zsh)  \
      --fish <($out/bin/swm completion fish)
  '';
  ```
- [x] 3.3 Build the package with `nix build .#swm` and verify completion files exist under `result/share/bash-completion/`, `result/share/zsh/site-functions/`, and `result/share/fish/vendor_completions.d/`

## 4. Final verification

- [x] 4.1 Run `task fmt` — confirm exit 0 with no diff
- [x] 4.2 Run `task lint` — confirm exit 0
- [x] 4.3 Run `task test` — confirm all tests pass
