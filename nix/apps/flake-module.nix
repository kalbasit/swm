_: {
  perSystem =
    { config, lib, ... }:
    let
      swmApp = {
        type = "app";
        program = lib.getExe config.packages.swm;
      };
    in
    {
      apps.swm = swmApp;
      apps.default = swmApp;
    };
}
