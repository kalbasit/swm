{
  perSystem =
    {
      config,
      pkgs,
      ...
    }:

    {
      devShells.default = pkgs.mkShell {
        buildInputs = [
          pkgs.buf
          pkgs.delve
          pkgs.go
          pkgs.go-task
          pkgs.golangci-lint
          pkgs.pre-commit
          pkgs.protoc-gen-go
          pkgs.protoc-gen-go-grpc
        ];

        _GO_VERSION = "${pkgs.go.version}";

        # Disable hardening for fortify otherwize it's not possible to use Delve.
        hardeningDisable = [ "fortify" ];

        shellHook =
          let
            goVersion = pkgs.go.version;
          in
          ''
            ${config.pre-commit.installationScript}

            if [[ ! -f go.work ]]; then
              go work init
              go work use ./cmd/swm
              go work use ./plugins/forge-github
              go work use ./plugins/picker-fzf
              go work use ./plugins/session-tmux
              go work use ./plugins/vcs-git
              go work use ./proto
              go work use ./sdk/go
            fi

            (
              for __go_mod__ in go.work cmd/**/go.mod sdk/**/go.mod plugins/**/go.mod; do
                if [[ "$(${pkgs.gnugrep}/bin/grep '^\(go \)[0-9.]*$' "$__go_mod__")" != "go ${goVersion}" ]]; then
                  ${pkgs.gnused}/bin/sed -e "s:^\(go \)[0-9.]*$:\1${goVersion}:" -i "$__go_mod__"
                fi
              done
            )
          '';
      };
    };
}
