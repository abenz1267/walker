{
  mkShell,
  go,
  pkg-config,
  glib,
  gobject-introspection,
  gtk4,
  gtk4-layer-shell,
  gdk-pixbuf,
  graphene,
  cairo,
  pango,
}:
mkShell {
  packages = [
    # Build
    go
    pkg-config

    # Dependencies
    glib
    gobject-introspection
    gtk4
    gtk4-layer-shell
    gdk-pixbuf
    graphene
    cairo
    pango
  ];
}
