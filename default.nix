{
  buildGoModule,
  lib,
  pkg-config,
  dependencies,
}:
buildGoModule {
  pname = "walker";
  version = lib.fileContents ./version.txt;

  src = ./.;
  vendorHash = "sha256-rzhcKIphKdb5woKEtNb3V6iFyc2R6QS8WK7942z7F2o=";

  nativeBuildInputs = [pkg-config];
  buildInputs = dependencies;

  meta = with lib; {
    description = "Wayland-native application runner";
    homepage = "https://github.com/abenz1267/walker";
    license = licenses.mit;
    maintainers = with maintainers; [diniamo];
    platforms = platforms.linux;
    mainProgram = "walker";
  };
}
