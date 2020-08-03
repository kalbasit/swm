{ pkgs ? import (import ./nix/pkgs.nix) {} }:

with pkgs;

buildGoModule rec {
  pname = "swm";
  version = "2020-08-03";

  src = nix-gitignore.gitignoreSource [ ".git" ".envrc" ".travis.yml" ".gitignore" ] ./.;

  vendorSha256 = null;

  buildFlagsArray = [ "-ldflags=" "-X=main.version=${version}" ];

  buildInputs = [ fzf git tmux procps ];

  nativeBuildInputs = [ installShellFiles ];

  postInstall = ''
    for shell in bash zsh fish; do
      $out/bin/swm auto-complete $shell > swm.$shell
      installShellCompletion swm.$shell
    done

    $out/bin/swm gen-doc man --path ./man
    installManPage man/*.7
  '';

  meta = with lib; {
    homepage = "https://github.com/kalbasit/swm";
    description = "swm (Story-based Workflow Manager) is a Tmux session manager specifically designed for Story-based development workflow";
    license = licenses.mit;
    maintainers = [ maintainers.kalbasit ];
  };
}
