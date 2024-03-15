{
  description = "Wayland-native application runner";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-parts = {
      url = "github:hercules-ci/flake-parts";
      inputs.nixpkgs-lib.follows = "nixpkgs";
    };
  };

  outputs = inputs @ {flake-parts, ...}:
    flake-parts.lib.mkFlake {inherit inputs;} {
      systems = ["x86_64-linux" "aarch64-linux"];

      perSystem = {
        pkgs,
        system,
        ...
      }: let
        inherit (pkgs) callPackage;

        dependencies = with pkgs; [
          glib
          gobject-introspection
          gtk4
          gtk4-layer-shell
          gdk-pixbuf
          graphene
          cairo
          pango
        ];
      in {
        formatter = pkgs.alejandra;

        devShells.default = callPackage ./shell.nix {inherit dependencies;};

        packages = rec {
          default = callPackage ./. {inherit dependencies;};
          walker = default;
        };
      };

      # flake = {};
    };
}
