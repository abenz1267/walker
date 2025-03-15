package providers

import (
	"github.com/abenz1267/walker/internal/config"
	"github.com/abenz1267/walker/internal/util"
	"github.com/diamondburned/gotk4/pkg/core/gioutil"
)

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type Provider interface {
	Query(query string, currentMessages *[]Message, currentPrompt *config.AIPrompt, items *gioutil.ListModel[Message])
	SetupData() []util.Entry
}
