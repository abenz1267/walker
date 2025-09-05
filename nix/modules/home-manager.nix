{ self, elephant }:
{
  config,
  lib,
  pkgs,
  ...
}:
let
  inherit (lib.modules) mkIf mkDefault mkMerge;
  inherit (lib.options) mkOption mkEnableOption mkPackageOption;
  inherit (lib.trivial) importTOML;
  inherit (lib.meta) getExe;
  inherit (lib.types)
    nullOr
    bool
    path
    ;

  tomlFormat = pkgs.formats.toml { };
  cfg = config.programs.walker;
in
{
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
        type = nullOr path;
        default = null;
        description = ''
          The custom theme used by walker. Setting this option overrides config.theme.
        '';
      };

      elephant = mkOption {
        inherit (tomlFormat) type;
        default = { };
        description = "Configuration for elephant";
      };
    };
  };

  config = mkIf cfg.enable (mkMerge [
    {
      programs.elephant = mkMerge [
        { enable = true; }
        cfg.elephant
      ];

      home.packages = [ cfg.package ];

      xdg.configFile."walker/config.toml".source = mkIf (cfg.config != { }) (
        tomlFormat.generate "walker-config.toml" cfg.config
      );

      systemd.user.services.walker = mkIf cfg.runAsService {
        Unit = {
          Description = "Walker - Application Runner";
          ConditionEnvironment = "WAYLAND_DISPLAY";
          After = [
            "graphical-session.target"
            "elephant.service"
          ];
          Requires = [ "elephant.service" ];
          PartOf = [ "graphical-session.target" ];
        };
        Service = {
          ExecStart = "${getExe cfg.package} --gapplication-service";
          Restart = "on-failure";
        };
        Install.WantedBy = [ "graphical-session.target" ];
      };
    }

    (mkIf (cfg.theme != null) {
      programs.walker.config.theme = mkDefault (builtins.baseNameOf cfg.theme);
      xdg.configFile."walker/themes/${builtins.baseNameOf cfg.theme}".source = cfg.theme;
    })
  ]);
}
