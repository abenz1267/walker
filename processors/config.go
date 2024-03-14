package processors

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
)

type Processor struct {
	Prefix string `json:"prefix,omitempty"`
	Name   string `json:"name,omitempty"`
	Src    string `json:"src,omitempty"`
	Cmd    string `json:"cmd,omitempty"`
}

func readCache(name string, data any) bool {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		log.Println(err)
		return false
	}

	cacheDir = filepath.Join(cacheDir, "walker")

	path := filepath.Join(cacheDir, fmt.Sprintf("%s.json", name))

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

		err = json.Unmarshal(b, &data)
		if err != nil {
			log.Fatalln(err)
		}

		return true
	}

	return false
}

func writeCache(name string, data any) {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		log.Println(err)
		return
	}

	cacheDir = filepath.Join(cacheDir, "walker")

	b, err := json.Marshal(data)
	if err != nil {
		log.Println(err)
		return
	}

	err = os.MkdirAll(cacheDir, 0755)
	if err != nil {
		log.Println(err)
		return
	}

	err = os.WriteFile(filepath.Join(cacheDir, fmt.Sprintf("%s.json", name)), b, 0644)
	if err != nil {
		log.Println(err)
	}
}
