package main

import (
	_ "embed"
	"fmt"
	"os"

	"github.com/abenz1267/walker/state"
	"github.com/abenz1267/walker/ui"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

//go:embed version.txt
var version string

func main() {
	state := state.Get()

	if state.IsRunning {
		return
	}

	// withArgs := false

	if len(os.Args) > 1 {
		args := os.Args[1:]

		if len(os.Args) > 0 {
			switch args[0] {
			case "--version":
				fmt.Println(version)
				return
			case "--gapplication-service":
				state.IsService = true
			case "--help", "-h", "--help-all":
				// withArgs = true
			default:
				fmt.Printf("Unsupported option '%s'\n", args[0])
				return
			}
		}
	}

	// if !state.IsService && !withArgs {
	// 	tmp := os.TempDir()
	// 	if _, err := os.Stat(filepath.Join(tmp, "walker.lock")); err == nil {
	// 		log.Println("lockfile exists. exiting.")
	// 		return
	// 	}
	//
	// 	err := os.WriteFile(filepath.Join(tmp, "walker.lock"), []byte{}, 0o600)
	// 	if err != nil {
	// 		log.Fatalln(err)
	// 	}
	// 	defer os.Remove(filepath.Join(tmp, "walker.lock"))
	// }

	app := gtk.NewApplication("dev.benz.walker", 0)
	app.Connect("activate", ui.Activate(state))

	app.Flags()

	if state.IsService {
		app.Hold()
	}

	if code := app.Run(os.Args); code > 0 {
		os.Exit(code)
	}
}
