{
  description = "Wayland-native application runner";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-parts = {
      url = "github:hercules-ci/flake-parts";
      inputs.nixpkgs-lib.follows = "nixpkgs";
    };
    gomod2nix = {
      url = "github:nix-community/gomod2nix";
      inputs.nixpkgs.follows = "nixpkgs";
    };
  };

  outputs = inputs @ {
    flake-parts,
    gomod2nix,
    ...
  }:
    flake-parts.lib.mkFlake {inherit inputs;} {
      systems = ["x86_64-linux" "aarch64-linux"];

      perSystem = {
        pkgs,
        system,
        ...
      }: let
        inherit (pkgs) callPackage;

        gomod2nixPkgs = gomod2nix.legacyPackages.${system};
      in {
        formatter = pkgs.alejandra;

        devShells.default = callPackage ./shell.nix {inherit (gomod2nixPkgs) mkGoEnv gomod2nix;};

        packages = rec {
          default = callPackage ./. {inherit (gomod2nixPkgs) buildGoApplication;};
          walker = default;
        };
      };

      # flake = {};
    };
}
