# Walker - A Modern Application Launcher

A fast, customizable application launcher built with GTK4 and Rust, designed for Linux desktop environments. Walker provides a clean, modern interface for launching applications, running commands, performing calculations, and more.

[![Discord](https://img.shields.io/discord/1402235361463242964?logo=discord)](https://discord.gg/mGQWBQHASt)
[![License: GPL v3](https://img.shields.io/badge/License-GPLv3-blue.svg)](https://www.gnu.org/licenses/gpl-3.0)

![screenshot](https://raw.githubusercontent.com/abenz1267/walker/refs/heads/master/resources/screenshot.png)

## Features

The following Elephant providers are implemented by default:

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

### Dependencies

- GTK4 (version 4.6+)
- gtk4-layer-shell
- Protocol Buffers compiler
- cairo
- poppler-glib
- make sure [elephant](https://github.com/abenz1267/elephant) is running before starting Walker

<details>
    <summary> <h3> Install using Nix </h3> </summary>

#### 1. Add flake inputs

Add walker and elephant to the inputs of your configs `flake.nix` and set walker to follow elephant

```nix
elephant.url = "github:abenz1267/elephant";

walker = {
  url = "github:abenz1267/walker";
  inputs.elephant.follows = "elephant";
};
```

#### 2. Install walker

You have 3 options for installing walker.

**Option A** (Home Manager Module): Import the home-manager module to your home-manager config and enable walker.

```nix
imports = [inputs.walker.homeManagerModules.default];

programs.walker.enable = true;
```

**Option B** (NixOS Module): Import the nixos module in your NixOS config and enable walker

```nix
imports = [inputs.walker.homeManagerModules.default];

programs.walker.enable = true;
```

**Option C** (Package): Add `inputs.walker.packages.<system>.default` to your system packages or home-manager packages. replace `<system>` with your system architecture

```nix
home.packages = [inputs.walker.packages.<system>.default];
```

```nix
environment.systemPackages = [inputs.walker.packages.<system>.default];

```

#### 3. Configure walker:

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

Optionally, there is 2 binary caches which can be used by adding the following to you config:

```nix
nix.settings = {
  extra-substituters = ["https://walker.cachix.org" "https://walker-git.cachix.org"];
  extra-trusted-public-keys = ["walker.cachix.org-1:fG8q+uAaMqhsMxWjwvk0IMb4mFPFLqHjuvfwQxE4oJM=" "walker-git.cachix.org-1:vmC0ocfPWh0S/vRAQGtChuiZBTAe4wiKDeyyXM0/7pM="];
};
```

</details>

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

Check out the [default config](https://raw.githubusercontent.com/abenz1267/walker/refs/heads/master/resources/config.toml).

## Theming

You can customize Walker's appearance by creating a custom theme. Checkout `resources/themes/default` for the default theme. Themes inherit the default theme by default, so if you just want to change the CSS, you can just create `themes/yours/style.css`.

You can customize rendering of list items for each provider individually, f.e. "item_files.xml" will define the layout for items sourced from the `files` provider.

Please refer to [the GTK4 docs](https://docs.gtk.org/gtk4/) to checkout how to write `*.xml` files for GTK4.

**THE DEFAULT THEME CANNOT BE CHANGED**.

## Contributing

Please do not make PRs to fix single typos. Fix all or nothing.

## License

This project is licensed under the GNU General Public License v3.0 - see the [LICENSE](LICENSE) file for details.
