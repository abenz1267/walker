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
- typeahead
- history-aware
- labels: F<1-8> or jkl;asdf
- start with explicit modules, style or config
- arrow-up history
- drag&drop support
- dmenu-mode
- run as password input

## Builtin Modules

- runner
- desktop applications
- websearch (google, duckduckgo, ecosia, yandex)
- hyprland windows, context-aware history (based on open windows)
- clipboard with fuzzy find and images (currently "wl-clipboard" only)
- module switcher
- commands (for Walker, f.e. clear cache)
- ssh
- finder
- emojis
- custom commands (for running simple commands)

## Requirements

- gtk4-layer-shell

## Installation

**_Building can take quite a while, be patient_**

```
arch:
yay -S walker
```

### Building from source

Make sure you have the following dependencies installed:

- go
- gtk4
- gtk4-layer-shell
- gobject-introspection

Without these you won't be able to build.

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
    search.placeholder = "Example";
    ui.fullscreen = true;
    list = {
      height = 200;
    };
    websearch.prefix = "?";
    switcher.prefix = "/";
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

[Check the wiki](https://github.com/abenz1267/walker/wiki)

The config can be written json, toml or yaml. Default values will be used, so you only have to overwrite.

Default config will be put into `$HOME/.config/walker/`.

See `config/config.default.json` and `ui/style.default.css`. Styling is done via GTK CSS.

## Usage SSH Module

In the searchbar type in: `<host> <username>`. Select the host you want. Enter.

## Start Walker with explicit modules

You can start walker with explicit modules by using the `--modules` flag. F.e:

```bash
walker --modules applications,ssh
```

Will tell Walker to only use the applications and ssh module.

## Special Labels

Modules can define a `special_label` which is used for AM. It only really makes sense, if a module returns one entry. This could be used to f.e. always have the websearch result have the same label, so you can activate it with the same label every time, even if you don't see it.

### Custom Special Labels

Format for custom special labels is: `"<entry label>;<entry sub>": "<special label>"`.

Example:

```json
{
  "special_labels": {
    "discord;internet messenger": "1"
  }
}
```

## Styling with typeahead enabled

If you have typeahead enabled, make sure that your `#search` has no background, so the typeahead is readable.

## Providing your own modules

If you want to extend walker with your own modules, you can do that in the config.

```json
{
  "plugins": [
    {
      "prefix": "!",
      "name": "mymodule",
      "src": "node /path/to/myscript.js"
    }
  ]
}
```

See the wiki for more information.

### Dynamic Styling

The window and items will have a class based on the source. Selecting an item will change the windows class to the current selections source. Using a prefix will apply that sources classes to the window.

F.e. search = `!somecommand` => `#window.runner`

| class                | condition              |
| -------------------- | ---------------------- |
| `#window.activation` | AM enabled             |
| `#spinner.visible`   | Processing in progress |
| `#item.<entryclass>` | Always                 |

### Starting as service

Start with `walker --gapplication-service` to start in service-mode. Calling `walker` normally afterwards should be rather fast.

### Additional flags

| Flag                  | Description                                  |
| --------------------- | -------------------------------------------- |
| `--modules`, `-m`     | Run with explicit modules                    |
| `--new`, `-n`         | Start new instance ignoring service          |
| `--config`, `-c`      | Config file to use                           |
| `--style`, `-s`       | Style file to use                            |
| `--dmenu`, `-d`       | Start in dmenu mode                          |
| `--keepsort`, `-k`    | Don't sort alphabetically                    |
| `--placeholder`, `-p` | Placeholder text                             |
| `--labelcolumn`, `-l` | Column to use for the label                  |
| `--password`, `-y`    | Launch in password mode                      |
| `--forceprint`, `-f`  | Forces printing input if no item is selected |

## Keybinds

AM = Activation Mode

| Key                                                                     | Description                                                              |
| ----------------------------------------------------------------------- | ------------------------------------------------------------------------ |
| `Enter`                                                                 | activate selection                                                       |
| `Alt+Enter`                                                             | activate selection with alternative command. By default: run in terminal |
| `Shift+Enter`                                                           | activate selection without closing                                       |
| `Ctrl+j` (if ActivationMode is disabled), `Down`, `Tab`                 | next entry                                                               |
| `Ctrl+k` (if ActivationMode is disabled), `Up`, `LEFT_TAB` (shift+tab?) | previous entry                                                           |
| `Escape`                                                                | close                                                                    |
| `Ctrl`                                                                  | start AM                                                                 |
| in AM: `<label>`                                                        | activate item                                                            |
| in AM: Hold `Shift+<label>`                                             | activate item (don't close)                                              |
| in AM: `Escape`                                                         | stop AM                                                                  |

### Activation Mode

Activation-Mode can be triggered by holding `LCtrl`. The window will get an additional class `activation` you can use for styling. While activated, you can run items by pressing their respective label. This only works for the top 8 items.

## Startup "Benchmarks"

System: Arch Linux, Hyprland, Amd 7950x, 32gb DDR5-6000, SSD

Measured time is until telling GTK to show the window.

| Mode         | Startup time                                           |
| ------------ | ------------------------------------------------------ |
| normal       | 23ms - 33ms                                            |
| with service | ~2.2ms (~3.5 when input needs to be reset, so always?) |

## FAQ

### Newly installed or removed applications aren't shown / are still shown

Make sure to clean the applications cache by either running the "Clear Applications Cache" command from within Walker (using the `commands` module) or by deleting the `applications.json` file in `$HOME/.cache/walker/`.

Additionally you can diasble the cache completely by setting

```json
  "applications": {
    "cache": false
  },
```

in your config.
