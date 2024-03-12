# Walker - Application Runner

WIP.

## Features

- Desktop Entries with actions
- Runner (default prefix: `!`)
- Websearch (default prefix: `?`)
- Icons
- notify via `notify-send` on cmd failure
- stay open in background (open via `USR1` signal)

## Requirements

- gtk4
- gtk4-layer-shell

## Installation

Make sure you have [gtk4-layer-shell](https://github.com/wmww/gtk4-layer-shell) installed properly.
Additionally, you need to make sure `/usr/local/lib/` is in your `LD_LIBRARY_PATH`. That's where the gtk4-layer-shell lib is located. `/usr/local/lib/pkgconfig` needs to be in your `PKG_CONFIG_PATH` as well.

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
  "keep_open": false,
  "notify_on_fail": true,
  "fullscreen": true,
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

### Dynamic Styling

The window and items will have a class based on the source. Selecting an item will change the windows class to the current selections source. Using a prefix will apply that sources classes to the window.

F.e. search = `!somecommand` => `#window.runner`

## Keybinds

| Key          | Description                        |
| ------------ | ---------------------------------- |
| `Enter`      | activate selection                 |
| `Ctrl+Enter` | activate selection without closing |
| `Ctrl+j`     | next entry                         |
| `Ctrl+k`     | previous entry                     |
| `Escape`     | close (hide if config.keep_open)   |

## Watchout for...

- Desktop entries will be parsed and cached in `.cache/walker`... currently no mechanism to refresh cache, so delete manually if needed
