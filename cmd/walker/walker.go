package main

import (
	_ "embed"
	"fmt"
)

//go:embed version.txt
var version string

func main() {
	fmt.Println(version)
}
