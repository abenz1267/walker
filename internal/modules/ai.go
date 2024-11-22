package modules

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/abenz1267/walker/internal/config"
	"github.com/abenz1267/walker/internal/util"
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
	config                config.AI
	entries               []util.Entry
	anthropicKey          string
	currentPrompt         string
	canProcess            bool
	currentMessages       []AnthropicMessage
	currentScroll         *gtk.ScrolledWindow
	history               map[string][]AnthropicMessage
	setupLabelWidgetStyle func(label *gtk.Label, style *config.LabelWidget)
	labelWidgetStyle      *config.LabelWidget
	scroll                *gtk.ScrolledWindow
}

func (ai *AI) Cleanup() {
	ai.currentPrompt = ""
	ai.currentMessages = []AnthropicMessage{}

	if ai.currentScroll == nil {
		return
	}

	glib.IdleAdd(func() {
		vp := ai.currentScroll.Child().(*gtk.Viewport)

		box := vp.Child().(*gtk.Box)
		for box.FirstChild() != nil {
			box.Remove(box.FirstChild())
		}

		ai.currentScroll = nil
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

	file := filepath.Join(util.CacheDir(), aiHistoryFile)

	ai.history = make(map[string][]AnthropicMessage)

	util.FromGob(file, &ai.history)

	return true
}

func (ai *AI) ResumeLastMessages() {
	ai.currentMessages = ai.history[ai.currentPrompt]
	box := ai.scroll.Child().(*gtk.Viewport).Child().(*gtk.Box)

	glib.IdleAdd(func() {
		for _, v := range ai.currentMessages {
			content := v.Content
			label := gtk.NewLabel(content)

			if v.Role == "user" {
				content = fmt.Sprintf(">> %s", content)
				label.SetText(content)
				label.SetCSSClasses([]string{"aiItem", "user"})
			} else {
				label.SetCSSClasses([]string{"aiItem", "assistant"})
			}

			label.SetSelectable(true)

			ai.setupLabelWidgetStyle(label, ai.labelWidgetStyle)

			box.Append(label)
		}
	})
}

func (ai *AI) ClearCurrent() {
	box := ai.scroll.Child().(*gtk.Viewport).Child().(*gtk.Box)
	ai.currentMessages = []AnthropicMessage{}

	glib.IdleAdd(func() {
		for box.FirstChild() != nil {
			box.Remove(box.FirstChild())
		}
	})
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
				SpecialFuncArgs:  []interface{}{"anthropic", v.Prompt},
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
	Model     string             `json:"model"`
	MaxTokens int                `json:"max_tokens"`
	System    string             `json:"system"`
	Messages  []AnthropicMessage `json:"messages"`
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
	box := ai.scroll.Child().(*gtk.Viewport).Child().(*gtk.Box)

	queryMsg := AnthropicMessage{
		Role:    "user",
		Content: query,
	}

	messages := []AnthropicMessage{}

	if len(ai.currentMessages) > 0 {
		messages = ai.currentMessages
	}

	messages = append(messages, queryMsg)
	spinner := gtk.NewSpinner()

	glib.IdleAdd(func() {
		l := gtk.NewLabel(fmt.Sprintf(">> %s", queryMsg.Content))
		l.SetCSSClasses([]string{"aiItem", "user"})
		l.SetSelectable(true)

		ai.setupLabelWidgetStyle(l, ai.labelWidgetStyle)

		box.Append(l)

		spinner.SetSpinning(true)

		box.Append(spinner)
	})

	req := AnthropicRequest{
		Model:     ai.config.Anthropic.Model,
		MaxTokens: ai.config.Anthropic.MaxTokens,
		System:    ai.currentPrompt,
		Messages:  messages,
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

	ai.history[ai.currentPrompt] = messages

	util.ToGob(&ai.history, filepath.Join(util.CacheDir(), aiHistoryFile))

	glib.IdleAdd(func() {
		box.Remove(spinner)

		for _, v := range responseMessages {
			label := gtk.NewLabel(v.Content)

			label.SetCSSClasses([]string{"aiItem", "assistant"})
			label.SetSelectable(true)

			ai.setupLabelWidgetStyle(label, ai.labelWidgetStyle)

			box.Append(label)
		}
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
	prompt := args[1].(string)
	query := args[2].(string)
	aiScroll := args[3].(*gtk.ScrolledWindow)
	setupLabelWidgetStyle := args[4].(func(label *gtk.Label, style *config.LabelWidget))
	labelWidgetStyle := args[5].(*config.LabelWidget)

	if ai.currentScroll == nil {
		ai.currentScroll = aiScroll
	}

	if ai.currentPrompt == "" {
		ai.currentPrompt = prompt
		ai.canProcess = true
		ai.setupLabelWidgetStyle = setupLabelWidgetStyle
		ai.labelWidgetStyle = labelWidgetStyle
		ai.scroll = aiScroll
		return
	}

	switch provider {
	case "anthropic":
		ai.anthropic(query)
	}
}
