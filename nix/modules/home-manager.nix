{self, elephant}: {
  config,
  lib,
  pkgs,
  ...
}: let
  inherit (lib.modules) mkIf mkDefault mkMerge;
  inherit (lib.options) mkOption mkEnableOption mkPackageOption;
  inherit (lib.trivial) importTOML;
  inherit (lib.meta) getExe;
  inherit (lib.types) nullOr bool submodule lines;

  tomlFormat = pkgs.formats.toml {};

  theme = {
    name = "nixos";
    type = submodule {
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
  };

  cfg = config.programs.walker;
in {
  imports = [
    elephant.homeManagerModules.default
  ];

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
        default = importTOML ../../resources/config.toml;
        defaultText = "importTOML ../../resources/config.toml";
        description = ''
          Configuration written to {file}`$XDG_CONFIG_HOME/walker/config.toml`.

          See <https://github.com/abenz1267/walker/wiki/Basic-Configuration> for the full list of options.
        '';
      };

      theme = mkOption {
        type = nullOr theme.type;
        default = null;
        description = "The custom theme used by walker. Setting this option overrides `config.theme`.";
      };
    };
  };

  config = mkIf cfg.enable (mkMerge [
    {
      programs.elephant = {
        enable = true;
        autoStart = true;
      };

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
      programs.walker.config.theme = mkDefault theme.name;

      xdg.configFile = {
        "walker/themes/${theme.name}.toml".source = tomlFormat.generate "walker-themes-${theme.name}.toml" cfg.theme.layout;
        "walker/themes/${theme.name}.css".text = cfg.theme.style;
      };
    })
  ]);
}