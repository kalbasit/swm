{ inputs, ... }:

{
  imports = [
    inputs.git-hooks-nix.flakeModule
  ];

  perSystem =
    { pkgs, ... }:
    {
      pre-commit.check.enable = false;
      pre-commit.settings.hooks = {
        check-merge-conflicts.enable = true;
        deadnix.enable = true;
        golangci-lint = {
          enable = true;
          # Loop over all modules (dirs with go.mod) and lint each from within that module.
          # This is the only way golangci-lint works: it must run from a module directory.
          # Wrapped with bash -c because language:unsupported runs the entry as a direct
          # executable, which cannot handle shell builtins like `for`.
          # Use $dir/go.mod (not ${dir}go.mod) to avoid Nix string interpolation.
          entry = "${pkgs.bash}/bin/bash -c 'for dir in apps/*/; do if [ -f \"$dir/go.mod\" ]; then (cd \"$dir\" && ${pkgs.golangci-lint}/bin/golangci-lint run ./...); fi; done'";
        };
        no-commit-to-branch.enable = true;
        no-commit-to-branch.settings.branch = [ "main" ];
        nixfmt-rfc-style.enable = true;
        prettier = {
          enable = true;

          excludes = [
            "\\.agent/skills/.*\\.md$"
            "\\.agent/workflows/[^/]*\\.md$"
            "\\.claude/commands/.*\\.md$"
            "\\.claude/skills/.*\\.md$"
            "\\.codex/skills/.*\\.md$"
            "\\.github/prompts/[^/]*\\.md$"
            "\\.github/skills/.*\\.md$"
            "AGENTS\\.md$"
            "CLAUDE\\.md$"
            "openspec/.*\\.md$"
          ];
        };
        statix.enable = true;
        trim-trailing-whitespace.enable = true;
      };
    };
}
