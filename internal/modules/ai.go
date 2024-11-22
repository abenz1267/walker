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
)

type AI struct {
	config           config.AI
	entries          []util.Entry
	anthropicKey     string
	currentPrompt    string
	currenctMessages []AnthropicMessage
	currentScroll    *gtk.ScrolledWindow
}

func (ai *AI) Cleanup() {
	ai.currentPrompt = ""
	ai.currenctMessages = []AnthropicMessage{}

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

	return true
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
				ScoreFinal:       100,
				RecalculateScore: false,
				Matching:         util.AlwaysTop,
				SpecialFunc:      ai.SpecialFunc,
				SpecialFuncArgs:  []interface{}{"anthropic", v.Prompt},
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

func (ai *AI) anthropic(query string, prompt string, scroll *gtk.ScrolledWindow, setupLabelWidgetStyle func(label *gtk.Label, style *config.LabelWidget), labelWidgetStyle *config.LabelWidget, spinner *gtk.Spinner) {
	box := scroll.Child().(*gtk.Viewport).Child().(*gtk.Box)

	queryMsg := AnthropicMessage{
		Role:    "user",
		Content: query,
	}

	messages := []AnthropicMessage{}

	if len(ai.currenctMessages) > 0 {
		messages = ai.currenctMessages
	}

	messages = append(messages, queryMsg)

	glib.IdleAdd(func() {
		l := gtk.NewLabel(fmt.Sprintf(">> %s", queryMsg.Content))
		l.SetCSSClasses([]string{"aiItem", "user"})
		l.SetSelectable(true)

		setupLabelWidgetStyle(l, labelWidgetStyle)

		box.Append(l)
	})

	req := AnthropicRequest{
		Model:     ai.config.Anthropic.Model,
		MaxTokens: ai.config.Anthropic.MaxTokens,
		System:    prompt,
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
	ai.currenctMessages = messages

	glib.IdleAdd(func() {
		spinner.SetVisible(false)

		for _, v := range responseMessages {
			label := gtk.NewLabel(v.Content)

			label.SetCSSClasses([]string{"aiItem", "assistant"})
			label.SetSelectable(true)

			setupLabelWidgetStyle(label, labelWidgetStyle)

			box.Append(label)
		}
	})
}

func (ai *AI) CopyLastResponse() {
	cmd := exec.Command("wl-copy")
	cmd.Stdin = strings.NewReader(ai.currenctMessages[len(ai.currenctMessages)-1].Content)

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
	spinner := args[6].(*gtk.Spinner)

	if ai.currentPrompt != "" {
		prompt = ai.currentPrompt
	}

	if ai.currentScroll == nil {
		ai.currentScroll = aiScroll
	}

	switch provider {
	case "anthropic":
		ai.anthropic(query, prompt, aiScroll, setupLabelWidgetStyle, labelWidgetStyle, spinner)
	}
}
