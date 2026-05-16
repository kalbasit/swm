{ ... }:
{
  imports = [
    ./swm
    ./swm-plugin-forge-github
    ./swm-plugin-picker-fzf
    ./swm-plugin-session-tmux
    ./swm-plugin-vcs-git
    ./swm-full
  ];

  perSystem =
    { config, ... }:
    {
      packages.default = config.packages.swm-full;
    };
}
