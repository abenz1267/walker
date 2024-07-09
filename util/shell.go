package util

import (
	"unicode"
)

func ParseShellCommand(cmd string) (string, []string) {
	words := []string{}

	currentWord := ""
	isEscaped := false
	isQuote := false
	for _, c := range cmd {
		if isEscaped {
			currentWord += string(c)
			isEscaped = false
			continue
		}

		if c == '\\' {
			isEscaped = true
			continue
		}

		if c == '"' || c == '\'' {
			isQuote = !isQuote
			continue
		}

		if unicode.IsSpace(c) && !isQuote {
			words = append(words, currentWord)
			currentWord = ""
			continue
		}

		currentWord += string(c)
	}

	if currentWord != "" {
		words = append(words, currentWord)
	}

	if len(words) == 0 {
		return "", []string{}
	}

	return words[0], words[1:]
}

