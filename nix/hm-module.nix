self: {
  config,
  lib,
  pkgs,
  ...
}: let
  inherit (lib) mkEnableOption mkOption mkPackageOption types mkIf;

  packages = self.packages.${pkgs.stdenv.hostPlatform.system};

  cfg = config.programs.walker;
in {
  options = {
    programs.walker = {
      enable = mkEnableOption "walker";
      package = mkPackageOption packages "walker" {
        default = "default";
        pkgsText = "walker.packages.\${pkgs.stdenv.hostPlatform.system}";
      };
      runAsService = mkOption {
        type = types.bool;
        default = false;
        description = "Run as a service";
      };

      config = mkOption {
        type = types.attrs;
        default = builtins.fromJSON (builtins.readFile ../config/config.default.json);
        description = "Configuration";
      };
      style = mkOption {
        type = types.str;
        default = builtins.readFile ../ui/themes/style.default.css;
        description = "Theming";
      };
    };
  };

  config = mkIf cfg.enable {
    home.packages = [cfg.package];

    xdg.configFile = {
      "walker/config.json".text = mkIf (cfg.config != { }) (builtins.toJSON cfg.config);
      "walker/style.css".text = mkIf (cfg.style != { }) cfg.style;
    };

    systemd.user.services.walker = mkIf cfg.runAsService {
      Unit = {
        Description = "Walker - Application Runner";
      };
      Install = {
        WantedBy = [
          "graphical-session.target"
        ];
      };
      Service = {
        ExecStart = "${cfg.package}/bin/walker --gapplication-service";
      };
    };
  };
}
