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
      packages.swm-test-fakefzf =
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

          vendorHash = config.packages.swm-plugin-picker-fzf.goModules.outputHash;

          pname = "swm-test-fakefzf";
          modRoot = "plugins/picker-fzf";
          subPackages = [ "internal/picker/testdata/fakefzf" ];

          src = lib.fileset.toSource {
            root = ../../..;
            fileset = lib.fileset.unions [
              ../../../plugins/picker-fzf
              ../../../proto
              ../../../sdk/go
            ];
          };

          doCheck = false;

          meta = {
            description = "fake fzf binary used by swm integration tests";
            homepage = "https://github.com/kalbasit/swm";
            license = lib.licenses.mit;
            mainProgram = "fakefzf";
            maintainers = [ lib.maintainers.kalbasit ];
          };
        };
    };
}
