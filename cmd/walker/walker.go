package main

import (
	_ "embed"
	"fmt"
	"os"
	"slices"

	"github.com/abenz1267/walker/internal/setup"
)

//go:embed version.txt
var version string

func main() {
	if slices.Contains(os.Args, "-v") || slices.Contains(os.Args, "--version") {
		fmt.Println(version)
		return
	}

	setup.GTK()
}
