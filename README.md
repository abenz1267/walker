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
