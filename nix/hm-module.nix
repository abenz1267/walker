self: {
  config,
  lib,
  pkgs,
  ...
}: let
  inherit (lib) mkEnableOption mkOption mkPackageOption importTOML mkIf getExe mkForce mkMerge;
  inherit (lib.types) bool nullOr submodule lines;

  tomlFormat = pkgs.formats.toml {};

  themeType = submodule {
    options = {
      layout = mkOption {
        inherit (tomlFormat) type;
        default = {};
        description = ''
          The layout of the theme.

          See <https://github.com/abenz1267/walker/wiki/Theming> for the full list of options.
        '';
      };

      style = mkOption {
        type = lines;
        default = "";
        description = "The styling of the theme, written in GTK CSS.";
      };
    };
  };
  themeName = "home-manager";

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
        type = bool;
        default = false;
        description = "Run walker as a service for faster startup times.";
      };

      config = mkOption {
        inherit (tomlFormat) type;
        default = importTOML ../internal/config/config.default.toml;
        description = ''
          Configuration written to `$XDG_CONFIG_HOME/walker/config.toml`.

          See <https://github.com/abenz1267/walker/wiki/Basic-Configuration> for the full list of options.
        '';
      };

      theme = mkOption {
        type = nullOr themeType;
        default = null;
        description = "The custom theme used by walker. Setting this option overrides `config.theme`.";
      };
    };
  };

  config = mkIf cfg.enable (mkMerge [
    {
      home.packages = [cfg.package];

      xdg.configFile."walker/config.toml".source = mkIf (cfg.config != {}) (tomlFormat.generate "walker-config.toml" cfg.config);

      systemd.user.services.walker = mkIf cfg.runAsService {
        Unit.Description = "Walker - Application Runner";
        Install.WantedBy = ["graphical-session.target"];
        Service = {
          ExecStart = "${getExe cfg.package} --gapplication-service";
          Restart = "on-failure";
        };
      };
    }

    (mkIf (cfg.theme != null) {
      programs.walker.config.theme = mkForce themeName;

      xdg.configFile = {
        "walker/themes/${themeName}.toml".source = tomlFormat.generate "walker-themes-${themeName}.toml" cfg.theme.layout;
        "walker/themes/${themeName}.css".text = cfg.theme.style;
      };
    })
  ]);
}
