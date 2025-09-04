use crate::config::get_config;
use crate::providers::PROVIDERS;
use crate::state::add_theme;
use crate::ui::window::{set_css_provider, with_css_provider};
use gtk4::gdk::Display;
use gtk4::prelude::GtkWindowExt;
use gtk4::{CssProvider, Window};
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
    pub preview: String,
    pub css: String,
    pub items: HashMap<String, String>,
}

impl Theme {
    pub fn default() -> Self {
        let mut s = Self {
            layout: include_str!("../../resources/themes/default/layout.xml").to_string(),
            preview: include_str!("../../resources/themes/default/preview.xml").to_string(),
            css: include_str!("../../resources/themes/default/style.css").to_string(),
            items: HashMap::new(),
        };

        for (k, v) in PROVIDERS.get().unwrap() {
            s.items.insert(k.clone(), v.get_item_layout());
        }

        return s;
    }
}

pub fn setup_themes(elephant: bool, theme: String, is_service: bool) {
    let mut themes: HashMap<String, Theme> = HashMap::new();
    let mut path = dirs::config_dir().unwrap();
    path.push("walker");
    path.push("themes");

    let mut paths = vec![path.to_string_lossy().to_string()];
    if let Some(a) = &get_config().additional_theme_location
        && let Ok(home) = env::var("HOME")
    {
        paths.push(a.replace("~", &home).to_string());
    }

    let files = vec![
        "layout.xml".to_string(),
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

    if theme != "default" || is_service {
        for mut path in paths {
            if !is_service {
                path = format!("{path}/{theme}");

                themes.insert(
                    theme.clone(),
                    setup_theme_from_path(path.clone().into(), &combined),
                );
                continue;
            }

            let Ok(entries) = fs::read_dir(path) else {
                continue;
            };

            for entry in entries {
                let entry = entry.unwrap();
                let path = entry.path();

                if !path.is_dir() {
                    continue;
                }

                let Some(name) = path.file_name() else {
                    continue;
                };

                let path_theme = name.to_string_lossy();

                themes.insert(
                    path_theme.to_string(),
                    setup_theme_from_path(path.clone(), &combined),
                );

                add_theme(path_theme.to_string());
            }
        }
    }

    themes.insert("default".to_string(), Theme::default());
    add_theme("default".to_string());

    THEMES.with(|s| s.set(themes).expect("failed initializing themes"));
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
                if let Some(s) = read_file(file)
                    && let Ok(home) = env::var("HOME")
                {
                    theme.css = s.replace("~", &home);
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
            name if name.ends_with(".xml") && name.starts_with("item_") => {
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
            with_css_provider(|p| p.load_from_string(&t.css));
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
