use std::{collections::HashMap, fmt::Debug, path::Path, process::Command, sync::OnceLock};

use gtk4::{
    Builder, Image, Label, ListItem, Picture, gdk,
    gio::{self, prelude::FileExtManual},
    glib,
    prelude::WidgetExt,
};

use crate::{
    config::get_config,
    keybinds::Keybind,
    protos::generated_proto::query::query_response::Item,
    providers::{
        archlinuxpkgs::ArchLinuxPkgs, calc::Calc, clipboard::Clipboard,
        desktopapplications::DesktopApplications, dmenu::Dmenu, files::Files, menus::Menus,
        providerlist::Providerlist, runner::Runner, symbols::Symbols, todo::Todo, unicode::Unicode,
        websearch::Websearch,
    },
};

pub mod archlinuxpkgs;
pub mod calc;
pub mod clipboard;
pub mod desktopapplications;
pub mod dmenu;
pub mod files;
pub mod menus;
pub mod providerlist;
pub mod runner;
pub mod symbols;
pub mod todo;
pub mod unicode;
pub mod websearch;

pub trait Provider: Sync + Send + Debug {
    fn get_keybinds(&self) -> &Vec<Keybind>;

    fn get_global_keybinds(&self) -> Option<&Vec<Keybind>> {
        return None;
    }

    fn default_action(&self) -> &str;

    fn get_keybind_hint(&self, state: &Vec<String>) -> String {
        self.get_keybinds()
            .iter()
            .chain(self.get_global_keybinds().unwrap_or(&Vec::new()).iter())
            .filter(|keybind| match &keybind.action.required_states {
                Some(required) => required
                    .iter()
                    .any(|required_state| state.contains(&required_state.to_string())),
                None => true,
            })
            .map(|keybind| format!("{}: <{}>", keybind.action.label, keybind.bind))
            .collect::<Vec<_>>()
            .join(" | ")
    }

    fn get_item_layout(&self) -> String {
        include_str!("../../resources/themes/default/item.xml").to_string()
    }

    fn text_transformer(&self, text: &str, label: &Label) {
        if text.is_empty() {
            label.set_visible(false);
            return;
        }

        label.set_text(&text);
    }

    fn subtext_transformer(&self, text: &str, label: &Label) {
        if text.is_empty() {
            label.set_visible(false);
            return;
        }

        label.set_text(&text);
    }

    fn image_transformer(&self, b: &Builder, _: &ListItem, item: &Item) {
        if item.icon.is_empty() {
            return;
        }

        if let Some(image) = b.object::<Image>("ItemImage") {
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
            "desktopapplications" => providers.insert(
                "desktopapplications".to_string(),
                Box::new(DesktopApplications::new()),
            ),
            "files" => providers.insert("files".to_string(), Box::new(Files::new())),
            "runner" => providers.insert("runner".to_string(), Box::new(Runner::new())),
            "symbols" => providers.insert("symbols".to_string(), Box::new(Symbols::new())),
            "unicode" => providers.insert("unicode".to_string(), Box::new(Unicode::new())),
            "providerlist" => {
                providers.insert("providerlist".to_string(), Box::new(Providerlist::new()))
            }
            provider if provider.starts_with("menus:") => {
                providers.insert(provider.to_string(), Box::new(Menus::new()))
            }
            "websearch" => providers.insert("websearch".to_string(), Box::new(Websearch::new())),
            "archlinuxpkgs" => {
                providers.insert("archlinuxpkgs".to_string(), Box::new(ArchLinuxPkgs::new()))
            }
            "todo" => providers.insert("todo".to_string(), Box::new(Todo::new())),
            _ => return,
        };
    });

    PROVIDERS
        .set(providers)
        .expect("couldn't initialize providers.")
}
