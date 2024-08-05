self: {
  config,
  lib,
  pkgs,
  ...
}: let
  inherit (lib) mkEnableOption mkOption mkPackageOption types importJSON mkIf getExe;

  cfg = config.programs.walker;
in {
  options = {
    programs.walker = {
      enable = mkEnableOption "walker";
      package = mkPackageOption self.packages.${pkgs.system} "walker" {
        default = "default";
        pkgsText = "walker.packages.\${pkgs.system}";
      };

      runAsService = mkOption {
        type = types.bool;
        default = false;
        description = "Run as a service";
      };

      config = mkOption {
        type = types.attrs;
        default = importJSON ../config/config.default.json;
        description = "Configuration";
      };
    };
  };

  config = mkIf cfg.enable {
    home.packages = [cfg.package];

    xdg.configFile."walker/config.json".text = mkIf (cfg.config != {}) (builtins.toJSON cfg.config);

    systemd.user.services.walker = mkIf cfg.runAsService {
      Unit.Description = "Walker - Application Runner";
      Install.WantedBy = ["graphical-session.target"];
      Service.ExecStart = "${getExe cfg.package} --gapplication-service";
    };
  };
}
