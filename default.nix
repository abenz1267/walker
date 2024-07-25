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
}:
buildGoModule {
  pname = "walker";
  version = lib.fileContents ./version.txt;

  src = builtins.path {
    name = "walker-source";
    path = ./.;
  };
  vendorHash = "sha256-NJLLPY5U6h/pVp3HOlRywVdizuMmDKzJI1J9czNl+dI=";

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
