{
  description = "Wayland-native application runner";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-parts = {
      url = "github:hercules-ci/flake-parts";
      inputs.nixpkgs-lib.follows = "nixpkgs";
    };
  };

  outputs = inputs @ {
    flake-parts,
    self,
    ...
  }:
    flake-parts.lib.mkFlake {inherit inputs;} {
      systems = ["x86_64-linux" "aarch64-linux"];

      perSystem = {pkgs, ...}: let
        walker = pkgs.callPackage ./nix/package.nix;
      in {
        formatter = pkgs.alejandra;

        devShells.default = pkgs.mkShell {
          inputsFrom = [walker];
        };

        packages = {
          default = walker;
          inherit walker;
        };
      };

      flake = {
        homeManagerModules = rec {
          walker = import ./nix/hm-module.nix self;
          default = walker;
        };

        nixConfig = {
          extra-substituters = ["https://walker.cachix.org"];
          extra-trusted-public-keys = ["walker.cachix.org-1:fG8q+uAaMqhsMxWjwvk0IMb4mFPFLqHjuvfwQxE4oJM="];
        };
      };
    };
}
