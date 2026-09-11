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

          vendorHash = "sha256-D0I6anisIMOZlMxMwuuSI8SuJ+zd1TvOwb82TFH74J8=";
        in
        pkgs.buildGoModule {
          inherit version vendorHash;

          pname = "swm";
          modRoot = "cmd/swm";

          # The binary must report the version this derivation was built with.
          # Without this the version is only a derivation-name attribute and
          # every build reports whatever default the source carries.
          ldflags = [
            "-X github.com/kalbasit/swm/cmd/swm/internal/version.version=${version}"
          ];

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
            # Guard the ldflags above: they are easy to drop and the loss is
            # silent, which is exactly how the version came to be frozen at a
            # stale default. Only runnable when we can execute what we built.
            reported="$($out/bin/swm --version)"
            if [[ "$reported" != "swm version ${version}" ]]; then
              echo "swm --version reported '$reported', expected 'swm version ${version}'" >&2
              exit 1
            fi

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
