{ self, ... }:
{
  perSystem =
    { lib, pkgs, ... }:
    {
      packages.swm-plugin-picker-fzf =
        let
          version =
            let
              rev = self.rev or self.dirtyRev;
              tag = lib.trim (builtins.readFile ./version.txt);
            in
            if tag != "" then tag else rev;

          vendorHash = "sha256-2Ua8hCmWWVCadVRZUtnuE5Gd9iKECpJWe5wfm+JI8jk=";
        in
        pkgs.buildGoModule {
          inherit version vendorHash;

          pname = "swm-plugin-picker-fzf";
          modRoot = "plugins/picker-fzf";

          src = lib.fileset.toSource {
            root = ../../..;
            fileset = lib.fileset.unions [
              ../../../plugins/picker-fzf
              ../../../proto
              ../../../sdk/go
            ];
          };

          doCheck = true;

          postInstall = ''
            mv "$out/bin/picker-fzf" "$out/bin/swm-plugin-picker-fzf"
          '';

          meta = {
            description = "swm fzf picker plugin";
            homepage = "https://github.com/kalbasit/swm";
            license = lib.licenses.mit;
            mainProgram = "swm-plugin-picker-fzf";
            maintainers = [ lib.maintainers.kalbasit ];
          };
        };
    };
}
