{ self, ... }:
{
  perSystem =
    { lib, pkgs, ... }:
    {
      packages.swm-plugin-forge-github =
        let
          version =
            let
              rev = self.rev or self.dirtyRev;
              tag = lib.trim (builtins.readFile ./version.txt);
            in
            if tag != "" then tag else rev;

          vendorHash = "sha256-BqAfhfTZgowSPp3zNCFV1N/4nrjbkvNvj/jGglpmlbk=";
        in
        pkgs.buildGoModule {
          inherit version vendorHash;

          pname = "swm-plugin-forge-github";
          modRoot = "plugins/forge-github";

          # The binary must report the version this derivation was built with;
          # buildVersion has no other source.
          ldflags = [
            "-X github.com/kalbasit/swm/plugins/forge-github/internal/forge.buildVersion=${version}"
          ];

          src = lib.fileset.toSource {
            root = ../../..;
            fileset = lib.fileset.unions [
              ../../../plugins/forge-github
              ../../../proto
              ../../../sdk/go
            ];
          };

          doCheck = true;

          postInstall = ''
            mv "$out/bin/forge-github" "$out/bin/swm-plugin-forge-github"
          '';

          meta = {
            description = "swm GitHub forge plugin";
            homepage = "https://github.com/kalbasit/swm";
            license = lib.licenses.mit;
            mainProgram = "swm-plugin-forge-github";
            maintainers = [ lib.maintainers.kalbasit ];
          };
        };
    };
}
