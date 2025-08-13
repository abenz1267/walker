package main

import (
	_ "embed"
	"fmt"

	"github.com/abenz1267/walker/internal/setup"
)

//go:embed version.txt
var version string

func main() {
	setup.GTK()
	fmt.Println(version)
}
