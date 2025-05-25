package wlr

import (
	"log"
	"sync"

	"github.com/neurlang/wayland/wl"
	"github.com/neurlang/wayland/wlclient"
)

var (
	registry *wl.Registry
	display  *wl.Display
	seat     []*wl.Seat
)

type windowmap map[wl.ProxyId]*Window

var windows = make(windowmap)

var IsRunning = false

func GetWindows() windowmap {
	return windows
}

func Activate(id wl.ProxyId) {
	err := windows[id].Toplevel.Activate(seat[len(seat)-1])
	if err != nil {
		log.Fatalf("unable to activate toplevel: %v", err)
	}
}

func StartWM() {
	var err error

	display, err = wl.Connect("")
	if err != nil {
		log.Fatalf("unable to connect to wayland server: %v", err)
	}

	display.AddErrorHandler(displayErrorHandler{})

	registry, err = display.GetRegistry()
	if err != nil {
		log.Fatalf("unable to get global registry object: %v", err)
	}

	registry.AddGlobalHandler(registryGlobalHandler{})

	_ = wlclient.DisplayRoundtrip(display)

	IsRunning = true

	for {
		err = display.Context().Run()
		if err != nil {
			log.Fatalf("error when running: %v", err)
		}
	}
}

type displayErrorHandler struct{}

func (displayErrorHandler) HandleDisplayError(e wl.DisplayErrorEvent) {
	log.Fatalf("display error event: %v", e)
}

type registryGlobalHandler struct{}

func (registryGlobalHandler) HandleRegistryGlobal(e wl.RegistryGlobalEvent) {
	switch e.Interface {
	case "zwlr_foreign_toplevel_manager_v1":
		manager := NewZwlrForeignToplevelManagerV1(display.Context())

		err := registry.Bind(e.Name, e.Interface, e.Version, manager)
		if err != nil {
			log.Fatalf("unable to bind wl_compositor interface: %v", err)
		}

		manager.AddToplevelHandler(&Window{})
	case "wl_seat":
		seat = append(seat, wl.NewSeat(display.Context()))

		err := registry.Bind(e.Name, e.Interface, e.Version, seat[len(seat)-1])
		if err != nil {
			log.Fatalf("unable to bind wl_seat interface: %v", err)
		}
	}
}

type Window struct {
	mutex      sync.Mutex
	Toplevel   *ZwlrForeignToplevelHandleV1
	AppId      string
	Title      string
	AddChan    chan string
	DeleteChan chan string
}

func (*Window) HandleZwlrForeignToplevelManagerV1Toplevel(e ZwlrForeignToplevelManagerV1ToplevelEvent) {
	handler := &Window{
		Toplevel:   e.Toplevel,
		AddChan:    addChan,
		DeleteChan: deleteChan,
	}

	e.Toplevel.AddTitleHandler(handler)
	e.Toplevel.AddAppIdHandler(handler)
	e.Toplevel.AddClosedHandler(handler)

	windows[e.Toplevel.Id()] = &Window{Toplevel: e.Toplevel}
}

func (h *Window) HandleZwlrForeignToplevelHandleV1Closed(e ZwlrForeignToplevelHandleV1ClosedEvent) {
	if h.DeleteChan != nil {
		h.DeleteChan <- h.AppId
	}

	h.mutex.Lock()
	defer h.mutex.Unlock()
	delete(windows, h.Toplevel.Id())
}

func (h *Window) HandleZwlrForeignToplevelHandleV1AppId(e ZwlrForeignToplevelHandleV1AppIdEvent) {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	windows[h.Toplevel.Id()].AppId = e.AppId
	h.AppId = e.AppId

	if h.AddChan != nil {
		h.AddChan <- e.AppId
	}
}

func (h *Window) HandleZwlrForeignToplevelHandleV1Title(e ZwlrForeignToplevelHandleV1TitleEvent) {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	windows[h.Toplevel.Id()].Title = e.Title
}
