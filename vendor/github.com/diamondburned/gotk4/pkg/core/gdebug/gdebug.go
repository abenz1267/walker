package gdebug

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"
)

var debug = strings.Split(os.Getenv("GOTK4_DEBUG"), ",")

func HasKey(key string) bool {
	for _, k := range debug {
		if k == key {
			return true
		}
	}
	return false
}

func NewDebugLogger(key string) *log.Logger {
	if !HasKey(key) {
		return log.New(io.Discard, "", 0)
	}
	return mustDebugLogger(key)
}

func NewDebugLoggerNullable(key string) *log.Logger {
	if !HasKey(key) {
		return nil
	}
	return mustDebugLogger(key)
}

func mustDebugLogger(name string) *log.Logger {
	if HasKey("to-console") {
		return log.Default()
	}

	f, err := os.CreateTemp(os.TempDir(), fmt.Sprintf("gotk4-%s-%d-*", name, os.Getpid()))
	if err != nil {
		log.Panicln("cannot create temp", name, "file:", err)
	}

	log.Println("gotk4: intern: enabled debug file at", f.Name())
	return log.New(f, "", log.LstdFlags)
}
