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
  vendorHash = "sha256-mey6LyBKWhSlrjSztMHOX+g/fqlX3yvVIa6Rfgt6t/k=";

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
