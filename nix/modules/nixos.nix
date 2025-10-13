{
  self,
  elephant,
}: {
  config,
  lib,
  pkgs,
  ...
}: let
  inherit (lib.modules) mkIf mkMerge;
  inherit (lib.options) mkOption mkEnableOption mkPackageOption;
  inherit (lib.trivial) importTOML;
  inherit (lib.meta) getExe;
  inherit (lib.types) nullOr bool;
  inherit (lib) optional types mapAttrs' mapAttrsToList nameValuePair mkDefault literalExpression;

  cfg = config.programs.walker;

  tomlFormat = pkgs.formats.toml {};
in {
  imports = [
    elephant.nixosModules.default
  ];

  options = {
    programs.walker = {
      enable = mkEnableOption "walker";

      package = mkPackageOption self.packages.${pkgs.stdenv.system} "walker" {
        default = "default";
        pkgsText = "walker.packages.\${pkgs.stdenv.system}";
      };

      runAsService = mkOption {
        type = bool;
        default = false;
        description = "Run walker as a service for faster launch times.";
      };

      config = mkOption {
        inherit (tomlFormat) type;
        default = importTOML ../../resources/config.toml;
        defaultText = "importTOML ../../resources/config.toml";
        description = ''
          Configuration options for walker.

          See the default configuration for available options: <https://github.com/abenz1267/walker/blob/master/resources/config.toml>
        '';
      };

      themes = mkOption {
        type = types.attrsOf (types.submodule {
          options = {
            style = mkOption {
              type = types.lines;
              default = "";
              description = ''
                The GTK CSS stylesheet used by this theme.

                See the default style sheet for available classes: <https://github.com/abenz1267/walker/blob/master/resources/themes/default/style.css>
              '';
            };

            layouts = mkOption {
              type = types.attrsOf types.str;
              default = {};
              description = ''
                The GTK xml layouts used for each provider.

                See the default layouts for correct names and structure: <https://github.com/abenz1267/walker/tree/master/resources/themes/default>
              '';
            };
          };
        });
        default = {};
        example = literalExpression ''
          themes."your-theme-name" = {
            style = \'\'
              /* CSS */
            \'\';
            layouts = {
              "layout" = \'\'
                <!-- XML Layout -->
              \'\';
              "item_calc" = \'\'
                <!-- XML Layout -->
              \'\';
            };
          };
        '';
        description = "Set of themes usable by walker";
      };

      elephant = mkOption {
        inherit (tomlFormat) type;
        default = {};
        description = "Configuration for elephant";
      };

      # The `theme` option will soon be deprecated please use the above `themes` option instead.
      theme = mkOption {
        type = with types;
          nullOr (submodule {
            options = {
              name = mkOption {
                type = types.str;
                default = "nixos";
                description = "The theme name.";
              };

              style = mkOption {
                type = lines;
                default = "";
                description = "The styling of the theme, written in GTK CSS.";
              };
            };
          });
        default = null;
        description = "The custom theme used by walker. Setting this option overrides `programs.walker.config.theme`.";
      };
    };
  };

  config = mkIf cfg.enable {
    warnings = optional (cfg.theme != null) ''
      The option `programs.walker.theme` is deprecated. Please migrate to `programs.walker.themes` instead.

      From

      programs.walker.theme = {
        name = "${cfg.theme.name}";
        style = " /* CSS */ ";
      };

      to

      programs.walker = {
        config.theme = "${cfg.theme.name}";
        themes."${cfg.theme.name}" = {
          style = " /* CSS */ ";
        };
      }
    '';

    services.elephant = mkMerge [
      {enable = true;}
      cfg.elephant
    ];

    environment.systemPackages = [cfg.package];

    # deprecated functions start
    programs.walker = mkIf (cfg.theme != null) {
      themes = {
        "${cfg.theme.name}" = mkDefault {
          style = cfg.theme.style;
        };
      };

      config.theme = mkDefault cfg.theme.name;
    };
    # deprecated functions end

    environment.etc = mkMerge [
      # Generate config file
      (
        mkIf (cfg.config != {}) {
          "xdg/walker/config.toml".source = tomlFormat.generate "walker-config.toml" cfg.config;
        }
      )

      # Generate theme files
      (
        mkMerge (
          mapAttrsToList
          (
            themeName: theme:
              {
                "xdg/walker/themes/${themeName}/style.css".text = theme.style;
              }
              // (
                mapAttrs'
                (
                  layoutName: layoutContent:
                    nameValuePair "xdg/walker/themes/${themeName}/${layoutName}.xml" {
                      text = layoutContent;
                    }
                )
                theme.layouts
              )
          )
          cfg.themes
        )
      )
    ];

    systemd.services.walker = mkIf cfg.runAsService {
      description = "Walker - Application Runner";
      unitConfig = {
        ConditionEnvironment = "WAYLAND_DISPLAY";
      };
      after = [
        "graphical-session.target"
        "elephant.service"
      ];
      requires = ["elephant.service"];
      partOf = ["graphical-session.target"];
      wantedBy = ["graphical-session.target"];
      serviceConfig = {
        ExecStart = "${getExe cfg.package} --gapplication-service";
        Restart = "on-failure";
      };
    };
  };
}
