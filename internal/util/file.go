package util

import (
	"bytes"
	"encoding/gob"
	"io"
	"log"
	"os"
	"path/filepath"
)

func ToGob[T any](val *T, dest string) {
	var b bytes.Buffer
	encoder := gob.NewEncoder(&b)
	if err := encoder.Encode(val); err != nil {
		log.Panicln(err)
	}

	writeFile(b.Bytes(), dest)
}

func FromGob[T any](src string, dest *T) bool {
	if _, err := os.Stat(src); err != nil {
		return false
	}

	file, err := os.Open(src)
	if err != nil {
		log.Panicln(err)
	}

	defer file.Close()

	b, err := io.ReadAll(file)
	if err != nil {
		log.Panic(err)
	}

	decoder := gob.NewDecoder(bytes.NewReader(b))

	err = decoder.Decode(dest)
	if err != nil {
		log.Println(err)

		log.Printf("cache file %s is malformed, truncating.\n", src)

		err = os.Truncate(src, 0)
		if err != nil {
			log.Panicln(err)
		}

		return false
	}

	return true
}

func TmpDir() string {
	return filepath.Join(os.TempDir())
}

func ThemeDir() string {
	return filepath.Join(ConfigDir(), "themes")
}

func ConfigDir() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		log.Panicln(err)
	}

	return filepath.Join(dir, "walker")
}

func CacheDir() string {
	dir, err := os.UserCacheDir()
	if err != nil {
		log.Panicln(err)
	}

	return filepath.Join(dir, "walker")
}

func ThumbnailsDir() string {
	return filepath.Join(CacheDir(), "thumbnails")
}

func writeFile(b []byte, dest string) {
	err := os.MkdirAll(filepath.Dir(dest), 0755)
	if err != nil {
		log.Println(err)
		return
	}

	err = os.WriteFile(dest, b, 0o600)
	if err != nil {
		log.Panicln(err)
	}
}

func FileExists(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil
}
