{
  mkShell,
  go,
  pkg-config,
  dependencies,
}:
mkShell {
  packages =
    [
      # Build
      go
      pkg-config
    ]
    ++ dependencies;
}
