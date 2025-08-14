package setup

import (
	"runtime"
	"runtime/debug"

	"github.com/abenz1267/walker/internal/data"
	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

func setupKeyEvents(app *gtk.Application, window *gtk.Window) {
	controller := gtk.NewEventControllerKey()
	controller.SetPropagationPhase(gtk.PropagationPhase(1))

	controller.ConnectKeyPressed(func(val, code uint, state gdk.ModifierType) (ok bool) {
		action, ok := Binds[int(val)][state]

		if ok {
			switch action.action {
			case ActionClose:
				quit()
			case ActionSelectNext:
				selection.SetSelected(selection.Selected() + 1)
			case ActionSelectPrevious:
				selection.SetSelected(selection.Selected() - 1)
			}

			return true
		}

		if data.Items.Len() > 0 {
			item := data.Items.At(int(selection.Selected()))

			action, ok = ProviderBinds[item.Item.Provider][int(val)][state]

			if ok {
				data.Activate(selection.Selected(), currentBuilder.input.Text(), action.action)

				switch action.after {
				case AfterClose:
					quit()
				case AfterReload:
					currentBuilder.input.Emit("changed")
				case AfterNothing:
				}

				return true
			}
		}

		return false
	})

	window.AddController(controller)
}

func quit() {
	if isService {
		currentBuilder.input.SetText("")
		currentBuilder.input.Emit("changed")
		currentBuilder.window.SetVisible(false)

		app.Hold()
		isRunning = false

		runtime.GC()
		debug.FreeOSMemory()
	} else {
		app.Quit()
	}
}
