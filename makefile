PREFIX ?= /usr/local
DESTDIR ?=
BINDIR = $(DESTDIR)$(PREFIX)/bin
LICENSEDIR = $(DESTDIR)$(PREFIX)/share/licenses/walker
CONFIGDIR = $(DESTDIR)/etc/xdg/walker
THEMEDIR = $(CONFIGDIR)/themes/default

CARGO_TARGET_DIR ?= target
RUSTUP_TOOLCHAIN ?= stable

.PHONY: all build install uninstall clean

all: build

build:
	export RUSTUP_TOOLCHAIN=$(RUSTUP_TOOLCHAIN) && \
	export CARGO_TARGET_DIR=$(CARGO_TARGET_DIR) && \
	cargo build --release

install: build
	install -Dm 755 $(CARGO_TARGET_DIR)/release/walker $(BINDIR)/walker
	install -Dm 644 LICENSE $(LICENSEDIR)/LICENSE
	install -Dm 644 resources/config.toml $(CONFIGDIR)/config.toml
	install -Dm 644 resources/themes/default/item.xml $(THEMEDIR)/item.xml
	install -Dm 644 resources/themes/default/item_calc.xml $(THEMEDIR)/item_calc.xml
	install -Dm 644 resources/themes/default/item_clipboard.xml $(THEMEDIR)/item_clipboard.xml
	install -Dm 644 resources/themes/default/item_dmenu.xml $(THEMEDIR)/item_dmenu.xml
	install -Dm 644 resources/themes/default/item_files.xml $(THEMEDIR)/item_files.xml
	install -Dm 644 resources/themes/default/item_providerlist.xml $(THEMEDIR)/item_providerlist.xml
	install -Dm 644 resources/themes/default/item_symbols.xml $(THEMEDIR)/item_symbols.xml
	install -Dm 644 resources/themes/default/layout.xml $(THEMEDIR)/layout.xml
	install -Dm 644 resources/themes/default/preview.xml $(THEMEDIR)/preview.xml
	install -Dm 644 resources/themes/default/style.css $(THEMEDIR)/style.css

uninstall:
	rm -f $(BINDIR)/walker
	rm -rf $(LICENSEDIR)
	rm -rf $(CONFIGDIR)

clean:
	cargo clean

dev-install: PREFIX = /usr/local
dev-install: install

help:
	@echo "Available targets:"
	@echo "  all       - Build the application (default)"
	@echo "  build     - Build the application"
	@echo "  install   - Install the application and resources"
	@echo "  uninstall - Remove installed files"
	@echo "  clean     - Clean build artifacts"
	@echo "  help      - Show this help"
	@echo ""
	@echo "Variables:"
	@echo "  PREFIX    - Installation prefix (default: /usr/local)"
	@echo "  DESTDIR   - Destination directory for staged installs"
