{ self, ... }:
{
  perSystem =
    { lib, pkgs, ... }:
    {
      packages.swm =
        let
          version =
            let
              rev = self.rev or self.dirtyRev;
              tag = lib.trim (builtins.readFile ./version.txt);
            in
            if tag != "" then tag else rev;

          vendorHash = "sha256-3QOG6+AnzkgJAqrKwYKdtbNiDssIsdA57vmLzM2XcZc=";
        in
        pkgs.buildGoModule {
          inherit version vendorHash;

          pname = "swm";
          modRoot = "cmd/swm";

          src = lib.fileset.toSource {
            root = ../../..;
            fileset = lib.fileset.unions [
              # Exclude integration tests: they compile all plugins at runtime using
              # the Go workspace, which is incompatible with single-module sandboxing.
              (lib.fileset.difference ../../../cmd/swm ../../../cmd/swm/tests/integration)
              ../../../proto
              ../../../sdk/go
            ];
          };

          doCheck = true;
          nativeBuildInputs = [
            pkgs.git
            pkgs.installShellFiles
          ];

          preCheck = ''
            export XDG_RUNTIME_DIR=$(mktemp -d)
            export HOME=$(mktemp -d)
          '';

          postInstall = lib.optionalString (pkgs.stdenv.hostPlatform == pkgs.stdenv.buildPlatform) ''
            installShellCompletion --cmd swm \
              --bash <($out/bin/swm completion bash) \
              --zsh  <($out/bin/swm completion zsh)  \
              --fish <($out/bin/swm completion fish)
          '';

          meta = {
            description = "Story-based Workflow Manager: per-story git worktrees and multiplexer sessions";
            homepage = "https://github.com/kalbasit/swm";
            license = lib.licenses.mit;
            mainProgram = "swm";
            maintainers = [ lib.maintainers.kalbasit ];
          };
        };
    };
}
