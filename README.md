# Walker - Application Runner

WIP.

![Screenshot](https://github.com/abenz1267/walker/blob/master/screenshot.png?raw=true)

## Features

- Desktop Entries with actions
- Runner (default prefix: `!`)
- Websearch (default prefix: `?`)
- Icons
- notifications on failure configurable
- extend with... anything?

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

Config in `.config/walker/`.

See `config.example.json` and `style.example.css`.

```json
{
  "terminal": "foot",
  "placeholder": "Search...",
  "shell_config": "/home/andrej/.zshrc", // for parsing shell aliases
  "notify_on_fail": true,
  "show_initial_entries": true, // always shows entries for emtpy search
  "fullscreen": true,
  "search": {
    "delay": 150, // in ms
    "hide_icons": true
  },
  "align": {
    "width": 400,
    "horizontal": "center", // fill, start, end, center
    "vertical": "start", // fill, start, end, center
    "margins": {
      "top": 300,
      "bottom": 0,
      "end": 0,
      "start": 0
    }
  },
  "list": {
    "height": 300,
    "style": "fixed", // dynamic, fixed
    "always_show": true
  },
  "orientation": "vertical", // vertical, horizontal
  "icons": {
    "hide": false,
    "size": 38
  },
  "processors": [
    {
      "name": "runner",
      "prefix": "!"
    },
    {
      "name": "applications",
      "prefix": ""
    },
    {
      "name": "websearch",
      "prefix": "?"
    }
  ]
}
```

## Providing your own processors

If you want to extend walker with your own processors, you can do that in the config.

```json
{
  "processors": [
    {
      "prefix": "!",
      "name": "myprocessor",
      "cmd": "node /path/to/myscript.js"
    }
  ]
}
```

Your plugin simply needs to return json with the following format:

```go
type Entry struct {
	Label      string `json:"label,omitempty"`
	Sub        string `json:"sub,omitempty"`
	Exec       string `json:"exec,omitempty"`
	Terminal   bool   `json:"terminal,omitempty"`
	Icon       string `json:"icon,omitempty"`
	Searchable string `json:"searchable,omitempty"`
	Notifyable bool   `json:"notifyable,omitempty"`
	Class      string `json:"class,omitempty"`
}
```

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
  "src": "fd --base-directory /home/andrej/%TERM%",
  "cmd": "xdg-open file://%RESULT%"
}
```

### Dynamic Styling

The window and items will have a class based on the source. Selecting an item will change the windows class to the current selections source. Using a prefix will apply that sources classes to the window.

F.e. search = `!somecommand` => `#window.runner`

## Keybinds

| Key                                     | Description                        |
| --------------------------------------- | ---------------------------------- |
| `Enter`                                 | activate selection                 |
| `Ctrl+Enter`                            | activate selection without closing |
| `Ctrl+j`, `Down`, `Tab`                 | next entry                         |
| `Ctrl+k`, `Up`, `LEFT_TAB` (shift+tab?) | previous entry                     |
| `Escape`                                | close                              |

## Watchout for...

- Desktop entries will be parsed and cached in `.cache/walker`... currently no mechanism to refresh cache, so delete manually if needed
