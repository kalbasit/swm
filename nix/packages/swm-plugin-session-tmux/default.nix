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

          vendorHash = "sha256-p6s9Hp4f6/OZyHtqCZaHqzhpG6sItuX1yLOsfS+b5dw=";
        in
        pkgs.buildGoModule {
          inherit version vendorHash;

          pname = "swm-plugin-session-tmux";
          modRoot = "plugins/session-tmux";

          # The binary must report the version this derivation was built with;
          # buildVersion has no other source.
          ldflags = [
            "-X github.com/kalbasit/swm/plugins/session-tmux/internal/session.buildVersion=${version}"
          ];

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
