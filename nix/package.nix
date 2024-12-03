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

  vendorHash = "sha256-XE8M6g4a2sV0sFeElsY2sShX8ddGCTZcf/ZQaeenRwk=";

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
  ];

  meta = with lib; {
    description = "Wayland-native application runner";
    homepage = "https://github.com/abenz1267/walker";
    license = licenses.mit;
    maintainers = with maintainers; [diniamo];
    platforms = platforms.linux;
    mainProgram = "walker";
  };
}
