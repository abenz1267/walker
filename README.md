# Walker - A Modern Application Launcher

A fast, customizable application launcher built with GTK4 and Rust, designed for Linux desktop environments. Walker provides a clean, modern interface for launching applications, running commands, performing calculations, and more.

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

Launch Walker with:

```bash
walker --gapplication-service
```

To open it, simply call:

```bash
walker
```

## Theming

You can customize Walker's appearance by creating a custom theme. Checkout `resources/themes/default` for the default theme. Themes inherit the default theme by default, so if you just want to change the CSS, you can just create `themes/yours/style.css`.

You can customize rendering of list items for each provider individually, f.e. "item_files.xml" will define the layout for items sourced from the `files` provider.

Please refer to [the GTK4 docs](https://docs.gtk.org/gtk4/) to checkout how to write `*.xml` files for GTK4.

## License

This project is licensed under the GNU General Public License v3.0 - see the [LICENSE](LICENSE) file for details.

## Development Status

This is a beta version (1.0.0-beta) undergoing active development. Features and APIs may change before the stable 1.0 release.
