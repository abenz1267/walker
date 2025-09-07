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
  poppler,
}:
rustPlatform.buildRustPackage rec {
  pname = "walker";
  version = (builtins.fromTOML (builtins.readFile ../Cargo.toml)).package.version;

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

  cargoLock.lockFile = "${src}/Cargo.lock";

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
    poppler
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
