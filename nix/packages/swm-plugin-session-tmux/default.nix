{ self, ... }:
{
  perSystem =
    { lib, pkgs, ... }:
    {
      packages.swm-plugin-session-tmux =
        let
          version =
            let
              rev = self.rev or self.dirtyRev;
              tag = lib.trim (builtins.readFile ./version.txt);
            in
            if tag != "" then tag else rev;

          vendorHash = "sha256-jRVqpPiiN0k370bj7nkKM7gZJdMwBmzrzCihgLv7pUY=";
        in
        pkgs.buildGoModule {
          inherit version vendorHash;

          pname = "swm-plugin-session-tmux";
          modRoot = "plugins/session-tmux";

          src = lib.fileset.toSource {
            root = ../../..;
            fileset = lib.fileset.unions [
              ../../../plugins/session-tmux
              ../../../proto
              ../../../sdk/go
            ];
          };

          doCheck = true;

          postInstall = ''
            mv "$out/bin/session-tmux" "$out/bin/swm-plugin-session-tmux"
          '';

          meta = {
            description = "swm tmux session plugin";
            homepage = "https://github.com/kalbasit/swm";
            license = lib.licenses.mit;
            mainProgram = "swm-plugin-session-tmux";
            maintainers = [ lib.maintainers.kalbasit ];
          };
        };
    };
}
