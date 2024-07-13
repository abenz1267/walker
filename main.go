package main

import (
	_ "embed"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/abenz1267/walker/config"
	"github.com/abenz1267/walker/state"
	"github.com/abenz1267/walker/ui"
	"github.com/abenz1267/walker/util"
	"github.com/diamondburned/gotk4/pkg/gio/v2"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"github.com/junegunn/fzf/src/algo"
)

//go:embed version.txt
var version string

func main() {
	state := state.Get()
	algo.Init("default")

	if state.IsRunning {
		return
	}

	withArgs := false
	forceNew := false
	appName := "dev.benz.walker"

	if len(os.Args) > 1 {
		args := os.Args[1:]

		if len(os.Args) > 0 {
			switch args[0] {
			case "-n", "--new":
				forceNew = true
				appName = fmt.Sprintf("%s-%d", appName, time.Now().Unix())
			case "-c", "--config":
			case "-s", "--style":
			case "-m", "--modules":
			case "--version":
				fmt.Println(version)
				return
			case "--gapplication-service":
				state.IsService = true
			case "--help", "-h", "--help-all":
				withArgs = true
			default:
				fmt.Printf("Unsupported option '%s'\n", args[0])
				return
			}
		}
	}

	if forceNew && state.IsService {
		log.Println("new instance is not supported with service mode")
		return
	}

	tmp := util.TmpDir()

	if !state.IsService && !withArgs && !forceNew {
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

	if state.IsService && !forceNew {
		if _, err := os.Stat(filepath.Join(tmp, "walker-service.lock")); err == nil {
			log.Println("lockfile exists. exiting. Remove '/tmp/walker-service.lock' and try again.")
			return
		}

		err := os.WriteFile(filepath.Join(tmp, "walker-service.lock"), []byte{}, 0o600)
		if err != nil {
			log.Panicln(err)
		}
		defer os.Remove(filepath.Join(tmp, "walker-service.lock"))
	}

	app := gtk.NewApplication(appName, gio.ApplicationHandlesCommandLine)

	app.AddMainOption("modules", 'm', glib.OptionFlagNone, glib.OptionArgString, "modules to be loaded", "the modules")
	app.AddMainOption("new", 'n', glib.OptionFlagNone, glib.OptionArgNone, "start new instance ignoring service", "")
	app.AddMainOption("config", 'c', glib.OptionFlagNone, glib.OptionArgString, "config file to use", "")
	app.AddMainOption("style", 's', glib.OptionFlagNone, glib.OptionArgString, "style file to use", "")

	app.Connect("activate", ui.Activate(state))

	app.ConnectCommandLine(func(cmd *gio.ApplicationCommandLine) int {
		options := cmd.OptionsDict()

		modulesString := options.LookupValue("modules", glib.NewVariantString("s").Type())
		configString := options.LookupValue("config", glib.NewVariantString("s").Type())
		styleString := options.LookupValue("style", glib.NewVariantString("s").Type())

		if modulesString != nil && modulesString.String() != "" {
			modules := strings.Split(modulesString.String(), ",")
			state.ExplicitModules = modules
		}

		if configString != nil && configString.String() != "" {
			state.ExplicitConfig = configString.String()
		}

		if styleString != nil && styleString.String() != "" {
			state.ExplicitStyle = styleString.String()
		}

		if state != nil && state.IsService {
			state.StartServiceableModules(config.Get(state.ExplicitConfig))
		}

		app.Activate()
		cmd.Done()

		return 0
	})

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

			if state.IsService {
				os.Remove(filepath.Join(util.TmpDir(), "walker-service.lock"))
			}

			os.Exit(0)
		}
	}()

	if code := app.Run(os.Args); code > 0 {
		os.Exit(code)
	}
}
