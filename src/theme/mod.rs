use crate::config::get_config;
use crate::providers::PROVIDERS;
use crate::state::add_theme;
use crate::ui::window::{set_css_provider, with_css_provider};
use gtk4::gdk::Display;
use gtk4::prelude::GtkWindowExt;
use gtk4::{CssProvider, Window, gio};
use gtk4_layer_shell::{Edge, KeyboardMode, Layer, LayerShell};
use std::cell::OnceCell;
use std::collections::HashMap;
use std::path::PathBuf;
use std::{env, fs};

thread_local! {
    pub static THEMES: OnceCell<HashMap<String, Theme>> = OnceCell::new();
}

#[derive(Debug)]
pub struct Theme {
    pub layout: String,
    pub keybind: String,
    pub preview: String,
    pub scss: Option<PathBuf>,
    pub css: Option<gio::File>,
    pub items: HashMap<String, String>,
    pub grid_items: HashMap<String, String>,
}

impl Theme {
    pub fn default() -> Self {
        let mut s = Self {
            layout: include_str!("../../resources/themes/default/layout.xml").to_string(),
            keybind: include_str!("../../resources/themes/default/keybind.xml").to_string(),
            preview: include_str!("../../resources/themes/default/preview.xml").to_string(),
            scss: None,
            css: None,
            items: HashMap::new(),
            grid_items: HashMap::new(),
        };

        for (k, v) in PROVIDERS.get().unwrap() {
            s.items.insert(k.clone(), v.get_item_layout());
        }

        for (k, v) in PROVIDERS.get().unwrap() {
            s.grid_items.insert(k.clone(), v.get_item_grid_layout());
        }

        s
    }
}

pub fn setup_themes(elephant: bool, theme: String, is_service: bool) {
    let mut themes: HashMap<String, Theme> = HashMap::new();

    let dirs = xdg::BaseDirectories::with_prefix("walker").find_config_files("themes");

    let mut config_paths: Vec<PathBuf> = dirs.collect();

    if let Some(a) = &get_config().additional_theme_location
        && let Ok(home) = env::var("HOME")
    {
        config_paths.push(PathBuf::from(a.replace("~", &home).to_string()));
    }

    let files = vec![
        "layout.xml".to_string(),
        "keybind.xml".to_string(),
        "style.scss".to_string(),
        "style.css".to_string(),
        "preview.xml".to_string(),
    ];

    let combined = if elephant {
        let mut result = files;
        let additional = PROVIDERS
            .get()
            .unwrap()
            .iter()
            .map(|v| format!("item_{}.xml", v.0));
        result.extend(additional);
        result
    } else {
        files
    };

    for config_path in config_paths {
        if !is_service {
            let mut path = config_path;

            path.push(&theme);

            if let Some(t) = setup_theme_from_path(path.clone(), &combined) {
                themes.insert(theme.clone(), t);
                add_theme(theme.clone());
            }

            path.pop();

            continue;
        }

        let Ok(theme_dirs) = fs::read_dir(config_path) else {
            continue;
        };

        for theme_dir in theme_dirs {
            let entry = theme_dir.unwrap();
            let path = entry.path();

            if !path.is_dir() {
                continue;
            }

            let Some(name) = path.file_name() else {
                continue;
            };

            let theme_name = name.to_string_lossy();

            if let Some(t) = setup_theme_from_path(path.clone(), &combined) {
                themes.insert(theme_name.to_string(), t);
                add_theme(theme_name.to_string());
            }
        }
    }

    if !themes.contains_key("default") {
        themes.insert("default".to_string(), Theme::default());
        add_theme("default".to_string());
    }

    THEMES.with(|s| s.set(themes).expect("failed initializing themes"));
}

fn setup_theme_from_path(path: PathBuf, files: &Vec<String>) -> Option<Theme> {
    let mut path = path;

    if !path.exists() {
        return None;
    }

    let mut theme = Theme::default();

    let mut pc = path.clone();

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
            "style.scss" => {
                pc.push("style.scss");
                theme.scss = Some(pc.clone());
                pc.pop();
            }
            "style.css" => {
                pc.push("style.css");
                theme.css = Some(gio::File::for_path(&pc));
                pc.pop();
            }
            "layout.xml" => {
                if let Some(s) = read_file(file) {
                    theme.layout = s;
                }
            }
            "keybind.xml" => {
                if let Some(s) = read_file(file) {
                    theme.keybind = s;
                }
            }
            "preview.xml" => {
                if let Some(s) = read_file(file) {
                    theme.preview = s;
                }
            }
            name if name.ends_with("_grid.xml") && name.starts_with("item_") => {
                if let Some(s) = read_file(file) {
                    let key = name
                        .strip_prefix("item_")
                        .unwrap()
                        .strip_suffix(".xml")
                        .unwrap();
                    theme.grid_items.insert(key.to_string(), s);
                }
            }
            name if name.ends_with(".xml")
                && name.starts_with("item_")
                && !name.ends_with("grid.xml") =>
            {
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

    Some(theme)
}

pub fn setup_css(theme: String) {
    with_themes(|t| {
        if let Some(t) = t.get(&theme) {
            with_css_provider(|p| {
                if let Some(f) = &t.scss {
                    if let Ok(scss) = fs::read_to_string(f) {
                        let options = match f.parent() {
                            Some(dir) => grass::Options::default().load_path(dir),
                            None => grass::Options::default(),
                        };
                        match grass::from_string(scss, &options) {
                            Ok(css) => {
                                p.load_from_string(&css);
                                return;
                            }
                            Err(err) => {
                                eprintln!("SCSS parse error: {err}");
                                return;
                            }
                        }
                    }
                };
                if let Some(f) = &t.css {
                    p.load_from_file(f);
                    return;
                } else {
                    p.load_from_string(include_str!("../../resources/themes/default/style.css"));
                }
            });
        }
    });
}

pub fn setup_css_provider() {
    let display = Display::default().unwrap();
    let p = CssProvider::new();

    gtk4::style_context_add_provider_for_display(&display, &p, gtk4::STYLE_PROVIDER_PRIORITY_USER);
    set_css_provider(p);
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
    win.set_keyboard_mode(if cfg.force_keyboard_focus {
        KeyboardMode::Exclusive
    } else {
        KeyboardMode::OnDemand
    });
    win.set_anchor(Edge::Left, cfg.shell.anchor_left);
    win.set_anchor(Edge::Right, cfg.shell.anchor_right);
    win.set_anchor(Edge::Top, cfg.shell.anchor_top);
    win.set_anchor(Edge::Bottom, cfg.shell.anchor_bottom);
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
