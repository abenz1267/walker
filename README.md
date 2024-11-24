# Walker - Application launcher

Walker is a highly extendable application launcher that doesn't hold back on features and usability. Fast. Unclutters your brain. Improves your workflow.

![Screenshot](https://github.com/abenz1267/walker/blob/master/assets/images.png?raw=true)

## Features

- plugin support: simple stdin/stdout (external or via configuration, see wiki)
- icons/images
- start as service for faster startup
- run entries via labels (F<1-8> or jkl;asdf)
- non-blocking async handling of results
- typeahead
- start with explicit modules, style or config
- arrow-up history
- drag&drop support
- dmenu-mode
- run as password input
- theming support (global, per module, with inheritance)

## Builtin Modules

- ai
  - currently only Claude 3.5
  - define different prompts
- runner
  - parses your shell config for aliases
  - exlusive list or all binaries
  - ignore-list
  - generic runner
  - semi-smart: `shu now` => `shutdown now`
- windows
  - simple window switcher
- desktop applications
  - history-aware
  - desktop actions (f.e. `Open a new private window` [Firefox])
  - puts newly installed applications on top
  - context-aware (context = open windows)
- websearch ()
  - simple websearch
  - google, duckduckgo, ecosia, yandex
  - can open websites directly
- clipboard
  - simple clipboard history
  - with images
- module switcher
  - lets you switch to specific modules
- commands (for Walker, f.e. clear cache)
- ssh
  - parses your `known_hosts` and `config` files
- finder
  - simple fuzzy finder
  - drag&drop support
- emojis
- symbols
- calculator
  - uses [libqalculate](https://github.com/Qalculate/libqalculate)
- custom commands (for running simple commands)
  - lets you define and run simple one-off commands
  - f.e. `toggle window floating`
  - no need to create keybinds for commands you don't run often

## Requirements

- gtk4-layer-shell

## Installation

```
arch:
yay -S walker-bin

// or to build from source
yay -S walker
```

### Building from source

**_Building can take quite a while, be patient_**

Make sure you have the following dependencies installed:

- go
- gtk4
- gtk4-layer-shell
- gobject-introspection

```bash
git clone https://github.com/abenz1267/walker /tmp/walker
cd /tmp/walker/cmd
go build -x -o walker // the '-x' is for debug output
sudo cp walker /usr/bin/
```

Without these you won't be able to build.

<details>
<summary>Install using Nix</summary>

You have two options of installing walker using Nix.

1.  Using the package exposed by this flake

    1. Add to your flake `inputs.walker.url = "github:abenz1267/walker";`
    2. Add `inputs.walker.packages.<system>.default` to `environment.systemPackages` or `home.packages`

2.  Using the home-manager module exposed by this flake:

    1. Add to your flake `inputs.walker.url = "github:abenz1267/walker";`
    2. Add `imports = [inputs.walker.homeManagerModules.default];` into your home-manager config
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

Additionally, there is a binary caches at `https://walker.cachix.org` and `https://walker-git.cachix.org` which you can use with the following:

```nix
nix.settings = {
  substituters = ["https://walker.cachix.org"];
  trusted-public-keys = ["walker.cachix.org-1:fG8q+uAaMqhsMxWjwvk0IMb4mFPFLqHjuvfwQxE4oJM="];
};
```

```nix
nix.settings = {
  substituters = ["https://walker-git.cachix.org"];
  trusted-public-keys = ["walker-git.cachix.org-1:vmC0ocfPWh0S/vRAQGtChuiZBTAe4wiKDeyyXM0/7pM="];
};
```

</details>

## Running as a service

This depends on your system. You simply need to autostart Walker with `walker --gapplication-service` and it will run in the background. Then just run `walker` to bring it up.

Example for Hyprland:

```bash
exec-once=walker --gapplication-service
```

## Config & Style

[Check the wiki](https://github.com/abenz1267/walker/wiki)

## Start Walker with explicit modules

You can start walker with explicit modules by using the `--modules` flag. F.e:

```bash
walker --modules applications,ssh
```

Will tell Walker to only use the applications and ssh module.

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

| class                | condition                  |
| -------------------- | -------------------------- |
| `#window.activation` | AM enabled                 |
| `#spinner.visible`   | Processing in progress     |
| `#item.<entryclass>` | Always                     |
| `#item.active`       | Dmenu with '--active'-flag |

### Starting as service

Start with `walker --gapplication-service` to start in service-mode. Calling `walker` normally afterwards should be rather fast.

### Additional flags

| Flag                  | Description                                  |
| --------------------- | -------------------------------------------- |
| `--modules`, `-m`     | Run with explicit modules                    |
| `--new`, `-n`         | Start new instance ignoring service          |
| `--config`, `-c`      | Config file to use                           |
| `--theme`, `-s`       | Theme to use                                 |
| `--dmenu`, `-d`       | Start in dmenu mode                          |
| `--keepsort`, `-k`    | Don't sort alphabetically                    |
| `--placeholder`, `-p` | Placeholder text                             |
| `--labelcolumn`, `-l` | Column to use for the label                  |
| `--password`, `-y`    | Launch in password mode                      |
| `--forceprint`, `-f`  | Forces printing input if no item is selected |
| `--query`, `-q`       | To set initial query                         |

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
| `Ctrl + Label`                                                          | Activate item by label                                                   |
| `Ctrl + c`                                                              | AI: copy last response                                                   |
| `Ctrl + p`                                                              | AI: resume last session for prompt                                       |
| `Ctrl + x`                                                              | AI: clear current session                                                |
| `Ctrl + Shift + Label`                                                  | Activate item by label without closing                                   |
| `Shift+Backspace`                                                       | delete entry from history                                                |

### Activation Mode

Activation-Mode can be triggered by holding `LCtrl` ( or `LAlt`). The window will get an additional class `activation` you can use for styling. While activated, you can run items by pressing their respective label. This only works for the top 8 items.

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
