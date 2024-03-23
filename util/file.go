package util

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"io"
	"log"
	"os"
	"path/filepath"
)

func ToGob[T any](val *T, dest string) {
	var b bytes.Buffer
	encoder := gob.NewEncoder(&b)
	if err := encoder.Encode(val); err != nil {
		log.Fatalln(err)
	}

	writeFile(b.Bytes(), dest)
}

func FromGob[T any](src string, dest *T) bool {
	if _, err := os.Stat(src); err != nil {
		return false
	}

	file, err := os.Open(src)
	if err != nil {
		log.Fatalln(err)
	}

	b, err := io.ReadAll(file)
	if err != nil {
		log.Panic(err)
	}

	decoder := gob.NewDecoder(bytes.NewReader(b))
	err = decoder.Decode(dest)
	if err != nil {
		log.Fatalln(err)
	}

	return true
}

func ToJson[T any](src *T, dest string) {
	b, err := json.Marshal(src)
	if err != nil {
		log.Fatalln(err)
	}

	writeFile(b, dest)
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

func writeFile(b []byte, dest string) {
	err := os.MkdirAll(filepath.Dir(dest), 0755)
	if err != nil {
		log.Println(err)
		return
	}

	err = os.WriteFile(dest, b, 0o600)
	if err != nil {
		log.Fatalln(err)
	}
}
