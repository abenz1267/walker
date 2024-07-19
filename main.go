package main

import (
	"bufio"
	_ "embed"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/abenz1267/walker/config"
	"github.com/abenz1267/walker/modules"
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

	var dmenu *modules.Dmenu

	if len(os.Args) > 1 {
		args := os.Args[1:]

		if len(os.Args) > 0 {
			if slices.Contains(args, "-d") || slices.Contains(args, "--dmenu") {
				forceNew = true

				dmenu = &modules.Dmenu{
					Content:     []string{},
					LabelColumn: 0,
				}

				scanner := bufio.NewScanner(os.Stdin)

				for scanner.Scan() {
					dmenu.Content = append(dmenu.Content, scanner.Text())
				}

				state.Dmenu = dmenu
			}

			if slices.Contains(args, "-n") || slices.Contains(args, "--new") {
				forceNew = true
			}

			if slices.Contains(args, "-k") || slices.Contains(args, "--keepsort") {
				state.KeepSort = true
			}

			if slices.Contains(args, "--gapplication-service") {
				state.IsService = true
			}

			if slices.Contains(args, "--help") || slices.Contains(args, "-h") || slices.Contains(args, "--help-all") {
				withArgs = true
			}
		}
	}

	if forceNew {
		appName = fmt.Sprintf("%s-%d", appName, time.Now().Unix())
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
	app.AddMainOption("keepsort", 'k', glib.OptionFlagNone, glib.OptionArgNone, "don't sort alphabetically", "")
	app.AddMainOption("dmenu", 'd', glib.OptionFlagNone, glib.OptionArgNone, "run in dmenu mode", "")
	app.AddMainOption("config", 'c', glib.OptionFlagNone, glib.OptionArgString, "config file to use", "")
	app.AddMainOption("style", 's', glib.OptionFlagNone, glib.OptionArgString, "style file to use", "")
	app.AddMainOption("placeholder", 'p', glib.OptionFlagNone, glib.OptionArgString, "placeholder text", "")
	app.AddMainOption("labelcolumn", 'l', glib.OptionFlagNone, glib.OptionArgString, "column to use for the label", "")

	app.Connect("activate", ui.Activate(state))

	app.ConnectCommandLine(func(cmd *gio.ApplicationCommandLine) int {
		options := cmd.OptionsDict()

		modulesString := options.LookupValue("modules", glib.NewVariantString("").Type())
		configString := options.LookupValue("config", glib.NewVariantString("").Type())
		styleString := options.LookupValue("style", glib.NewVariantString("").Type())
		placeholderString := options.LookupValue("placeholder", glib.NewVariantString("").Type())
		labelColumnString := options.LookupValue("labelcolumn", glib.NewVariantString("").Type())

		if labelColumnString != nil && labelColumnString.String() != "" {
			col, err := strconv.Atoi(labelColumnString.String())
			if err != nil {
				log.Panicln(err)
			}

			if col < 1 {
				col = 1
			}

			dmenu.LabelColumn = col
		}

		if placeholderString != nil && placeholderString.String() != "" {
			state.ExplicitPlaceholder = placeholderString.String()
		}

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
