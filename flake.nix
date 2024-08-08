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
        walker = pkgs.callPackage ./nix/package.nix {};
      in {
        formatter = pkgs.alejandra;

        devShells.default = pkgs.mkShell { inputsFrom = [walker]; };

        packages.default = walker;
      };

      flake = {
        homeManagerModules.default = import ./nix/hm-module.nix self;

        nixConfig = {
          extra-substituters = ["https://walker-git.cachix.org"];
          extra-trusted-public-keys = ["walker-git.cachix.org-1:vmC0ocfPWh0S/vRAQGtChuiZBTAe4wiKDeyyXM0/7pM="];
        };
      };
    };
}
