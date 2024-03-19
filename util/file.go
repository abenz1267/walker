package util

import (
	"encoding/json"
	"io"
	"log"
	"os"
	"path/filepath"
)

func ToJson(src any, dest string) {
	err := os.MkdirAll(filepath.Dir(dest), 0755)
	if err != nil {
		log.Println(err)
		return
	}

	b, err := json.Marshal(src)
	if err != nil {
		log.Fatalln(err)
	}

	err = os.WriteFile(dest, b, 0o600)
	if err != nil {
		log.Fatalln(err)
	}
}

func FromJson[T any](src string, dest *T) bool {
	if _, err := os.Stat(src); err != nil {
		return false
	}

	file, err := os.Open(src)
	if err != nil {
		log.Fatalln(err)
	}

	b, err := io.ReadAll(file)
	if err != nil {
		log.Fatalln(err)
	}

	err = json.Unmarshal(b, dest)
	if err != nil {
		log.Fatalln(err)
	}

	return true
}

func TmpDir() string {
	return filepath.Join(os.TempDir())
}

func ConfigDir() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		log.Fatalln(err)
	}

	return filepath.Join(dir, "walker")
}

func CacheDir() string {
	dir, err := os.UserCacheDir()
	if err != nil {
		log.Fatalln(err)
	}

	return filepath.Join(dir, "walker")
}
