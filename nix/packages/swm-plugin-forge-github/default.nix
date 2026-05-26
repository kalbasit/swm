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

          vendorHash = "sha256-daGfl2RfMia0m+LJ5Vbab7BTmZr3nsYp1A5tdim0M8Y=";
        in
        pkgs.buildGoModule {
          inherit version vendorHash;

          pname = "swm-plugin-forge-github";
          modRoot = "plugins/forge-github";

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
