{
  rustPlatform,
  lib,
  pkg-config,
  protobuf,
  glib,
  gobject-introspection,
  gst_all_1,
  gtk4,
  gtk4-layer-shell,
  gdk-pixbuf,
  graphene,
  cairo,
  pango,
  wrapGAppsHook4,
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
    wrapGAppsHook4
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
  ] ++ (with gst_all_1; [
    gstreamer
    gst-plugins-base
    gst-plugins-good
    gst-libav
  ]);

  meta = {
    description = "Wayland-native application runner";
    homepage = "https://github.com/abenz1267/walker";
    license = lib.licenses.mit;
    maintainers = with lib.maintainers; [diniamo NotAShelf];
    platforms = lib.platforms.linux;
    mainProgram = "walker";
  };
}
