package main

import (
	_ "embed"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/abenz1267/walker/config"
	"github.com/abenz1267/walker/state"
	"github.com/abenz1267/walker/ui"
	"github.com/abenz1267/walker/util"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

//go:embed version.txt
var version string

func main() {
	state := state.Get()

	if state.IsRunning {
		return
	}

	withArgs := false

	if len(os.Args) > 1 {
		args := os.Args[1:]

		if len(os.Args) > 0 {
			switch args[0] {
			case "--version":
				fmt.Println(version)
				return
			case "--gapplication-service":
				state.IsService = true
				state.StartServiceableModules(config.Get())
			case "--help", "-h", "--help-all":
				withArgs = true
			default:
				fmt.Printf("Unsupported option '%s'\n", args[0])
				return
			}
		}
	}

	if !state.IsService && !withArgs {
		tmp := util.TmpDir()

		if _, err := os.Stat(filepath.Join(tmp, "walker.lock")); err == nil {
			log.Println("lockfile exists. exiting. Remove '/tmp/walker.lock' and try again.")
			return
		}

		err := os.WriteFile(filepath.Join(tmp, "walker.lock"), []byte{}, 0o600)
		if err != nil {
			log.Panicln(err)
		}
		defer os.Remove(filepath.Join(tmp, "walker.lock"))
	}

	app := gtk.NewApplication("dev.benz.walker", 0)
	app.Connect("activate", ui.Activate(state))

	app.Flags()

	if state.IsService {
		app.Hold()
	}

	signal_chan := make(chan os.Signal, 1)
	signal.Notify(signal_chan,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)

	go func() {
		for {
			<-signal_chan

			os.Remove(filepath.Join(util.TmpDir(), "walker.lock"))
			os.Exit(0)
		}
	}()

	if code := app.Run(os.Args); code > 0 {
		os.Exit(code)
	}
}
