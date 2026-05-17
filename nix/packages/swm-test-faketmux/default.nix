{ self, ... }:
{
  perSystem =
    {
      config,
      lib,
      pkgs,
      ...
    }:
    {
      packages.swm-test-faketmux =
        let
          version =
            let
              rev = self.rev or self.dirtyRev;
              tag = lib.trim (builtins.readFile ./version.txt);
            in
            if tag != "" then tag else rev;
        in
        pkgs.buildGoModule {
          inherit version;

          vendorHash = config.packages.swm-plugin-session-tmux.goModules.outputHash;

          pname = "swm-test-faketmux";
          modRoot = "plugins/session-tmux";
          subPackages = [ "internal/session/testdata/faketmux" ];

          src = lib.fileset.toSource {
            root = ../../..;
            fileset = lib.fileset.unions [
              ../../../plugins/session-tmux
              ../../../proto
              ../../../sdk/go
            ];
          };

          doCheck = false;

          meta = {
            description = "fake tmux binary used by swm integration tests";
            homepage = "https://github.com/kalbasit/swm";
            license = lib.licenses.mit;
            mainProgram = "faketmux";
            maintainers = [ lib.maintainers.kalbasit ];
          };
        };
    };
}
