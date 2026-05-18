{ inputs, ... }:
{
  imports = [ inputs.treefmt-nix.flakeModule ];

  perSystem = {
    treefmt = {
      settings.global.excludes = [
        ".agent/skills/**/*.md"
        ".agent/workflows/*.md"
        ".claude/commands/**/*.md"
        ".claude/skills/**/*.md"
        ".claude/worktrees/**"
        ".codex/skills/**/*.md"
        ".github/prompts/*.md"
        ".github/skills/**/*.md"
        ".env"
        ".envrc"
        "AGENTS.md"
        "CLAUDE.md"
        "LICENSE"
        "openspec/**/*.md"
        "proto/**/*.pb.go"
        "renovate.json"
      ];

      programs = {
        actionlint.enable = true;
        deadnix.enable = true;
        gofumpt.enable = true;
        nixfmt.enable = true;
        prettier.enable = true;
        statix.enable = true;
      };
    };
  };
}
