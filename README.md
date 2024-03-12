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
