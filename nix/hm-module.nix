inputs: {
  config,
  lib,
  pkgs,
  ...
}: let
  inherit (pkgs.stdenv.hostPlatform) system;
  defaultConfig = builtins.fromJSON (builtins.readFile ../config/config.default.json);
  defaultStyle = builtins.readFile ../ui/themes/style.default.css;
  cfg = config.programs.walker;
in {
  imports = [
    (lib.mkRenamedOptionModule [ "programs" "walker" "enabled" ] [ "programs" "walker" "enable" ])
  ];
  options = {
    programs.walker = with lib; {
      enable = mkEnableOption "walker";
      runAsService = mkOption {
        type = types.bool;
        default = false;
        description = "Run as service";
      };
      style = mkOption {
        type = types.str;
        default = defaultStyle;
        description = "Theming";
      };
      config = mkOption {
        type = types.attrs;
        default = defaultConfig;
        description = "Configuration";
      };
    };
  };

  config = lib.mkIf cfg.enable {
    home.packages = [inputs.self.packages.${system}.walker];

    xdg.configFile."walker/config.json".text = builtins.toJSON (lib.recursiveUpdate defaultConfig config.programs.walker.config);
    xdg.configFile."walker/style.css".text = config.programs.walker.style;

    systemd.user.services.walker = lib.mkIf cfg.runAsService {
      Unit = {
        Description = "Walker - Application Runner";
      };
      Install = {
        WantedBy = [
          "graphical-session.target"
        ];
      };
      Service = {
        ExecStart = "${inputs.self.packages.${system}.walker}/bin/walker --gapplication-service";
      };
    };
  };
}
