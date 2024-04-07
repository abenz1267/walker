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
        callPackage = pkgs.lib.callPackageWith (pkgs // packages);

        packages = {
          walker = callPackage ./. {};
        };
      in {
        formatter = pkgs.alejandra;

        devShells.default = callPackage ./shell.nix {};

        packages = packages // {default = packages.walker;};
      };

      flake = {
        homeManagerModules = rec {
          walker = import ./nix/hm-module.nix inputs.self;
          default = walker;
        };
      };
    };
}
