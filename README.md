# Walker - Application Runner

![Screenshot](https://github.com/abenz1267/walker/blob/master/screenshot.png?raw=true)

(default style)

## Features

- Icons
- notifications on failure configurable
- extend with... anything?
- start as service for faster startup (see benchmarks below)
- activation-mode: run entries via labels
- display images
- non-blocking async handling of results

## Builtin Modules

- runner
- desktop applications
- websearch
- hyprland windows

## Requirements

- gtk4-layer-shell

## Installation

If you have problems installing `gtk4-layer-shell`, try switching your GTK4 theme to a default one. You can switch back after installing.

**_Building can take quite a while, be patient_**

```
arch:
yay -S walker
```

## Config & Style

Default config will be put into `$HOME/.config/walker/`.

See `config/config.default.json` and `ui/style.default.css`.

## Providing your own modules

If you want to extend walker with your own modules, you can do that in the config.

```json
{
  "external": [
    {
      "prefix": "!",
      "name": "mymodule",
      "src": "node /path/to/myscript.js"
    }
  ]
}
```

Your plugin simply needs to return json with the following format:

```go
type Entry struct {
	Label             string       `json:"label,omitempty"`
	Sub               string       `json:"sub,omitempty"`
	Exec              string       `json:"exec,omitempty"`
	Terminal          bool         `json:"terminal,omitempty"`
	Icon              string       `json:"icon,omitempty"`
	IconIsImage       bool         `json:"icon_is_image,omitempty"`
	HideText          bool         `json:"hide_text,omitempty"`
	Categories        []string     `json:"categories,omitempty"`
	Notifyable        bool         `json:"notifyable,omitempty"`
	Class             string       `json:"class,omitempty"`
	History           bool         `json:"history,omitempty"`
	HistoryIdentifier string       `json:"history_identifier,omitempty"`
	Matching          MatchingType `json:"matching,omitempty"`
	ScoreFinal        float64      `json:"score_final,omitempty"`
	MinScoreToInclude float64      `json:"min_score_to_include,omitempty"`
	ScoreFuzzy        int          `json:"score_fuzzy,omitempty"`
}
```

F.e.:

```json
[
  {
    "label": "First Item",
    "exec": "remindme in 1s test",
    "searchable": "first item",
    "notifyable": true
  }
]
```

You can also do:

```json
{
  "name": "filesystem",
  "prefix": "/",
  "src": "fd --base-directory /home/andrej/ %TERM%",
  "cmd": "xdg-open file://%RESULT%",
  "transform": true
}
```

### Dynamic Styling

The window and items will have a class based on the source. Selecting an item will change the windows class to the current selections source. Using a prefix will apply that sources classes to the window.

F.e. search = `!somecommand` => `#window.runner`

### Starting as service

Start with `walker --gapplication-service` to start in service-mode. Calling `walker` normally afterwards should be rather fast.

## Keybinds

AM = Activation Mode

| Key                                                                     | Description                        |
| ----------------------------------------------------------------------- | ---------------------------------- |
| `Enter`                                                                 | activate selection                 |
| `Ctrl+Enter`                                                            | activate selection without closing |
| `Ctrl+j` (if ActivationMode is disabled), `Down`, `Tab`                 | next entry                         |
| `Ctrl+k` (if ActivationMode is disabled), `Up`, `LEFT_TAB` (shift+tab?) | previous entry                     |
| `Escape`                                                                | close                              |
| `Ctrl`                                                                  | start AM                           |
| in AM: `<label>`                                                        | activate item                      |
| in AM: Hold `Ctrl+<label>`                                              | activate item (don't close)        |
| in AM: `Escape`                                                         | stop AM                            |

### Activation Mode

Activation-Mode can be triggered by holding `LCtrl`. The window will get an additional class `activation` you can use for styling. While activated, you can run items by pressing their respective label. This only works for the top 8 items.

## Startup "Benchmarks"

System: Arch Linux, Hyprland, Amd 7950x, 32gb DDR5-6000, SSD

Measured time is until the focus is in the search-bar and you can type.

| Mode         | Startup time                                     |
| ------------ | ------------------------------------------------ |
| normal       | 37ms                                             |
| with service | < 500µs / (2.3ms when input needs to be cleared) |

## Watchout for...

- Desktop entries will be parsed and cached in `.cache/walker`... currently no mechanism to refresh cache, so delete manually if needed
