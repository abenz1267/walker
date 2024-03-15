{buildGoApplication, lib}:

buildGoApplication {
  pname = "walker";
  version = lib.fileContents ./version.txt;
  src = ./.;
  modules = ./gomod2nix.toml;
}
