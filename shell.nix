{
  mkShell,
  walker,
}:
mkShell {
  inputsFrom = [walker];
}
