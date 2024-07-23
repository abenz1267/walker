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
}

type Entry struct {
	Label            string            `json:"label,omitempty"`
	Sub              string            `json:"sub,omitempty"`
	Exec             string            `json:"exec,omitempty"`
	ExecAlt          string            `json:"exec_alt,omitempty"`
	Terminal         bool              `json:"terminal,omitempty"`
	Piped            Piped             `json:"-"`
	PipedAlt         Piped             `json:"-"`
	Icon             string            `json:"icon,omitempty"`
	IconIsImage      bool              `json:"icon_is_image,omitempty"`
	DragDrop         bool              `json:"drag_drop,omitempty"`
	DragDropData     string            `json:"drag_drop_data,omitempty"`
	Image            string            `json:"image,omitempty"`
	HideText         bool              `json:"hide_text,omitempty"`
	Categories       []string          `json:"categories,omitempty"`
	Searchable       string            `json:"searchable,omitempty"`
	MatchFields      int               `json:"match_fields,omitempty"`
	Class            string            `json:"class,omitempty"`
	History          bool              `json:"history,omitempty"`
	Matching         util.MatchingType `json:"matching,omitempty"`
	RecalculateScore bool              `json:"recalculate_score,omitempty"`
	ScoreFinal       float64           `json:"score_final,omitempty"`
	ScoreFuzzy       float64           `json:"score_fuzzy,omitempty"`
	Used             int               `json:"-"`
	DaysSinceUsed    int               `json:"-"`
	SpecialLabel     string            `json:"special_label,omitempty"`
	LastUsed         time.Time         `json:"-"`
	InitialClass     string            `json:"initial_class,omitempty"`
	OpenWindows      uint              `json:"-"`
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
