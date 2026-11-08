{
  config,
  lib,
  pkgs,
  ...
}:
let
  cfg = config.programs.snitchx;

  themes = [
    "ansi"
    "catppuccin-mocha"
    "catppuccin-macchiato"
    "catppuccin-frappe"
    "catppuccin-latte"
    "gruvbox-dark"
    "gruvbox-light"
    "dracula"
    "nord"
    "tokyo-night"
    "tokyo-night-storm"
    "tokyo-night-light"
    "solarized-dark"
    "solarized-light"
    "one-dark"
    "mono"
    "auto"
  ];

  defaultFields = [
    "pid"
    "process"
    "user"
    "proto"
    "state"
    "laddr"
    "lport"
    "raddr"
    "rport"
  ];

  tomlFormat = pkgs.formats.toml { };

  settingsType = lib.types.submodule {
    freeformType = tomlFormat.type;

    options = {
      defaults = lib.mkOption {
        type = lib.types.submodule {
          freeformType = tomlFormat.type;

          options = {
            interval = lib.mkOption {
              type = lib.types.str;
              default = "1s";
              example = "2s";
              description = "Default refresh interval for watch/stats/trace commands.";
            };

            numeric = lib.mkOption {
              type = lib.types.bool;
              default = false;
              description = "Disable name/service resolution by default.";
            };

            fields = lib.mkOption {
              type = lib.types.listOf lib.types.str;
              default = defaultFields;
              example = [ "pid" "process" "proto" "state" "laddr" "lport" ];
              description = "Default fields to display.";
            };

            theme = lib.mkOption {
              type = lib.types.enum themes;
              default = "ansi";
              description = ''
                Color theme for the TUI. "ansi" inherits terminal colors.
              '';
            };

            units = lib.mkOption {
              type = lib.types.enum [ "auto" "si" "iec" ];
              default = "auto";
              description = "Default units for byte display.";
            };

            color = lib.mkOption {
              type = lib.types.enum [ "auto" "always" "never" ];
              default = "auto";
              description = "Default color mode.";
            };

            resolve = lib.mkOption {
              type = lib.types.bool;
              default = true;
              description = "Enable name resolution by default.";
            };

            dns_cache = lib.mkOption {
              type = lib.types.bool;
              default = true;
              description = "Enable DNS caching.";
            };

            ipv4 = lib.mkOption {
              type = lib.types.bool;
              default = false;
              description = "Filter to IPv4 only by default.";
            };

            ipv6 = lib.mkOption {
              type = lib.types.bool;
              default = false;
              description = "Filter to IPv6 only by default.";
            };

            no_headers = lib.mkOption {
              type = lib.types.bool;
              default = false;
              description = "Omit headers in output by default.";
            };

            output_format = lib.mkOption {
              type = lib.types.enum [ "table" "json" "csv" ];
              default = "table";
              description = "Default output format.";
            };

            sort_by = lib.mkOption {
              type = lib.types.str;
              default = "";
              example = "pid";
              description = "Default sort field.";
            };
          };
        };
        default = { };
        description = "Default settings for snitchx commands.";
      };
    };
  };
in
{
  options.programs.snitchx = {
    enable = lib.mkEnableOption "snitchx, cross-platform security inspection";

    package = lib.mkPackageOption pkgs "snitchx" { };

    settings = lib.mkOption {
      type = settingsType;
      default = { };
      example = lib.literalExpression ''
        {
          defaults = {
            theme = "catppuccin-mocha";
            interval = "2s";
            resolve = true;
          };
        }
      '';
      description = ''
        Configuration written to {file}`$XDG_CONFIG_HOME/snitchx/snitchx.toml`.

        See <https://github.com/ahzslanh/snitchx> for available options.
      '';
    };
  };

  config = lib.mkIf cfg.enable {
    home.packages = [ cfg.package ];

    xdg.configFile."snitchx/snitchx.toml" = lib.mkIf (cfg.settings != { }) {
      source = tomlFormat.generate "snitchx.toml" cfg.settings;
    };
  };
}

