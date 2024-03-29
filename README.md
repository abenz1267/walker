# Walker - Application Runner

![Screenshot](https://github.com/abenz1267/walker/blob/master/screenshot.png?raw=true)

(default style)

![Screenshot](https://github.com/abenz1267/walker/blob/master/screenshot_horizontal.png?raw=true)

(horizontal example)

## Features

- Icons
- extend with... anything?
- start as service for faster startup (see benchmarks below)
- activation-mode: run entries via labels
- display images
- non-blocking async handling of results
- typeahead (currently not persisted)
- history-aware

## Builtin Modules

- runner
- desktop applications
- websearch
- hyprland windows
- clipboard with fuzzy find and images (currently "wl-clipboard" only)
- module switcher
- commands (for Walker, f.e. clear cache)
- ssh
- finder (requires fzf and fd) with drag&drop

## Requirements

- gtk4-layer-shell

## Installation

If you have problems installing `gtk4-layer-shell`, try switching your GTK4 theme to a default one. You can switch back after installing.

**_Building can take quite a while, be patient_**

```
arch:
yay -S walker
```

<details>
<summary>Install using Nix</summary>

You have two options of installing walker using Nix.

1.  Using the package exposed by this flake

    1. Add to your flake `inputs.walker.url = "github:abenz1267/walker";`
    2. Add `inputs.walker.packages.<system>.default` to `environment.systemPackages` or `home.packages`

2.  Using the home-manager module exposed by this flake:

    1. Add to your flake `inputs.walker.url = "github:abenz1267/walker";`
    2. Add `imports = [inputs.walker.homeManagerModules.walker];` into your home-manager config
    3. Configure walker using:

```nix
programs.walker = {
  enable = true;
  runAsService = true;

  # All options from the config.json can be used here.
  config = {
    placeholder = "Example";
    fullscreen = true;
    list = {
      height = 200;
    };
    modules = [
      {
        name = "websearch";
        prefix = "?";
      }
      {
        name = "switcher";
        prefix = "/";
      }
    ];
  };

  # If this is not set the default styling is used.
  style = ''
    * {
      color: #dcd7ba;
    }
  '';
};
```

Additionally, there is a binary cache at https://walker.cachix.org which you can use with the following:

```nix
nix.settings = {
  substituters = ["https://walker.cachix.org"];
  trusted-public-keys = ["walker.cachix.org-1:fG8q+uAaMqhsMxWjwvk0IMb4mFPFLqHjuvfwQxE4oJM="];
};
```

</details>

## Config & Style

Default config will be put into `$HOME/.config/walker/`.

See `config/config.default.json` and `ui/style.default.css`. Styling is done via GTK CSS.

Definition for modules:

```go
type Module struct {
	Prefix            string `json:"prefix,omitempty"`
	Name              string `json:"name,omitempty"`
	Src               string `json:"src,omitempty"`
	Cmd               string `json:"cmd,omitempty"`
	Transform         bool   `json:"transform,omitempty"`
	History           bool   `json:"history,omitempty"`
	SwitcherExclusive bool   `json:"switcher_exclusive,omitempty"`
}
```

## Usage SSH Module

In the searchbar type in: `<host> <username>`. Select the host you want. Enter.

## Styling with typeahead enabled

If you have typeahead enabled, make sure that your `#search` has no background, so the typeahead is readable.

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
	RawExec           []string     `json:"raw_exec,omitempty"`
	Terminal          bool         `json:"terminal,omitempty"`
	Piped             Piped        `json:"piped,omitempty"`
	Icon              string       `json:"icon,omitempty"`
	IconIsImage       bool         `json:"icon_is_image,omitempty"`
	Image             string       `json:"image,omitempty"`
	HideText          bool         `json:"hide_text,omitempty"`
	Categories        []string     `json:"categories,omitempty"`
	Searchable        string       `json:"searchable,omitempty"`
	Class             string       `json:"class,omitempty"`
	History           bool         `json:"history,omitempty"`
	HistoryIdentifier string       `json:"history_identifier,omitempty"`
	Matching          MatchingType `json:"matching,omitempty"`
	RecalculateScore  bool         `json:"recalculate_score,omitempty"`
	ScoreFinal        float64      `json:"score_final,omitempty"`
	ScoreFuzzy        int          `json:"score_fuzzy,omitempty"`
	Used              int          `json:"-"`
	DaysSinceUsed     int          `json:"-"`
	LastUsed          time.Time    `json:"-"`
}
```

F.e.:

```json
[
  {
    "label": "First Item",
    "exec": "remindme in 1s test",
    "searchable": "first item"
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
| `Shift+Enter`                                                           | activate selection without closing |
| `Ctrl+j` (if ActivationMode is disabled), `Down`, `Tab`                 | next entry                         |
| `Ctrl+k` (if ActivationMode is disabled), `Up`, `LEFT_TAB` (shift+tab?) | previous entry                     |
| `Escape`                                                                | close                              |
| `Ctrl`                                                                  | start AM                           |
| in AM: `<label>`                                                        | activate item                      |
| in AM: Hold `Shift+<label>`                                             | activate item (don't close)        |
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

## FAQ

### "lockfile exists" - i can't run Walker.

This happens if Walker get's closed unexpectedly, f.e. via SIGKILL. Remove `/tmp/walker.lock` manually and try again.
