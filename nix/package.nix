{
  rustPlatform,
  lib,
  pkg-config,
  protobuf,
  glib,
  gobject-introspection,
  gtk4,
  gtk4-layer-shell,
  gdk-pixbuf,
  graphene,
  cairo,
  pango,
  wrapGAppsHook,
  poppler-glib
}:
rustPlatform.buildRustPackage {
  pname = "walker";
  version = "1.0.0-beta11";

  src = lib.fileset.toSource {
    root = ../.;
    fileset = lib.fileset.unions [
      ../Cargo.toml
      ../Cargo.lock
      ../src
      ../build.rs
      ../resources
    ];
  };

  cargoHash = "sha256-WqvT+4Yf16cqd0A7y6lW7oW2Mx2GoGRWoejrWXXwrVc=";

  nativeBuildInputs = [
    gobject-introspection
    pkg-config
    protobuf
    wrapGAppsHook
  ];

  buildInputs = [
    glib
    gtk4
    gtk4-layer-shell
    gdk-pixbuf
    graphene
    cairo
    pango
    poppler-glib
  ];

  meta = {
    description = "Wayland-native application runner";
    homepage = "https://github.com/abenz1267/walker";
    license = lib.licenses.mit;
    maintainers = with lib.maintainers; [diniamo NotAShelf];
    platforms = lib.platforms.linux;
    mainProgram = "walker";
  };
}
