package history

import (
	"encoding/json"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"
)

type History map[string]Entry

type Entry struct {
	LastUsed      time.Time `json:"last_used,omitempty"`
	Used          int       `json:"used,omitempty"`
	DaysSinceUsed int
}

func Get() History {
	history := make(History)

	cacheDir, err := os.UserCacheDir()
	if err != nil {
		log.Println(err)
		log.Fatalf("failed to get cache dir: %s", err)
	}

	cacheDir = filepath.Join(cacheDir, "walker")

	path := filepath.Join(cacheDir, "history.json")

	if _, err := os.Stat(path); err == nil {
		file, err := os.Open(path)
		if err != nil {
			log.Println(err)
		}
		defer file.Close()

		b, err := io.ReadAll(file)
		if err != nil {
			log.Fatalln(err)
		}

		err = json.Unmarshal(b, &history)
		if err != nil {
			log.Fatalln(err)
		}

		for k, v := range history {
			today := time.Now()
			v.DaysSinceUsed = int(today.Sub(v.LastUsed).Hours() / 24)

			history[k] = v
		}
	}

	return history
}
