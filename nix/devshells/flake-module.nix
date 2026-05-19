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
          pkgs.flock
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
            export PATH="$PWD/scripts:$PATH"

            ${config.pre-commit.installationScript}

            (
              flock -x 200
              if [[ ! -f go.work ]]; then
                go work init
              fi
              for __go_mod__ in cmd/**/go.mod sdk/**/go.mod plugins/**/go.mod proto/go.mod; do
                go work use "$(dirname "$__go_mod__")"
              done
              for __go_mod__ in go.work cmd/**/go.mod sdk/**/go.mod plugins/**/go.mod proto/go.mod; do
                if [[ "$(${pkgs.gnugrep}/bin/grep '^\(go \)[0-9.]*$' "$__go_mod__")" != "go ${goVersion}" ]]; then
                  ${pkgs.gnused}/bin/sed -e "s:^\(go \)[0-9.]*$:\1${goVersion}:" -i "$__go_mod__"
                fi
              done
            ) 200>"$PWD/go.work.lock"
          '';
      };
    };
}
