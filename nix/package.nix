{
  buildGoModule,
  lib,
  pkg-config,
  glib,
  gobject-introspection,
  gtk4,
  gtk4-layer-shell,
  gdk-pixbuf,
  graphene,
  cairo,
  pango,
  wrapGAppsHook,
  vips,
  libqalculate,
}:
buildGoModule {
  pname = "walker";
  version = lib.fileContents ../cmd/version.txt;

  src = lib.fileset.toSource {
    root = ../.;
    fileset = lib.fileset.unions [
      ../go.mod
      ../go.sum

      ../cmd
      ../internal
    ];
  };
  subPackages = ["cmd/walker.go"];

  vendorHash = "sha256-SG1JTl/Al9bRyDkzN7xliuZIAMifQJZdIeC5fr0WpWw=";

  nativeBuildInputs = [
    gobject-introspection
    pkg-config
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
    vips
    libqalculate
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
