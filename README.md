# Walker - A Modern Application Launcher

A fast, customizable application launcher built with GTK4 and Rust, designed for Linux desktop environments. Walker provides a clean, modern interface for launching applications, running commands, performing calculations, and more.

[![Discord](https://img.shields.io/discord/1402235361463242964?logo=discord)](https://discord.gg/mGQWBQHASt)
[![License: GPL v3](https://img.shields.io/badge/License-GPLv3-blue.svg)](https://www.gnu.org/licenses/gpl-3.0)

## Features

Walker offers multiple provider types for different use cases:

- **Desktop Applications**: Launch installed GUI applications
- **Calculator**: Perform mathematical calculations with `=` prefix
- **File Browser**: Navigate and open files with `/` prefix
- **Command Runner**: Execute shell commands
- **Websearch**: Search the web with custom-defined engines
- **Clipboard History**: Access clipboard history with `:` prefix
- **Symbol Picker**: Insert special symbols with `.` prefix
- **Provider List**: Switch between providers with `;` prefix
- **Menu Integration**: Create custom menus with elephant and let walker display them
- **Dmenu**: Your good old dmenu ... with seamless menus!
- **Arch Linux Packages**: Search through available packages (official and aur), install or delete a target! List all exlusively installed packages.
- **Todo List**: create simple todo items with basic time tracking, scheduling and notifications
- **Bluetooth**: basic bluetooth management

## Installation

### Build from Source

```bash
# Clone the repository
git clone https://github.com/abenz1267/walker.git
cd walker

# Build with Cargo
cargo build --release

# Run Walker
./target/release/walker
```

### Install using Nix

You have two options of installing walker using Nix.

1. Using the package exposed by this flake
   1. Add to your flake `inputs.walker.url = "github:abenz1267/walker";`
   2. Add `inputs.walker.packages.<system>.default` to `environment.systemPackages` or `home.packages`

2. Using the home-manager module exposed by this flake:
   1. Add to your flake `inputs.walker.url = "github:abenz1267/walker";`
   2. Add `imports = [inputs.walker.homeManagerModules.default];` into your home-manager config
   3. Configure walker using:

```nix
programs.walker = {
  enable = true;
  runAsService = true;

  # All options from the config.toml can be used here.
  config = {
    placeholders."default".input = "Example";
    providers.prefixes = [
      {provider = "websearch"; prefix = "+";}
      {provider = "providerlist"; prefix = "_";}
    ];
    keybinds.quick_activate = ["F1" "F2" "F3"];
  };

  # If this is not set the default styling is used.
  theme.style = ''
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

### Dependencies

- GTK4 (version 4.6+)
- gtk4-layer-shell
- Protocol Buffers compiler
- cairo
- poppler-glib
- make sure [elephant](https://github.com/abenz1267/elephant) is running before starting Walker

## Usage

### Basic Usage

**Make sure `elephant` is running and you have providers installed. `elephant-providerlist` and f.e. `elephant-desktopapplications`.**

Launch Walker with `walker`.

In order to improve startup performance, run a Walker service with:

```bash
walker --gapplication-service
```

If the service is running, you can either open Walker with:

```bash
walker

```

or for an even faster launch make a socket call, f.e. with `openbsd-netcat`:

```bash
nc -U /run/user/1000/walker/walker.sock
```

The downside of the socket call is that it does not handle any commandline options, so it's just a faster alternative to a simple `walker` call.

## Keybinds

The following modifier keys are valid: `ctrl`, `alt`, `shift`, `super`.

To get a full list of possible key values, look here: [GDK key-values](https://github.com/gtk-rs/gtk4-rs/blob/0.9/gdk4/sys/src/lib.rs#L302).

F.e. `pub const GDK_KEY_semicolon: c_int = 59;` means that `ctrl semicolon` would be a valid keybind.

## Config

Configuration should be done in `~/.config/walker`.

Check out the [default config](https://raw.githubusercontent.com/abenz1267/walker/refs/heads/1.0.0/resources/config.toml).

## Theming

You can customize Walker's appearance by creating a custom theme. Checkout `resources/themes/default` for the default theme. Themes inherit the default theme by default, so if you just want to change the CSS, you can just create `themes/yours/style.css`.

You can customize rendering of list items for each provider individually, f.e. "item_files.xml" will define the layout for items sourced from the `files` provider.

Please refer to [the GTK4 docs](https://docs.gtk.org/gtk4/) to checkout how to write `*.xml` files for GTK4.

**THE DEFAULT THEME CANNOT BE CHANGED**.

## License

This project is licensed under the GNU General Public License v3.0 - see the [LICENSE](LICENSE) file for details.
