package translation

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
)

type GoogleFree struct{}

func (googlefree *GoogleFree) Name() string {
	return "googlefree"
}

func (googlefree *GoogleFree) Translate(text, src, dest string) string {
	if text == "" {
		return ""
	}

	baseURL := "https://translate.googleapis.com/translate_a/single"

	params := url.Values{}
	params.Add("client", "gtx")
	params.Add("sl", src)
	params.Add("tl", dest)
	params.Add("dt", "t")
	params.Add("q", text)

	req, err := http.NewRequest("GET", baseURL+"?"+params.Encode(), nil)
	if err != nil {
		return text
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "*/*")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return text
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return text
	}

	var result []interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return text
	}

	if len(result) > 0 {
		if translations, ok := result[0].([]interface{}); ok {
			var translatedParts []string
			for _, trans := range translations {
				if transArray, ok := trans.([]interface{}); ok && len(transArray) > 0 {
					if translatedText, ok := transArray[0].(string); ok {
						translatedParts = append(translatedParts, translatedText)
					}
				}
			}
			return strings.Join(translatedParts, "")
		}
	}

	return text
}
