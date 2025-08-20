use crate::config::{DEFAULT_STYLE, get_config};
use crate::ui::window::with_window;
use gtk4::gdk::Display;
use gtk4::prelude::GtkWindowExt;
use gtk4::{CssProvider, Window};
use gtk4_layer_shell::{Edge, KeyboardMode, Layer, LayerShell};
use notify::{Config, Event, RecommendedWatcher, RecursiveMode, Watcher};
use std::sync::mpsc;
use std::{fs, thread};

pub fn setup_css(theme: String) {
    let css: String;

    if theme == "default" {
        css = DEFAULT_STYLE.to_string();
    } else {
        let mut path = dirs::config_dir()
            .ok_or("Could not find config directory")
            .unwrap();

        path.push("walker");
        path.push("themes");
        path.push(theme);
        path.push("style.css");

        css = fs::read_to_string(&path).unwrap();
    }

    with_window(|w| {
        w.css_provider.load_from_string(&css);
    });
}

pub fn setup_css_provider() -> CssProvider {
    let css_provider = CssProvider::new();

    gtk4::style_context_add_provider_for_display(
        &Display::default().expect("Could not connect to a display."),
        &css_provider,
        gtk4::STYLE_PROVIDER_PRIORITY_USER,
    );

    return css_provider;
}

pub fn start_theme_watcher(theme_name: String) {
    let mut path = dirs::config_dir()
        .ok_or("Could not find config directory")
        .unwrap();

    path.push("walker");
    path.push("themes");
    path.push(&theme_name);
    path.push("style.css");

    thread::spawn(move || {
        let (tx, rx) = mpsc::channel();

        let mut watcher = RecommendedWatcher::new(
            move |result: Result<Event, notify::Error>| {
                if let Err(_) = tx.send(result) {
                    return;
                }
            },
            Config::default(),
        )
        .expect("Failed to create watcher");

        if let Err(_) = watcher.watch(&path, RecursiveMode::NonRecursive) {
            return;
        }

        let theme_name_for_callback = theme_name.clone();

        for result in rx {
            match result {
                Ok(_event) => {
                    let theme_name_clone = theme_name_for_callback.clone();
                    gtk4::glib::idle_add_once(move || {
                        setup_css(theme_name_clone);
                    });
                }
                Err(error) => println!("Watch error: {:?}", error),
            }
        }
    });
}

pub fn setup_layer_shell(win: &Window) {
    if !gtk4_layer_shell::is_supported() {
        let titlebar = gtk4::Box::new(gtk4::Orientation::Vertical, 0);
        win.set_titlebar(Some(&titlebar));
        return;
    }

    let _cfg = get_config();
    win.init_layer_shell();
    win.set_namespace(Some("walker"));
    win.set_exclusive_zone(-1);
    win.set_layer(Layer::Overlay);
    win.set_keyboard_mode(KeyboardMode::OnDemand);

    win.set_anchor(Edge::Left, true);
    win.set_anchor(Edge::Right, true);
    win.set_anchor(Edge::Top, true);
    win.set_anchor(Edge::Bottom, true);
}
