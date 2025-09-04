use crate::config::get_config;
use crate::providers::PROVIDERS;
use crate::state::add_theme;
use crate::ui::window::{set_css_provider, with_css_provider};
use gtk4::gdk::Display;
use gtk4::prelude::GtkWindowExt;
use gtk4::{CssProvider, Window};
use gtk4_layer_shell::{Edge, KeyboardMode, Layer, LayerShell};
use std::borrow::Cow;
use std::cell::OnceCell;
use std::collections::HashMap;
use std::path::PathBuf;
use std::{env, fs};

thread_local! {
    pub static THEMES: OnceCell<HashMap<String, Theme>> = OnceCell::new();
}

#[derive(Debug)]
pub struct Theme {
    pub layout: Cow<'static, str>,
    pub preview: Cow<'static, str>,
    pub css: Cow<'static, str>,
    pub items: HashMap<String, Cow<'static, str>>,
}

impl Theme {
    pub fn default() -> Self {
        let mut s = Self {
            layout: Cow::Borrowed(include_str!("../../resources/themes/default/layout.xml")),
            preview: Cow::Borrowed(include_str!("../../resources/themes/default/preview.xml")),
            css: Cow::Borrowed(include_str!("../../resources/themes/default/style.css")),
            items: HashMap::new(),
        };

        for (k, v) in PROVIDERS.get().unwrap() {
            s.items
                .insert(k.clone(), Cow::Borrowed(v.get_default_item_layout()));
        }

        return s;
    }
}

pub fn setup_themes(elephant: bool, theme: String, is_service: bool) {
    let mut themes: HashMap<String, Theme> = HashMap::new();
    let mut path = dirs::config_dir().unwrap();
    path.push("walker");
    path.push("themes");

    let mut paths = vec![path.to_string_lossy()];
    if let Some(a) = &get_config().additional_theme_location
        && let Ok(home) = env::var("HOME")
    {
        paths.push(Cow::Owned(a.replace("~", &home)));
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
        for path in paths {
            if !is_service {
                let path = format!("{path}/{theme}");
                themes.insert(theme.clone(), setup_theme_from_path(path.into(), &combined));
                continue;
            }

            let Ok(entries) = fs::read_dir(path.as_ref()) else {
                continue;
            };

            for entry in entries.filter_map(Result::ok) {
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
        match (file.as_str(), read_file(file)) {
            ("item.xml", Some(s)) => {
                theme.items.insert("default".to_string(), Cow::Owned(s));
            }
            ("style.css", Some(s)) => {
                if let Ok(home) = env::var("HOME") {
                    theme.css = Cow::Owned(s.replace("~", &home));
                }
            }
            ("layout.xml", Some(s)) => {
                theme.layout = Cow::Owned(s);
            }
            ("preview.xml", Some(s)) => {
                theme.preview = Cow::Owned(s);
            }
            (name, Some(s)) if name.ends_with(".xml") && name.starts_with("item_") => {
                let key = name
                    .strip_prefix("item_")
                    .unwrap()
                    .strip_suffix(".xml")
                    .unwrap();
                theme.items.insert(key.to_string(), Cow::Owned(s));
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
