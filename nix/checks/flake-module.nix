{ self, ... }:
{
  perSystem =
    {
      self',
      config,
      lib,
      pkgs,
      ...
    }:
    {
      checks =
        self'.packages
        // self'.devShells
        // {
          swm-integration-tests =
            let
              version = self.rev or self.dirtyRev;
            in
            pkgs.buildGoModule {
              inherit version;

              vendorHash = config.packages.swm.goModules.outputHash;

              pname = "swm-integration-tests";
              modRoot = "cmd/swm";

              src = lib.fileset.toSource {
                root = ../..;
                fileset = lib.fileset.unions [
                  ../../cmd/swm
                  ../../proto
                  ../../sdk/go
                ];
              };

              buildPhase = ":";
              installPhase = "mkdir -p $out";

              doCheck = true;
              nativeBuildInputs = [ pkgs.git ];

              preCheck = ''
                export XDG_RUNTIME_DIR=$(mktemp -d)
                export HOME=$(mktemp -d)
                export SWM_PLUGIN_VCS_GIT_BIN="${self'.packages.swm-plugin-vcs-git}/bin/swm-plugin-vcs-git"
                export SWM_PLUGIN_SESSION_TMUX_BIN="${self'.packages.swm-plugin-session-tmux}/bin/swm-plugin-session-tmux"
                export SWM_PLUGIN_PICKER_FZF_BIN="${self'.packages.swm-plugin-picker-fzf}/bin/swm-plugin-picker-fzf"
                export SWM_PLUGIN_FORGE_GITHUB_BIN="${self'.packages.swm-plugin-forge-github}/bin/swm-plugin-forge-github"

                # Tests do filepath.Dir(faketmuxBin) and prepend it to PATH, then the
                # session-tmux plugin searches for "tmux" by name. The binaries must live
                # in a shared temp dir under their expected names ("tmux", "fzf").
                fakebins=$(mktemp -d)
                ln -s "${self'.packages.swm-test-faketmux}/bin/faketmux" "$fakebins/tmux"
                ln -s "${self'.packages.swm-test-fakefzf}/bin/fakefzf"   "$fakebins/fzf"
                export SWM_TEST_FAKETMUX_BIN="$fakebins/tmux"
                export SWM_TEST_FAKEFZF_BIN="$fakebins/fzf"
              '';

              checkPhase = ''
                runHook preCheck
                go test -v -count=1 ./tests/integration/...
                runHook postCheck
              '';
            };
        };
    };
}
