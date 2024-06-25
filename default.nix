let
  pkgs = import (import ./nix/pkgs.nix) {};
in

{
  buildGoModule ? pkgs.buildGoModule
, nix-gitignore ? pkgs.nix-gitignore
, version ? "dev"
, fzf ? pkgs.fzf
, git ? pkgs.git
, tmux ? pkgs.tmux
, procps ? pkgs.procps
, installShellFiles ? pkgs.installShellFiles
, lib ? pkgs.lib
, makeWrapper ? pkgs.makeWrapper
}:

buildGoModule rec {
  inherit version;

  pname = "swm";

  src = nix-gitignore.gitignoreSource [ ".git" ".envrc" ".travis.yml" ".gitignore" ] ./.;

  vendorHash = null;

  ldflags = ["-X=github.com/kalbasit/swm/cmd.version=${version}"];

  nativeBuildInputs = [ fzf git tmux procps installShellFiles makeWrapper ];

  postInstall = ''
    for shell in bash zsh fish; do
      $out/bin/swm auto-complete $shell > swm.$shell
      installShellCompletion swm.$shell
    done

    $out/bin/swm gen-doc man --path ./man
    installManPage man/*.7

    wrapProgram $out/bin/swm --prefix PATH : ${lib.makeBinPath [ fzf git tmux procps ]}
  '';

  doCheck = true;
  preCheck = ''
    export HOME=$NIX_BUILD_TOP/home
    mkdir -p $HOME

    git config --global user.email "nix-test@example.com"
    git config --global user.name "Nix Test"
  '';

  meta = with lib; {
    homepage = "https://github.com/kalbasit/swm";
    description = "swm (Story-based Workflow Manager) is a Tmux session manager specifically designed for Story-based development workflow";
    license = licenses.mit;
    maintainers = [ maintainers.kalbasit ];
  };
}
