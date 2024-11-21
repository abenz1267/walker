package modules

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

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
	config       config.AI
	entries      []util.Entry
	anthropicKey string
}

func (ai *AI) Cleanup() {
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

func (ai *AI) anthropic(query string, prompt string, scroll *gtk.ScrolledWindow, setupLabelWidgetStyle func(label *gtk.Label, style *config.LabelWidget), labelWidgetStyle *config.LabelWidget) {
	req := AnthropicRequest{
		Model:     ai.config.Anthropic.Model,
		MaxTokens: ai.config.Anthropic.MaxTokens,
		System:    prompt,
		Messages: []AnthropicMessage{
			{
				Role:    "user",
				Content: query,
			},
		},
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

	messages := []AnthropicMessage{}
	messages = append(messages, AnthropicMessage{
		Role:    "user",
		Content: query,
	})

	messages = append(messages, AnthropicMessage{
		Role:    "assistant",
		Content: anthropicResp.Content[0].Text,
	})

	glib.IdleAdd(func() {
		box := scroll.Child().(*gtk.Viewport).Child().(*gtk.Box)
		spinner := box.FirstChild().(*gtk.Spinner)
		spinner.SetVisible(false)

		for _, v := range messages {
			content := v.Content

			if v.Role == "user" {
				content = fmt.Sprintf(">> %s", content)
			}

			label := gtk.NewLabel(content)

			label.SetWrap(true)

			if v.Role == "user" {
				label.SetCSSClasses([]string{"aiItem", "user"})
			} else {
				label.SetCSSClasses([]string{"aiItem", "assistant"})
			}

			setupLabelWidgetStyle(label, labelWidgetStyle)

			box.Append(label)
		}

		scroll.SetChild(box)
	})
}

func (ai *AI) SpecialFunc(args ...interface{}) {
	provider := args[0].(string)
	prompt := args[1].(string)
	query := args[2].(string)
	aiScroll := args[3].(*gtk.ScrolledWindow)
	setupLabelWidgetStyle := args[4].(func(label *gtk.Label, style *config.LabelWidget))
	labelWidgetStyle := args[5].(*config.LabelWidget)

	switch provider {
	case "anthropic":
		ai.anthropic(query, prompt, aiScroll, setupLabelWidgetStyle, labelWidgetStyle)
	}
}
