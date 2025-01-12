package translation

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strings"
)

type DeeplFree struct {
	key string
}

func (deepl *DeeplFree) Name() string {
	return "deeplfree"
}

func (deepl *DeeplFree) Translate(text, src, dest string) string {
	if deepl.key == "" {
		deepl.SetupAPIKey()
	}

	if deepl.key == "" {
		slog.Error("deeplfree", "no api key set")
		return ""
	}

	if text == "" {
		return ""
	}

	baseURL := "https://api-free.deepl.com/v2/translate"

	// Convert language codes if needed (DeepL uses different codes than Google)
	if src == "auto" {
		src = ""
	}

	if dest == "zh" {
		dest = "zh-CN"
	}

	data := url.Values{}
	data.Set("text", text)

	if src != "" {
		data.Set("source_lang", strings.ToUpper(src))
	}

	data.Set("target_lang", strings.ToUpper(dest))

	req, err := http.NewRequest("POST", baseURL, strings.NewReader(data.Encode()))
	if err != nil {
		return text
	}

	req.Header.Set("Authorization", "DeepL-Auth-Key "+deepl.key)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return text
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return text
	}

	var result struct {
		Translations []struct {
			Text string `json:"text"`
		} `json:"translations"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return text
	}

	if len(result.Translations) > 0 {
		return result.Translations[0].Text
	}

	return text
}

func (deepl *DeeplFree) SetupAPIKey() {
	key := os.Getenv("DEEPL_AUTH_KEY")
	deepl.key = key
}
