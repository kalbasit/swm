_: {
  perSystem =
    { config, pkgs, ... }:
    {
      packages.swm-full = pkgs.symlinkJoin {
        name = "swm-full";
        paths = [
          config.packages.swm
          config.packages.swm-plugin-forge-github
          config.packages.swm-plugin-picker-fzf
          config.packages.swm-plugin-session-tmux
          config.packages.swm-plugin-vcs-git
        ];
        meta = {
          description = "swm with all plugins";
          mainProgram = "swm";
        };
      };
    };
}
