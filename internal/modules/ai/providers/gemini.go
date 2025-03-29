package providers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"os"

	"github.com/abenz1267/walker/internal/config"
	"github.com/abenz1267/walker/internal/util"
	"github.com/diamondburned/gotk4/pkg/core/gioutil"
)

const (
	GEMINI_ENV_KEY = "GEMINI_API_KEY"
	GEMINI_API_URL = "https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s"
)

func NewGeminiProvider(config config.AI, specialFunc func(args ...interface{})) Provider {
	apiKey := os.Getenv(GEMINI_ENV_KEY)
	if apiKey == "" {
		log.Println("gemini: no api key set")
		return nil
	}
	return &GeminiProvider{
		apiKey:      apiKey,
		config:      config,
		specialFunc: specialFunc,
	}
}

type GeminiProvider struct {
	apiKey      string
	config      config.AI
	specialFunc func(args ...interface{})
}

type GeminiRequest struct {
	GenerationConfig  GenerationConfig `json:"generationConfig"`
	SystemInstruction Content          `json:"system_instruction"`
	Contents          []Content        `json:"contents"`
}

type GenerationConfig struct {
	Temperature     float64 `json:"temperature"`
	MaxOutputTokens int     `json:"maxOutputTokens"`
}

type Content struct {
	Role  string `json:"role"`
	Parts []Part `json:"parts"`
}

type Part struct {
	Text string `json:"text"`
}

type GeminiResponse struct {
	Candidates []Candidate `json:"candidates"`
}

type Candidate struct {
	Content Content `json:"content"`
}

func messagesToContents(messages []Message) []Content {
	contents := make([]Content, len(messages))
	for i, msg := range messages {
		contents[i] = Content{
			Role: msg.Role,
			Parts: []Part{
				{Text: msg.Content},
			},
		}
	}
	return contents
}

func (p *GeminiProvider) Query(query string, currentMessages *[]Message, currentPrompt *config.AIPrompt, items *gioutil.ListModel[Message]) {
	url := fmt.Sprintf(GEMINI_API_URL, currentPrompt.Model, p.apiKey)

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

	if currentPrompt.MaxTokens == 0 {
		currentPrompt.MaxTokens = 1000
	}
	if currentPrompt.Temperature == 0 {
		currentPrompt.Temperature = 1
	}
	data := GeminiRequest{
		Contents: messagesToContents(messages),
		GenerationConfig: GenerationConfig{
			Temperature:     currentPrompt.Temperature,
			MaxOutputTokens: currentPrompt.MaxTokens,
		},
	}
	if currentPrompt.Prompt != "" {
		data.SystemInstruction = Content{Parts: []Part{{Text: currentPrompt.Prompt}}}
	}

	payload, err := json.Marshal(data)
	if err != nil {
		slog.Error("Error marshaling JSON: %v", err)
		return
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payload))
	if err != nil {
		slog.Error("Error creating request: %v", err)
		return
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		slog.Error("Error sending request: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		slog.Error("Error: received unexpected status code %d", resp.StatusCode)
		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		slog.Error("Error reading response body: %v", err)
	}

	var geminiResp GeminiResponse
	err = json.Unmarshal(body, &geminiResp)
	if err != nil {
		slog.Error("Error unmarshalling response: %v", err)
		return
	}
	if len(geminiResp.Candidates) == 0 {
		return
	}
	var responseMessages []Message
	for _, candidate := range geminiResp.Candidates {
		for _, part := range candidate.Content.Parts {
			responseMessages = append(responseMessages, Message{
				Role:    "model",
				Content: part.Text,
			})
		}
	}
	messages = append(messages, responseMessages...)
	*currentMessages = messages
}

func (p *GeminiProvider) SetupData() []util.Entry {
	var entries []util.Entry

	for _, v := range p.config.Gemini.Prompts {
		entries = append(entries, util.Entry{
			Label:            v.Label,
			Sub:              "Gemini: " + v.Model,
			Exec:             "",
			RecalculateScore: true,
			Matching:         util.Fuzzy,
			SpecialFunc:      p.specialFunc,
			SpecialFuncArgs:  []interface{}{"gemini", &v},
			SingleModuleOnly: v.SingleModuleOnly,
		})
	}

	if len(p.config.Gemini.Prompts) == 0 {
		log.Println("gemini: no prompts set.")
	}

	return entries
}
