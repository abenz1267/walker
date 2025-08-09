package main

import (
	"context"
	_ "embed"
	"fmt"
	"log"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/abenz1267/walker/internal/config"
	"github.com/abenz1267/walker/internal/modules"
	"github.com/abenz1267/walker/internal/modules/clipboard"
	"github.com/abenz1267/walker/internal/state"
	"github.com/abenz1267/walker/internal/ui"
	"github.com/abenz1267/walker/internal/util"
	"github.com/adrg/xdg"
	"github.com/diamondburned/gotk4/pkg/gio/v2"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

//go:embed version.txt
var version string

var (
	now          = time.Now().UnixMilli()
	SocketReopen = filepath.Join(util.TmpDir(), "walker-reopen.sock")
	app          *gtk.Application
)

func main() {
	state := state.Get()

	appName := "dev.benz.walker"

	if len(os.Args) > 1 {
		args := os.Args[1:]

		if len(os.Args) > 0 {
			if slices.Contains(args, "-n") ||
				slices.Contains(args, "--new") ||
				slices.Contains(args, "-y") ||
				slices.Contains(args, "--password") {
				appName = fmt.Sprintf("%s-%d", appName, time.Now().Unix())
			}

			if slices.Contains(args, "--gapplication-service") {
				state.IsService = true
			}
		}
	}

	app = gtk.NewApplication(appName, gio.ApplicationHandlesCommandLine)
	app.Connect("activate", ui.Activate(state))
	app.ConnectCommandLine(handleCmd(state))
	app.ConnectHandleLocalOptions(func(options *glib.VariantDict) int {
		if options.Contains("config") {
			state.ExplicitConfig = options.LookupValue("config", glib.NewVariantType("s")).String()
		}

		if config.Cfg == nil {
			state.ConfigError = config.Init(state.ExplicitConfig)
		}

		if state.IsService {
			if !state.ModulesStarted {
				state.StartServiceableModules()
			}

			go listenActivationSocket()
		}

		return -1
	})

	addFlags(app)
	app.Flags()

	if state.IsService {
		app.Hold()

		signal_chan := make(chan os.Signal, 1)
		signal.Notify(signal_chan,
			syscall.SIGHUP,
			syscall.SIGINT,
			syscall.SIGTERM,
			syscall.SIGKILL,
			syscall.SIGQUIT, syscall.SIGUSR1)

		go func() {
			for {
				signal := <-signal_chan

				switch signal {
				case syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL, syscall.SIGQUIT:

					os.Exit(0)
				case syscall.SIGUSR1:
					if state.Clipboard != nil {
						state.Clipboard.(*clipboard.Clipboard).Update()
					}
				default:
					slog.Error("signal", "error", "unknown signal", signal)
				}
			}
		}()
	}

	code := app.Run(os.Args)

	os.Exit(code)
}

func addFlags(app *gtk.Application) {
	app.AddMainOption("autoselect", 'x', glib.OptionFlagNone, glib.OptionArgNone, "auto select only item in list", "")
	app.AddMainOption("hidebar", 'H', glib.OptionFlagNone, glib.OptionArgNone, "hide the search bar", "")
	app.AddMainOption("modules", 'm', glib.OptionFlagNone, glib.OptionArgString, "modules to be loaded", "the modules")
	app.AddMainOption("new", 'n', glib.OptionFlagNone, glib.OptionArgNone, "start new instance ignoring service", "")
	app.AddMainOption("keepsort", 'k', glib.OptionFlagNone, glib.OptionArgNone, "don't sort alphabetically", "")
	app.AddMainOption("password", 'y', glib.OptionFlagNone, glib.OptionArgNone, "launch in password mode", "")
	app.AddMainOption("config", 'c', glib.OptionFlagNone, glib.OptionArgString, "config file to use", "")
	app.AddMainOption("theme", 's', glib.OptionFlagNone, glib.OptionArgString, "theme to use", "")
	app.AddMainOption("clear-clipboard", 'u', glib.OptionFlagNone, glib.OptionArgNone, "clears the clipboard content", "")
	app.AddMainOption("width", 'w', glib.OptionFlagNone, glib.OptionArgString, "overwrite width", "")
	app.AddMainOption("height", 'h', glib.OptionFlagNone, glib.OptionArgString, "overwrite height", "")
	app.AddMainOption("placeholder", 'p', glib.OptionFlagNone, glib.OptionArgString, "placeholder text", "")
	app.AddMainOption("query", 'q', glib.OptionFlagNone, glib.OptionArgString, "initial query", "")
	app.AddMainOption("version", 'v', glib.OptionFlagNone, glib.OptionArgNone, "print version", "")
	app.AddMainOption("forceprint", 'f', glib.OptionFlagNone, glib.OptionArgNone, "forces printing input if no item is selected", "")
	app.AddMainOption("enableautostart", 'A', glib.OptionFlagNone, glib.OptionArgNone, "creates a desktop file for autostarting on login", "")
	app.AddMainOption("disableautostart", 'D', glib.OptionFlagNone, glib.OptionArgNone, "removes the autostart desktop file", "")
	app.AddMainOption("createuserconfig", 'C', glib.OptionFlagNone, glib.OptionArgNone, "writes the default config to xdg_user_config", "")

	// dmenu flags
	app.AddMainOption("active", 'a', glib.OptionFlagNone, glib.OptionArgString, "active item (visually) (dmenu)", "")
	app.AddMainOption("preselect", 'P', glib.OptionFlagNone, glib.OptionArgString, "preselected item (dmenu)", "")
	app.AddMainOption("dmenu", 'd', glib.OptionFlagNone, glib.OptionArgNone, "run in dmenu mode", "")
	app.AddMainOption("label", 'l', glib.OptionFlagNone, glib.OptionArgString, "column to use for the label (dmenu)", "")
	app.AddMainOption("icon", 'i', glib.OptionFlagNone, glib.OptionArgString, "column to use for the icon (dmenu)", "")
	app.AddMainOption("value", 'V', glib.OptionFlagNone, glib.OptionArgString, "column to use for the value (dmenu)", "")
	app.AddMainOption("separator", 't', glib.OptionFlagNone, glib.OptionArgString, "column separator (dmenu)", "")
	app.AddMainOption("stream", 's', glib.OptionFlagNone, glib.OptionArgNone, "stream data (dmenu)", "")
}

func handleCmd(state *state.AppState) func(cmd *gio.ApplicationCommandLine) int {
	return func(cmd *gio.ApplicationCommandLine) int {
		options := cmd.OptionsDict()

		if options.Contains("version") {
			cmd.PrintLiteral(fmt.Sprintf("Running Service: %t\n", state.IsService))
			cmd.PrintLiteral(fmt.Sprintf("%s", version))
			cmd.Done()
			return 0
		}

		if options.Contains("clear-clipboard") {
			state.Clipboard.(*clipboard.Clipboard).Clear()
			cmd.Done()
			return 0
		}

		if options.Contains("createuserconfig") {
			config.WriteUserConfig()
			cmd.Done()
			return 0
		}

		if options.Contains("enableautostart") {
			enableAutostart()
			cmd.Done()
			return 0
		}

		if options.Contains("disableautostart") {
			disableAutostart()
			cmd.Done()
			return 0
		}

		state.WidthOverwrite = gtkStringToInt(options.LookupValue("width", glib.NewVariantType("s")))
		state.HeightOverwrite = gtkStringToInt(options.LookupValue("height", glib.NewVariantType("s")))
		state.AutoSelect = options.Contains("autoselect")
		state.Hidebar = options.Contains("hidebar")

		if state.IsDmenu {
			state.DmenuResultChan <- "ABRT"
		}

		state.IsDmenu = options.Contains("dmenu")

		if state.IsDmenu {
			if !app.Flags().Has(gio.ApplicationIsService) || state.Dmenu == nil {
				state.Dmenu = &modules.Dmenu{
					DmenuShowChan: state.DmenuShowChan,
				}

				state.Dmenu.Setup()
			}

			state.ExplicitModules = []string{"dmenu"}

			stream := options.Contains("stream")
			state.Dmenu.General().Stream = stream

			if options.Contains("separator") {
				state.Dmenu.Separator = options.LookupValue("separator", glib.NewVariantType("s")).String()
			}

			if options.Contains("label") {
				col := gtkStringToInt(options.LookupValue("label", glib.NewVariantType("s")))
				state.Dmenu.LabelColumn = col - 1
			}

			if options.Contains("icon") {
				col := gtkStringToInt(options.LookupValue("icon", glib.NewVariantType("s")))
				state.Dmenu.IconColum = col - 1
			}

			if options.Contains("value") {
				col := gtkStringToInt(options.LookupValue("value", glib.NewVariantType("s")))

				state.Dmenu.ValueColumn = col - 1
			}

			if options.Contains("active") {
				val := gtkStringToInt(options.LookupValue("active", glib.NewVariantType("s"))) - 1
				state.ActiveItem = &val
			} else {
				state.ActiveItem = nil
			}

			if options.Contains("preselect") {
				val := gtkStringToInt(options.LookupValue("preselect", glib.NewVariantType("s"))) - 1
				state.Preselected = &val
			} else {
				state.Preselected = nil
			}

			if !slices.Contains(config.Cfg.Disabled, state.Dmenu.General().Name) {
				stdin := cmd.Stdin()
				if stdin == nil {
					fmt.Println("No stdin available")
					cmd.Done()
					return 1
				}

				reader := gio.NewDataInputStream(stdin)
				ctx := context.Background()

				if stream {
					go func() {
						throttler := ui.NewLatestOnlyThrottler(100 * time.Millisecond)
						defer throttler.Stop()

						id := int(time.Now().UnixMilli())
						state.DmenuStreamId = id
						state.Dmenu.ClearEntries()

						for {
							if id != state.DmenuStreamId {
								return
							}

							size, res, err := reader.ReadLine(ctx)
							if err == nil && size != 0 {
								state.Dmenu.Append(string(res))
								throttler.Execute(id)
							} else {
								break
							}
						}

						if id == state.DmenuStreamId {
							state.DmenuStreamDone <- struct{}{}
						}
					}()
				} else {
					for {
						size, res, err := reader.ReadLine(ctx)
						if err == nil && size != 0 {
							state.Dmenu.Append(string(res))
						} else {
							break
						}
					}
				}
			}
		} else {
			if options.Contains("modules") {
				m := strings.Split(options.LookupValue("modules", glib.NewVariantType("s")).String(), ",")
				state.ExplicitModules = m
			}
		}

		state.ForcePrint = options.Contains("forceprint")
		state.Password = options.Contains("password")
		state.KeepSort = options.Contains("keepsort")

		if options.Contains("placeholder") {
			state.ExplicitPlaceholder = options.LookupValue("placeholder", glib.NewVariantType("s")).String()
		}

		if options.Contains("query") {
			state.InitialQuery = options.LookupValue("query", glib.NewVariantType("s")).String()
		}

		if options.Contains("theme") {
			state.ExplicitTheme = options.LookupValue("theme", glib.NewVariantType("s")).String()
		}

		app.Activate()

		if state.IsDmenu {
			go func() {
				result := <-state.DmenuResultChan

				if result == "CNCLD" || result == "ABRT" {
					cmd.SetExitStatus(130)
				} else {
					cmd.PrintLiteral(fmt.Sprintf("%s\n", result))
				}

				if result != "ABRT" {
					for state.IsRunning {
						time.Sleep(1 * time.Millisecond)
					}
				}

				cmd.Done()
			}()
		} else {
			cmd.Done()
		}

		return 0
	}
}

func listenActivationSocket() {
	os.Remove(SocketReopen)

	l, _ := net.ListenUnix("unix", &net.UnixAddr{
		Name: SocketReopen,
	})
	defer l.Close()

	for {
		conn, err := l.AcceptUnix()
		if err != nil {
			log.Panic(err)
		}
		conn.Close()

		ui.Show(app)
	}
}

func enableAutostart() {
	content := `
[Desktop Entry]
Name=Walker
Comment=Walker Service
Exec=walker --gapplication-service
StartupNotify=false
Terminal=false
Type=Application
				`

	dir := filepath.Join(xdg.ConfigHome, "autostart")
	os.MkdirAll(dir, 0755)

	err := os.WriteFile(filepath.Join(dir, "walker-service.desktop"), []byte(strings.TrimSpace(content)), 0644)
	if err != nil {
		log.Panicln(err)
	}
}

func disableAutostart() {
	err := os.Remove(filepath.Join(xdg.ConfigHome, "autostart", "walker-service.desktop"))
	if err != nil {
		log.Panicln(err)
	}
}

func gtkStringToInt(in *glib.Variant) int {
	if in != nil && in.String() != "" {
		out, err := strconv.Atoi(in.String())
		if err != nil {
			slog.Error("argument", "width", err)
			return 0
		}

		return out
	}

	return 0
}
