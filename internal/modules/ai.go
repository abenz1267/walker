package modules

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/abenz1267/walker/internal/config"
	"github.com/abenz1267/walker/internal/util"
	"github.com/diamondburned/gotk4/pkg/core/gioutil"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

const (
	ANTHROPIC_VERSION_HEADER = "anthropic-version"
	ANTHROPIC_VERSION        = "2023-06-01"
	ANTHROPIC_API_URL        = "https://api.anthropic.com/v1/messages"
	ANTHROPIC_AUTH_HEADER    = "x-api-key"
	ANTHROPIC_API_KEY        = "ANTHROPIC_API_KEY"
	aiHistoryFile            = "ai_history_0.9.6.gob"
)

type AI struct {
	config          config.AI
	entries         []util.Entry
	anthropicKey    string
	currentPrompt   *config.AnthropicPrompt
	canProcess      bool
	currentMessages []AnthropicMessage
	history         map[string][]AnthropicMessage
	list            *gtk.ListView
	items           *gioutil.ListModel[AnthropicMessage]
	spinner         *gtk.Spinner
	terminal        string
}

func (ai *AI) Cleanup() {
	ai.currentPrompt = nil
	ai.currentMessages = []AnthropicMessage{}

	if ai.list == nil {
		return
	}

	glib.IdleAdd(func() {
		ai.items.Splice(0, int(ai.items.NItems()))
	})
}

func (ai *AI) Entries(ctx context.Context, term string) []util.Entry {
	return ai.entries
}

func (ai *AI) General() *config.GeneralModule {
	return &ai.config.GeneralModule
}

func (ai *AI) Refresh() {
}

func (ai *AI) Setup(cfg *config.Config) bool {
	ai.config = cfg.Builtins.AI
	ai.terminal = cfg.Terminal

	file := filepath.Join(util.CacheDir(), aiHistoryFile)

	ai.history = make(map[string][]AnthropicMessage)

	util.FromGob(file, &ai.history)

	return true
}

func (ai *AI) ResumeLastMessages() {
	ai.currentMessages = ai.history[ai.currentPrompt.Prompt]
	ai.items.Splice(0, int(ai.items.NItems()), ai.currentMessages...)

	glib.IdleAdd(func() {
		ai.list.ScrollTo(uint(len(ai.currentMessages)-1), gtk.ListScrollNone, nil)
	})
}

func (ai *AI) ClearCurrent() {
	ai.currentMessages = []AnthropicMessage{}
	ai.items.Splice(0, int(ai.items.NItems()))
}

func (ai *AI) SetupData(cfg *config.Config, ctx context.Context) {
	ai.entries = []util.Entry{}

	ai.anthropicKey = os.Getenv(ANTHROPIC_API_KEY)

	if ai.anthropicKey != "" {
		for _, v := range ai.config.Anthropic.Prompts {
			ai.entries = append(ai.entries, util.Entry{
				Label:            v.Label,
				Sub:              "Claude 3.5",
				Exec:             "",
				RecalculateScore: true,
				Matching:         util.Fuzzy,
				SpecialFunc:      ai.SpecialFunc,
				SpecialFuncArgs:  []interface{}{"anthropic", &v},
				SingleModuleOnly: v.SingleModuleOnly,
			})
		}

		if len(ai.config.Anthropic.Prompts) == 0 {
			log.Println("ai: no prompts set.")
		}
	} else {
		log.Println("ai: no anthropic api key set.")
	}

	ai.config.IsSetup = true
	ai.config.HasInitialSetup = true
}

type AnthropicRequest struct {
	Model       string             `json:"model"`
	MaxTokens   int                `json:"max_tokens"`
	Temperature float64            `json:"temperature"`
	System      string             `json:"system"`
	Messages    []AnthropicMessage `json:"messages"`
}

type AnthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type AnthropicResponse struct {
	Id      string `json:"id,omitempty"`
	Type    string `json:"type,omitempty"`
	Role    string `json:"role,omitempty"`
	Model   string `json:"model,omitempty"`
	Content []struct {
		Type         string `json:"type,omitempty"`
		Text         string `json:"text,omitempty"`
		StopReason   string `json:"stop_reason,omitempty"`
		StopSequence string `json:"stop_sequence,omitempty"`
	} `json:"content,omitempty"`
}

func (ai *AI) anthropic(query string) {
	glib.IdleAdd(func() {
		ai.spinner.SetVisible(true)
	})

	queryMsg := AnthropicMessage{
		Role:    "user",
		Content: query,
	}

	messages := []AnthropicMessage{}

	if len(ai.currentMessages) > 0 {
		messages = ai.currentMessages
	}

	messages = append(messages, queryMsg)

	queryPos := len(messages) - 1

	ai.items.Splice(0, int(ai.items.NItems()), messages...)

	req := AnthropicRequest{
		Model:       ai.currentPrompt.Model,
		MaxTokens:   ai.currentPrompt.MaxTokens,
		Temperature: ai.currentPrompt.Temperature,
		System:      ai.currentPrompt.Prompt,
		Messages:    messages,
	}

	b, err := json.Marshal(req)
	if err != nil {
		log.Panicln(err)
	}

	request, err := http.NewRequest("POST", ANTHROPIC_API_URL, bytes.NewBuffer(b))
	request.Header.Set(ANTHROPIC_AUTH_HEADER, ai.anthropicKey)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set(ANTHROPIC_VERSION_HEADER, ANTHROPIC_VERSION)

	resp, err := http.DefaultClient.Do(request)
	if err != nil {
		log.Panicln(err)
	}

	var anthropicResp AnthropicResponse

	err = json.NewDecoder(resp.Body).Decode(&anthropicResp)
	if err != nil {
		log.Panicln(err)
	}

	responseMessages := []AnthropicMessage{}

	for _, v := range anthropicResp.Content {
		responseMessages = append(responseMessages, AnthropicMessage{
			Role:    "assistant",
			Content: v.Text,
		})
	}

	messages = append(messages, responseMessages...)
	ai.currentMessages = messages

	ai.history[ai.currentPrompt.Prompt] = messages

	util.ToGob(&ai.history, filepath.Join(util.CacheDir(), aiHistoryFile))

	ai.items.Splice(0, int(ai.items.NItems()), messages...)

	glib.IdleAdd(func() {
		ai.list.ScrollTo(uint(queryPos), gtk.ListScrollNone, nil)
		ai.spinner.SetVisible(false)
	})
}

func (ai *AI) CopyLastResponse() {
	cmd := exec.Command("wl-copy")
	cmd.Stdin = strings.NewReader(ai.currentMessages[len(ai.currentMessages)-1].Content)

	err := cmd.Start()
	if err != nil {
		log.Println(err)
		return
	}
}

func (ai *AI) SpecialFunc(args ...interface{}) {
	provider := args[0].(string)
	prompt := args[1].(*config.AnthropicPrompt)
	query := args[2].(string)
	list := args[3].(*gtk.ListView)
	items := args[4].(*gioutil.ListModel[AnthropicMessage])
	spinner := args[5].(*gtk.Spinner)

	if ai.currentPrompt == nil {
		ai.currentPrompt = prompt
		ai.canProcess = true
		ai.list = list
		ai.items = items
		ai.spinner = spinner
		return
	}

	switch provider {
	case "anthropic":
		ai.anthropic(query)
	}
}

func (ai *AI) RunLastMessageInTerminal() {
	last := ai.currentMessages[len(ai.currentMessages)-1].Content
	shell := os.Getenv("SHELL")

	toRun := fmt.Sprintf("%s --title %s -e sh -c \"%s; exec %s\"", ai.terminal, "WalkerRunner", last, shell)
	cmd := exec.Command("sh", "-c", toRun)

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
}
