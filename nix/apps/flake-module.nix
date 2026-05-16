_: {
  perSystem =
    { config, lib, ... }:
    let
      swmApp = {
        type = "app";
        program = lib.getExe config.packages.swm-full;
      };
    in
    {
      apps.swm = swmApp;
      apps.default = swmApp;
    };
}
