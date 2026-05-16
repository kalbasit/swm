{ self, ... }:
{
  perSystem =
    { lib, pkgs, ... }:
    {
      packages.swm-plugin-vcs-git =
        let
          version =
            let
              rev = self.rev or self.dirtyRev;
              tag = lib.trim (builtins.readFile ./version.txt);
            in
            if tag != "" then tag else rev;

          vendorHash = "sha256-nzPUygfJm4+mTZMO/cETYbIAiIK22YSFr93yj+z5pDk=";
        in
        pkgs.buildGoModule {
          inherit version vendorHash;

          pname = "swm-plugin-vcs-git";
          modRoot = "plugins/vcs-git";

          src = lib.fileset.toSource {
            root = ../../..;
            fileset = lib.fileset.unions [
              ../../../plugins/vcs-git
              ../../../proto
              ../../../sdk/go
            ];
          };

          doCheck = true;
          nativeBuildInputs = [ pkgs.git ];

          postInstall = ''
            mv "$out/bin/vcs-git" "$out/bin/swm-plugin-vcs-git"
          '';

          meta = {
            description = "swm git VCS plugin";
            homepage = "https://github.com/kalbasit/swm";
            license = lib.licenses.mit;
            mainProgram = "swm-plugin-vcs-git";
            maintainers = [ lib.maintainers.kalbasit ];
          };
        };
    };
}
