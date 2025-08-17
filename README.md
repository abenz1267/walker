# Walker - A Modern Application Launcher

A fast, customizable application launcher built with GTK4 and Rust, designed for Linux desktop environments. Walker provides a clean, modern interface for launching applications, running commands, performing calculations, and more.

## Features

Walker offers multiple provider types for different use cases:

- **Desktop Applications**: Launch installed GUI applications
- **Calculator**: Perform mathematical calculations with `=` prefix
- **File Browser**: Navigate and open files with `/` prefix
- **Command Runner**: Execute shell commands
- **Clipboard History**: Access clipboard history with `:` prefix
- **Symbol Picker**: Insert special symbols with `.` prefix
- **Provider List**: Switch between providers with `;` prefix
- **Menu Integration**: System menu integration support

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

### Keyboard Shortcuts

| Key          | Action                 |
| ------------ | ---------------------- |
| `Escape`     | Close Walker           |
| `Down Arrow` | Select next item       |
| `Up Arrow`   | Select previous item   |
| `Ctrl+E`     | Toggle exact search    |
| `Enter`      | Activate selected item |

### Provider Prefixes

| Prefix | Provider      | Description                  |
| ------ | ------------- | ---------------------------- |
| `=`    | Calculator    | Perform calculations         |
| `/`    | Files         | Browse files and directories |
| `:`    | Clipboard     | Access clipboard history     |
| `.`    | Symbols       | Insert special symbols       |
| `;`    | Provider List | Switch between providers     |

### Search Modes

- **Normal Search**: Type to search across all enabled providers
- **Exact Search**: Prefix with `'` or use `Ctrl+E` for exact matches
- **Provider-Specific**: Use prefixes to search within specific providers

## Configuration

### Caution

Walker currently hard-codes the configuration, this is will change. Below are the current values.

#### Application Settings

- **Application ID**: `dev.benz.walker`
- **Version**: `1.0.0-beta`
- **Window Title**: `Walker`
- **Layer Shell Namespace**: `walker`

#### Socket Communication

- **Socket Path**: `{temp_dir}/elephant.sock`
- **Max Results**: `50` items per search

#### UI Layout

- **Window Dimensions**:
  - Default width: `400px`
  - Default height: `400px`
  - Top margin: `300px`
- **Placeholder Text**: `"No Results"`

#### Default Keybindings

- **Close**: `escape`
- **Next Item**: `Down`
- **Previous Item**: `Up`
- **Toggle Exact Search**: `ctrl e`
- **Keep Open Modifier**: `shift`

#### Provider-Specific Actions

- **Files**:
  - Open: `enter`
  - Open Directory: `ctrl enter`
  - Copy Path: `ctrl shift C`
  - Copy File: `ctrl c`
- **Calculator**:
  - Copy Result: `enter`
  - Save: `ctrl s`
  - Delete: `ctrl d`
- **Clipboard**:
  - Copy: `enter`
  - Delete: `ctrl d`
- **Runner**:
  - Start: `enter`
  - Start in Terminal: `ctrl enter`

#### Search Configuration

- **Global Argument Delimiter**: `#`
- **Exact Search Prefix**: `'`
- **Clipboard Time Format**: `dd.MM. - hh:mm`

#### Default Providers

- **On Empty Search**: Desktop Applications
- **On Text Search**: Desktop Applications, Calculator, Runner, Menus

#### Window Positioning

- **Anchors**: All edges (top, bottom, left, right) enabled by default
- **Layer**: Overlay
- **Keyboard Mode**: On Demand
- **Exclusive Zone**: `-1`

## Architecture

Walker uses a client-server architecture:

- **Frontend**: GTK4-based GUI written in Rust
- **Backend**: Service that handles provider queries via Unix sockets
- **Protocol**: Custom protocol buffer-based communication
- **Providers**: Modular system for different content types

## License

This project is licensed under the GNU General Public License v3.0 - see the [LICENSE](LICENSE) file for details.

## Development Status

This is a beta version (1.0.0-beta) undergoing active development. Features and APIs may change before the stable 1.0 release.
