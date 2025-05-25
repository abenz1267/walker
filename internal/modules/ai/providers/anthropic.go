package providers

import (
	"bytes"
	"encoding/json"
	"log"
	"log/slog"
	"net/http"
	"os"

	"github.com/abenz1267/walker/internal/config"
	"github.com/abenz1267/walker/internal/util"

	"github.com/diamondburned/gotk4/pkg/core/gioutil"
)

const (
	ANTHROPIC_VERSION_HEADER = "anthropic-version"
	ANTHROPIC_VERSION        = "2023-06-01"
	ANTHROPIC_API_URL        = "https://api.anthropic.com/v1/messages"
	ANTHROPIC_AUTH_HEADER    = "x-api-key"
	ANTHROPIC_API_KEY        = "ANTHROPIC_API_KEY"
)

type AnthropicProvider struct {
	config      config.AI
	key         string
	specialFunc func(args ...interface{})
}

func NewAnthropicProvider(config config.AI, specialFunc func(...interface{})) Provider {
	key := os.Getenv(ANTHROPIC_API_KEY)

	if key == "" {
		return nil
	}

	return &AnthropicProvider{
		config:      config,
		key:         os.Getenv(ANTHROPIC_API_KEY),
		specialFunc: specialFunc,
	}
}

func (p *AnthropicProvider) SetupData() []util.Entry {
	var entries []util.Entry

	for _, v := range p.config.Anthropic.Prompts {
		entries = append(entries, util.Entry{
			Label:            v.Label,
			Sub:              "Anthropic Claude 3.5",
			Exec:             "",
			RecalculateScore: true,
			Matching:         util.Fuzzy,
			SpecialFunc:      p.specialFunc,
			SpecialFuncArgs:  []interface{}{"anthropic", &v},
			SingleModuleOnly: v.SingleModuleOnly,
		})
	}

	if len(p.config.Anthropic.Prompts) == 0 {
		log.Println("anthropic: no prompts set.")
	}

	return entries
}

type AnthropicRequest struct {
	Model       string    `json:"model"`
	MaxTokens   int       `json:"max_tokens"`
	Temperature float64   `json:"temperature"`
	System      string    `json:"system"`
	Messages    []Message `json:"messages"`
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

func (p *AnthropicProvider) Query(query string, currentMessages *[]Message, currentPrompt *config.AIPrompt, items *gioutil.ListModel[Message]) {
	queryMsg := Message{
		Role:    "user",
		Content: query,
	}

	messages := []Message{}

	if currentMessages != nil && len(*currentMessages) > 0 {
		messages = *currentMessages
	}

	messages = append(messages, queryMsg)

	items.Splice(0, int(items.NItems()), messages...)

	req := AnthropicRequest{
		Model:       currentPrompt.Model,
		MaxTokens:   currentPrompt.MaxTokens,
		Temperature: currentPrompt.Temperature,
		System:      currentPrompt.Prompt,
		Messages:    messages,
	}

	b, err := json.Marshal(req)
	if err != nil {
		log.Panicln(err)
	}

	request, err := http.NewRequest("POST", ANTHROPIC_API_URL, bytes.NewBuffer(b))
	if err != nil {
		log.Panicln(err)
	}
	request.Header.Set(ANTHROPIC_AUTH_HEADER, p.key)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set(ANTHROPIC_VERSION_HEADER, ANTHROPIC_VERSION)

	resp, err := http.DefaultClient.Do(request)
	if err != nil {
		slog.Error("Error making request: %v", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		slog.Error("Anthropic API returned unexpected status code %d", resp.StatusCode)
		return
	}

	var anthropicResp AnthropicResponse

	err = json.NewDecoder(resp.Body).Decode(&anthropicResp)
	if err != nil {
		slog.Error("Error decoding response: %v", err)
		return
	}

	responseMessages := []Message{}

	for _, v := range anthropicResp.Content {
		responseMessages = append(responseMessages, Message{
			Role:    "assistant",
			Content: v.Text,
		})
	}

	messages = append(messages, responseMessages...)
	*currentMessages = messages
}
