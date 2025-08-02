package ai

import (
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/abenz1267/walker/internal/config"
	"github.com/abenz1267/walker/internal/modules/ai/providers"
	"github.com/abenz1267/walker/internal/util"

	"github.com/diamondburned/gotk4/pkg/core/gioutil"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

const (
	aiHistoryFile = "ai_history_0.9.6.gob"
)

type AI struct {
	config          config.AI
	entries         []*util.Entry
	currentPrompt   *config.AIPrompt
	canProcess      bool
	currentMessages []providers.Message
	history         map[string][]providers.Message
	list            *gtk.ListView
	items           *gioutil.ListModel[providers.Message]
	spinner         *gtk.Spinner
	terminal        string
	provider        map[string]providers.Provider
}

func (ai *AI) Cleanup() {
	ai.currentPrompt = nil
	ai.currentMessages = []providers.Message{}

	if ai.list == nil {
		return
	}

	glib.IdleAdd(func() {
		ai.items.Splice(0, int(ai.items.NItems()))
	})
}

func (ai *AI) Entries(term string) []*util.Entry {
	return ai.entries
}

func (ai *AI) General() *config.GeneralModule {
	return &ai.config.GeneralModule
}

func (ai *AI) Refresh() {
}

type providerInitFunction func(config.AI, func(args ...interface{})) providers.Provider

var knownProviders = map[string]providerInitFunction{
	"gemini":    providers.NewGeminiProvider,
	"anthropic": providers.NewAnthropicProvider,
}

func (ai *AI) Setup() bool {
	ai.config = config.Cfg.Builtins.AI
	ai.provider = make(map[string]providers.Provider)
	for name, initFunc := range knownProviders {
		if provider := initFunc(ai.config, ai.SpecialFunc); provider != nil {
			ai.provider[name] = provider
		}
	}

	file := filepath.Join(util.CacheDir(), aiHistoryFile)

	ai.history = make(map[string][]providers.Message)

	util.FromGob(file, &ai.history)

	return len(ai.provider) > 0
}

func (ai *AI) ResumeLastMessages() {
	ai.currentMessages = ai.history[ai.currentPrompt.Prompt]
	ai.items.Splice(0, int(ai.items.NItems()), ai.currentMessages...)

	glib.IdleAdd(func() {
		ai.list.ScrollTo(uint(len(ai.currentMessages)-1), gtk.ListScrollNone, nil)
	})
}

func (ai *AI) ClearCurrent() {
	ai.currentMessages = []providers.Message{}
	ai.items.Splice(0, int(ai.items.NItems()))
}

func (ai *AI) SetupData() {
	for _, v := range ai.provider {
		ai.entries = append(ai.entries, v.SetupData()...)
	}
	ai.config.IsSetup = true
	ai.config.HasInitialSetup = true
}

func (ai *AI) CopyLastResponse() {
	cmd := exec.Command("wl-copy")
	cmd.Stdin = strings.NewReader(ai.currentMessages[len(ai.currentMessages)-1].Content)

	err := cmd.Start()
	if err != nil {
		log.Println(err)
		return
	}

	go func() {
		cmd.Wait()
	}()
}

func (ai *AI) SpecialFunc(args ...interface{}) {
	provider := args[0].(string)
	prompt := args[1].(*config.AIPrompt)
	query := args[2].(string)
	list := args[3].(*gtk.ListView)
	items := args[4].(*gioutil.ListModel[providers.Message])
	spinner := args[5].(*gtk.Spinner)

	if ai.currentPrompt == nil {
		ai.currentPrompt = prompt
		ai.canProcess = true
		ai.list = list
		ai.items = items
		ai.spinner = spinner
		return
	}
	glib.IdleAdd(func() {
		ai.spinner.SetVisible(true)
	})

	queryPos := len(ai.currentMessages)

	ai.provider[provider].Query(query, &ai.currentMessages, ai.currentPrompt, items)

	ai.history[ai.currentPrompt.Prompt] = ai.currentMessages

	util.ToGob(&ai.history, filepath.Join(util.CacheDir(), aiHistoryFile))

	ai.items.Splice(0, int(ai.items.NItems()), ai.currentMessages...)

	glib.IdleAdd(func() {
		ai.list.ScrollTo(uint(queryPos), gtk.ListScrollNone, nil)
		ai.spinner.SetVisible(false)
	})
}

func (ai *AI) RunLastMessageInTerminal() {
	last := ai.currentMessages[len(ai.currentMessages)-1].Content
	shell := os.Getenv("SHELL")

	toRun := fmt.Sprintf("%s %s -e sh -c \"%s; exec %s\"", ai.terminal, config.Cfg.TerminalTitleFlag, last, shell)
	cmd := exec.Command("sh", "-c", util.WrapWithPrefix(ai.General().LaunchPrefix, toRun))

	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid:    true,
		Pgid:       0,
		Foreground: false,
	}

	err := cmd.Start()
	if err != nil {
		slog.Error("Failed to start terminal", "err", err)
		return
	}

	go func() {
		cmd.Wait()
	}()
}
