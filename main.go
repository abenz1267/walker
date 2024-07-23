package main

import (
	"bufio"
	_ "embed"
	"fmt"
	"log"
	"os"
	"os/signal"
	"slices"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/abenz1267/walker/config"
	"github.com/abenz1267/walker/modules"
	"github.com/abenz1267/walker/state"
	"github.com/abenz1267/walker/ui"
	"github.com/diamondburned/gotk4/pkg/gio/v2"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

//go:embed version.txt
var version string

func main() {
	state := state.Get()

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

			if slices.Contains(args, "-y") || slices.Contains(args, "--password") {
				forceNew = true
				state.Password = true
			}

			if slices.Contains(args, "--gapplication-service") {
				state.IsService = true
			}

			if slices.Contains(args, "--forceprint") || slices.Contains(args, "-f") {
				state.ForcePrint = true
			}

			if slices.Contains(args, "--bench") || slices.Contains(args, "-b") {
				fmt.Println(time.Now().UnixNano())
				state.Benchmark = true
			}

			if slices.Contains(args, "--version") || slices.Contains(args, "-v") || slices.Contains(args, "--help-all") {
				fmt.Println(version)
				return
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

	app := gtk.NewApplication(appName, gio.ApplicationHandlesCommandLine)

	app.AddMainOption("modules", 'm', glib.OptionFlagNone, glib.OptionArgString, "modules to be loaded", "the modules")
	app.AddMainOption("new", 'n', glib.OptionFlagNone, glib.OptionArgNone, "start new instance ignoring service", "")
	app.AddMainOption("keepsort", 'k', glib.OptionFlagNone, glib.OptionArgNone, "don't sort alphabetically", "")
	app.AddMainOption("password", 'y', glib.OptionFlagNone, glib.OptionArgNone, "launch in password mode", "")
	app.AddMainOption("dmenu", 'd', glib.OptionFlagNone, glib.OptionArgNone, "run in dmenu mode", "")
	app.AddMainOption("config", 'c', glib.OptionFlagNone, glib.OptionArgString, "config file to use", "")
	app.AddMainOption("style", 's', glib.OptionFlagNone, glib.OptionArgString, "style file to use", "")
	app.AddMainOption("placeholder", 'p', glib.OptionFlagNone, glib.OptionArgString, "placeholder text", "")
	app.AddMainOption("labelcolumn", 'l', glib.OptionFlagNone, glib.OptionArgString, "column to use for the label", "")
	app.AddMainOption("separator", 't', glib.OptionFlagNone, glib.OptionArgString, "column separator", "")
	app.AddMainOption("version", 'v', glib.OptionFlagNone, glib.OptionArgNone, "print version", "")
	app.AddMainOption("forceprint", 'f', glib.OptionFlagNone, glib.OptionArgNone, "forces printing input if no item is selected", "")
	app.AddMainOption("bench", 'b', glib.OptionFlagNone, glib.OptionArgNone, "prints nanoseconds for start and displaying in both service and client", "")

	app.Connect("activate", ui.Activate(state))

	app.ConnectCommandLine(func(cmd *gio.ApplicationCommandLine) int {
		options := cmd.OptionsDict()

		modulesString := options.LookupValue("modules", glib.NewVariantString("").Type())
		configString := options.LookupValue("config", glib.NewVariantString("").Type())
		styleString := options.LookupValue("style", glib.NewVariantString("").Type())
		placeholderString := options.LookupValue("placeholder", glib.NewVariantString("").Type())
		labelColumnString := options.LookupValue("labelcolumn", glib.NewVariantString("").Type())
		separatorString := options.LookupValue("separator", glib.NewVariantString("").Type())

		if separatorString != nil && separatorString.String() != "" {
			dmenu.Separator = separatorString.String()
		}

		if labelColumnString != nil && labelColumnString.String() != "" {
			col, err := strconv.Atoi(labelColumnString.String())
			if err != nil {
				log.Panicln(err)
			}

			dmenu.LabelColumn = col
		}

		if placeholderString != nil && placeholderString.String() != "" {
			state.ExplicitPlaceholder = placeholderString.String()
		}

		if modulesString != nil && modulesString.String() != "" {
			m := strings.Split(modulesString.String(), ",")
			state.ExplicitModules = m
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

			os.Exit(0)
		}
	}()

	if code := app.Run(os.Args); code > 0 {
		os.Exit(code)
	}
}
