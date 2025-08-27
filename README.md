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
- **Arch Linux Packages**: Search through available packages (official and aur), install or delete a target! List all exlusively installed packages.

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
