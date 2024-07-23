package modules

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/abenz1267/walker/config"
	"github.com/abenz1267/walker/util"
)

type Workable interface {
	Entries(ctx context.Context, term string) []Entry
	Prefix() string
	Name() string
	Placeholder() string
	SwitcherOnly() bool
	IsSetup() bool
	Setup(cfg *config.Config) bool
	SetupData(cfg *config.Config)
	Refresh()
	KeepSort() bool
	Typeahead() bool
	History() bool
}

type Entry struct {
	Categories       []string          `json:"categories,omitempty"`
	Class            string            `json:"class,omitempty"`
	DragDrop         bool              `json:"drag_drop,omitempty"`
	DragDropData     string            `json:"drag_drop_data,omitempty"`
	Exec             string            `json:"exec,omitempty"`
	ExecAlt          string            `json:"exec_alt,omitempty"`
	HideText         bool              `json:"hide_text,omitempty"`
	Icon             string            `json:"icon,omitempty"`
	IconIsImage      bool              `json:"icon_is_image,omitempty"`
	Image            string            `json:"image,omitempty"`
	InitialClass     string            `json:"initial_class,omitempty"`
	Label            string            `json:"label,omitempty"`
	MatchFields      int               `json:"match_fields,omitempty"`
	Matching         util.MatchingType `json:"matching,omitempty"`
	Path             string            `json:"path,omitempty"`
	RecalculateScore bool              `json:"recalculate_score,omitempty"`
	ScoreFinal       float64           `json:"score_final,omitempty"`
	ScoreFuzzy       float64           `json:"score_fuzzy,omitempty"`
	Searchable       string            `json:"searchable,omitempty"`
	SpecialLabel     string            `json:"special_label,omitempty"`
	Sub              string            `json:"sub,omitempty"`
	Terminal         bool              `json:"terminal,omitempty"`

	// internal
	DaysSinceUsed int       `json:"-"`
	History       bool      `json:"-"`
	LastUsed      time.Time `json:"-"`
	Module        string    `json:"-"`
	OpenWindows   uint      `json:"-"`
	Piped         Piped     `json:"-"`
	PipedAlt      Piped     `json:"-"`
	Used          int       `json:"-"`
}

func (e Entry) Identifier() string {
	str := fmt.Sprintf("%s %s %s %s", e.Label, e.Sub, e.Searchable, strings.Join(e.Categories, " "))

	hash := md5.Sum([]byte(str))
	return hex.EncodeToString(hash[:])
}

type Piped struct {
	Content string `json:"content,omitempty"`
	Type    string `json:"type,omitempty"`
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
			log.Panicln(err)
		}

		err = json.Unmarshal(b, &data)
		if err != nil {
			log.Panicln(err)
		}

		return true
	}

	return false
}

func Find(plugins []config.Plugin, name string) (config.Plugin, error) {
	for _, v := range plugins {
		if v.Name == name {
			return v, nil
		}
	}

	return config.Plugin{}, errors.New("plugin not found")
}
