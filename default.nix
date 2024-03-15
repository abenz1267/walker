{
  buildGoApplication,
  lib,
  go,
  pkg-config,
  glibc,
}:
buildGoApplication {
  pname = "walker";
  version = lib.fileContents ./version.txt;

  pwd = ./.;
  src = ./.;
  modules = ./gomod2nix.toml;
  inherit go;

  nativeBuildInputs = [pkg-config glibc];

  meta = with lib; {
    description = "Wayland-native application runner";
    homepage = "https://github.com/abenz1267/walker";
    license = licenses.mit;
    maintainers = with maintainers; [diniamo];
    platforms = platforms.linux;
    mainProgram = "walker";
  };
}
