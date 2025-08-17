{
  description = ''
    Multi-Purpose Launcher with a lot of features. Highly Customizable and fast.
  '';

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    systems.url = "github:nix-systems/default-linux";
    elephant.url = "github:kagurazakei/elephant";
  };

  outputs = {
    self,
    nixpkgs,
    systems,
    elephant,
    ...
  }: let
    inherit (nixpkgs) lib;
    eachSystem = f:
      lib.genAttrs (import systems)
      (system: f nixpkgs.legacyPackages.${system});
  in {
    formatter = eachSystem (pkgs: pkgs.alejandra);

    devShells = eachSystem (pkgs: {
      default = pkgs.mkShell {
        name = "walker";
        inputsFrom = [self.packages.${pkgs.system}.walker];
      };
    });

    packages = eachSystem (pkgs: {
      default = self.packages.${pkgs.system}.walker;
      walker = pkgs.callPackage ./nix/package.nix {};
    });

    homeManagerModules = {
      default = self.homeManagerModules.walker;
      walker = import ./nix/modules/home-manager.nix {inherit self elephant;};
    };

    nixosModules = {
      default = self.nixosModules.walker;
      walker = import ./nix/modules/nixos.nix {inherit self elephant;};
    };

    nixConfig = {
      extra-substituters = ["https://walker-git.cachix.org"];
      extra-trusted-public-keys = ["walker-git.cachix.org-1:vmC0ocfPWh0S/vRAQGtChuiZBTAe4wiKDeyyXM0/7pM="];
    };
  };
}