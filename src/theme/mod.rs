use crate::config::{DEFAULT_STYLE, get_config, get_user_theme_path};
use crate::ui::window::with_window;
use gtk4::gdk::Display;
use gtk4::prelude::GtkWindowExt;
use gtk4::{CssProvider, Window};
use gtk4_layer_shell::{Edge, KeyboardMode, Layer, LayerShell};
use notify::{Config, Event, RecommendedWatcher, RecursiveMode, Watcher};
use std::collections::HashMap;
use std::path::PathBuf;
use std::process::Command;
use std::sync::{OnceLock, mpsc};
use std::{fs, thread};

thread_local! {
    pub static THEMES: OnceLock<HashMap<String, Theme>> = OnceLock::new();
}

#[derive(Debug)]
pub struct Theme {
    pub layout: String,
    pub preview: String,
    pub css: String,
    pub items: HashMap<String, String>,
}

impl Theme {
    pub fn default() -> Self {
        Self {
            layout: include_str!("../../resources/themes/default/layout.xml").to_string(),
            preview: include_str!("../../resources/themes/default/preview.xml").to_string(),
            css: include_str!("../../resources/themes/default/style.css").to_string(),
            items: HashMap::from([
                (
                    "default".to_string(),
                    include_str!("../../resources/themes/default/item.xml").to_string(),
                ),
                (
                    "clipboard".to_string(),
                    include_str!("../../resources/themes/default/item_clipboard.xml").to_string(),
                ),
                (
                    "symbols".to_string(),
                    include_str!("../../resources/themes/default/item_symbols.xml").to_string(),
                ),
                (
                    "calc".to_string(),
                    include_str!("../../resources/themes/default/item_calc.xml").to_string(),
                ),
                (
                    "files".to_string(),
                    include_str!("../../resources/themes/default/item_files.xml").to_string(),
                ),
                (
                    "providerlist".to_string(),
                    include_str!("../../resources/themes/default/item_providerlist.xml")
                        .to_string(),
                ),
            ]),
        }
    }
}

pub fn setup_themes() {
    let mut themes: HashMap<String, Theme> = HashMap::new();
    let mut path = dirs::config_dir().unwrap();
    path.push("walker");
    path.push("themes");

    let mut paths = vec![path.to_string_lossy().to_string()];
    if let Some(a) = &get_config().additional_theme_location {
        paths.push(a.to_string());
    }

    let output = Command::new("elephant")
        .arg("listproviders")
        .output()
        .unwrap();

    let stdout = String::from_utf8(output.stdout).unwrap();

    let providers: Vec<String> = stdout
        .lines()
        .filter_map(|line| {
            line.split_once(':')
                .map(|(_, value)| format!("item_{}.xml", value.to_string()))
        })
        .collect();

    let files = vec![
        "item.xml".to_string(),
        "layout.xml".to_string(),
        "style.css".to_string(),
        "preview.xml".to_string(),
    ];

    let combined = [files, providers].concat();

    for path in paths {
        if let Ok(entries) = fs::read_dir(path) {
            for entry in entries {
                let entry = entry.unwrap();
                let path = entry.path();

                if path.is_dir() {
                    if let Some(name) = path.file_name() {
                        let theme = name.to_string_lossy();
                        themes.insert(
                            theme.to_string(),
                            setup_theme_from_path(path.clone(), &combined),
                        );
                    }
                }
            }
        }
    }

    themes.insert("default".to_string(), Theme::default());

    THEMES.with(|s| {
        s.set(themes).expect("failed initializing themes");
    });
}

fn setup_theme_from_path(mut path: PathBuf, files: &Vec<String>) -> Theme {
    let mut theme = Theme::default();

    let mut read_file = |filename: &str| -> Option<String> {
        path.push(filename);
        let result = fs::read_to_string(&path).ok();
        path.pop();
        result
    };

    for file in files {
        match file.as_str() {
            "item.xml" => {
                if let Some(s) = read_file(file) {
                    theme.items.insert("default".to_string(), s);
                }
            }
            "style.css" => {
                if let Some(s) = read_file(file) {
                    theme.css = s;
                }
            }
            "layout.xml" => {
                if let Some(s) = read_file(file) {
                    theme.layout = s;
                }
            }
            "preview.xml" => {
                if let Some(s) = read_file(file) {
                    theme.preview = s;
                }
            }
            name if name.starts_with(".xml") && name.starts_with("item_") => {
                if let Some(s) = read_file(file) {
                    let key = name
                        .strip_prefix("item_")
                        .unwrap()
                        .strip_suffix(".xml")
                        .unwrap();
                    theme.items.insert(key.to_string(), s);
                }
            }
            _ => (),
        }
    }

    return theme;
}

pub fn setup_css(theme: String) {
    with_themes(|t| {
        if let Some(t) = t.get(&theme) {
            with_window(|w| {
                w.css_provider.load_from_string(&t.css);
            });
        }
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

    let cfg = get_config();

    win.init_layer_shell();
    win.set_namespace(Some("walker"));
    win.set_exclusive_zone(-1);
    win.set_layer(Layer::Overlay);
    win.set_keyboard_mode(KeyboardMode::OnDemand);

    win.set_anchor(Edge::Left, cfg.shell.anchor_left);
    win.set_anchor(Edge::Right, cfg.shell.anchor_right);
    win.set_anchor(Edge::Top, cfg.shell.anchor_top);
    win.set_anchor(Edge::Bottom, cfg.shell.anchor_top);
}

pub fn with_themes<F, R>(f: F) -> R
where
    F: FnOnce(&HashMap<String, Theme>) -> R,
{
    THEMES.with(|state| {
        let data = state.get().expect("Themes not initialized");
        f(data)
    })
}
