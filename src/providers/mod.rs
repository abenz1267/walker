use std::{
    ascii::AsciiExt, collections::HashMap, fmt::Debug, path::Path, process::Command, sync::OnceLock,
};

use gtk4::{
    Builder, Image, Label, ListItem, Picture,
    ffi::GtkLabel,
    gdk,
    gio::{self, prelude::FileExtManual},
    glib,
    prelude::WidgetExt,
};

use crate::{
    config::get_config,
    keybinds::Action,
    protos::generated_proto::query::query_response::Item,
    providers::{
        archlinuxpkgs::ArchLinuxPkgs, calc::Calc, clipboard::Clipboard,
        default_provider::DefaultProvider, dmenu::Dmenu, files::Files, providerlist::Providerlist,
        symbols::Symbols, todo::Todo, unicode::Unicode,
    },
};

pub mod archlinuxpkgs;
pub mod calc;
pub mod clipboard;
pub mod default_provider;
pub mod dmenu;
pub mod files;
pub mod providerlist;
pub mod symbols;
pub mod todo;
pub mod unicode;

pub trait Provider: Sync + Send + Debug {
    fn get_name(&self) -> &str;

    fn get_actions(&self) -> Vec<Action> {
        get_config()
            .providers
            .actions
            .get(self.get_name())
            .cloned()
            .unwrap_or_else(|| {
                vec![Action {
                    action: "activate".to_string(),
                    default: Some(true),
                    bind: Some("Return".to_string()),
                    global: None,
                    after: None,
                    label: None,
                }]
            })
    }

    fn get_keybind_hint(&self, actions: &[String]) -> Vec<Action> {
        let mut result: Vec<Action> = self
            .get_actions()
            .iter()
            .filter(|v| actions.contains(&v.action) || v.global.unwrap_or(false))
            .cloned()
            .collect();

        if result.is_empty()
            || (result.len() == 1 && result.first().unwrap().global.unwrap_or(false))
        {
            result.push(Action {
                action: actions.first().unwrap().to_string(),
                global: None,
                default: Some(true),
                bind: Some("Return".to_string()),
                after: None,
                label: None,
            });
        }

        result.sort_by_key(|v| v.default.unwrap_or(false));
        result
    }

    fn get_item_layout(&self) -> String {
        include_str!("../../resources/themes/default/item.xml").to_string()
    }

    fn text_transformer(&self, text: &str, label: &Label) {
        if text.is_empty() {
            label.set_visible(false);
            return;
        }

        label.set_text(text);
    }

    fn subtext_transformer(&self, item: &Item, label: &Label) {
        if item.subtext.is_empty() {
            label.set_visible(false);
            return;
        }

        label.set_text(&item.subtext);
    }

    fn image_transformer(&self, b: &Builder, _: &ListItem, item: &Item) {
        let mut is_text = false;

        if let Some(image) = b.object::<Label>("ItemImageFont") {
            image.set_visible(false);

            if !item.icon.is_ascii() {
                image.set_text(&item.icon);
                image.set_visible(true);
                is_text = true;
            }
        }

        if let Some(image) = b.object::<Image>("ItemImage") {
            image.set_visible(true);
            if is_text {
                image.set_visible(false);
                return;
            }

            if item.icon.is_empty() {
                image.set_visible(false);
                return;
            }

            if !Path::new(&item.icon).is_absolute() {
                image.set_icon_name(Some(&item.icon));
                return;
            };

            let icon = item.icon.clone();
            glib::spawn_future_local(async move {
                let Ok((bytes, _)) = gio::File::for_path(&icon).load_contents_future().await else {
                    return;
                };

                let texture = gdk::Texture::from_bytes(&glib::Bytes::from(&bytes)).unwrap();
                image.set_paintable(Some(&texture));
            });
        } else if let Some(image) = b.object::<Picture>("ItemImage") {
            image.set_visible(true);
            if is_text {
                image.set_visible(false);
                return;
            }

            if item.icon.is_empty() {
                image.set_visible(false);
                return;
            }

            let icon = item.icon.clone();

            glib::spawn_future_local(async move {
                let Ok((bytes, _)) = gio::File::for_path(&icon).load_contents_future().await else {
                    return;
                };

                let texture = gdk::Texture::from_bytes(&glib::Bytes::from(&bytes)).unwrap();
                image.set_paintable(Some(&texture));
            });
        }
    }
}

pub static PROVIDERS: OnceLock<HashMap<String, Box<dyn Provider>>> = OnceLock::new();

pub fn setup_providers(elephant: bool) {
    let mut providers: HashMap<String, Box<dyn Provider>> = HashMap::new();
    providers.insert("dmenu".to_string(), Box::new(Dmenu::new()));

    let provider_list: Vec<String> = {
        let config = get_config();

        if let Some(val) = &config.installed_providers {
            val.clone()
        } else if elephant {
            match Command::new("elephant").arg("listproviders").output() {
                Ok(output) => match String::from_utf8(output.stdout) {
                    Ok(stdout) => stdout
                        .lines()
                        .filter_map(|line| line.split_once(';').map(|(_, value)| value.to_string()))
                        .collect(),
                    Err(e) => {
                        eprintln!("Error parsing elephant output as UTF-8: {}", e);
                        Vec::new()
                    }
                },
                Err(e) => {
                    eprintln!(
                        "Error running 'elephant' command: {}. Make sure it is installed.",
                        e
                    );
                    Vec::new()
                }
            }
        } else {
            Vec::new()
        }
    };

    provider_list.into_iter().for_each(|p| {
        match p.as_str() {
            "calc" => providers.insert("calc".to_string(), Box::new(Calc::new())),
            "clipboard" => providers.insert("clipboard".to_string(), Box::new(Clipboard::new())),
            "files" => providers.insert("files".to_string(), Box::new(Files::new())),
            "symbols" => providers.insert("symbols".to_string(), Box::new(Symbols::new())),
            "unicode" => providers.insert("unicode".to_string(), Box::new(Unicode::new())),
            "providerlist" => {
                providers.insert("providerlist".to_string(), Box::new(Providerlist::new()))
            }
            // provider if provider.starts_with("menus:") => providers.insert(
            //     provider.to_string(),
            //     Box::new(DefaultProvider::new(provider.to_string())),
            // ),
            "archlinuxpkgs" => {
                providers.insert("archlinuxpkgs".to_string(), Box::new(ArchLinuxPkgs::new()))
            }
            "todo" => providers.insert("todo".to_string(), Box::new(Todo::new())),
            provider => providers.insert(
                provider.to_string(),
                Box::new(DefaultProvider::new(provider.to_string())),
            ),
        };
    });

    PROVIDERS
        .set(providers)
        .expect("couldn't initialize providers.")
}
