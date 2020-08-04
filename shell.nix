{ pkgs ? import (import ./nix/pkgs.nix) {}, home ? builtins.getEnv "HOME" }:

with pkgs;

let
  swm = callPackage ./default.nix {};
in mkShell {
  buildInputs = [ go gotools ] ++ swm.buildInputs ++ swm.nativeBuildInputs;

  GOPATH = "${home}/.cache/go";
  GO111MODULE = "on";

  shellHook = builtins.concatStringsSep "\n" [
    # the shell hook is exporting an invalid cert file. Get rid of this cert
    # file if it does not exist.
    ''
      if [ -n "''${SSL_CERT_FILE:-}" ] || [ -n "''${NIX_SSL_CERT_FILE:-}" ]; then
        local ca=""
        if [ -n "''${SSL_CERT_FILE:-}" ] && [ -f "$SSL_CERT_FILE" ]; then
          ca="$SSL_CERT_FILE"
        elif [ -n "''${NIX_SSL_CERT_FILE:-}" ] && [ -f "$NIX_SSL_CERT_FILE" ]; then
          ca="$NIX_SSL_CERT_FILE"
        elif [ -f"/etc/ssl/certs/ca-bundle.crt" ]; then
          ca="/etc/ssl/certs/ca-bundle.crt"
        fi
        if [ -f "$ca" ]; then
            export SSL_CERT_FILE="$ca"
            export NIX_SSL_CERT_FILE="$ca"
          fi
      else
        unset SSL_CERT_FILE NIX_SSL_CERT_FILE
      fi
    ''
  ];
}
