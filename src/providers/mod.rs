use std::{
    collections::{HashMap, HashSet},
    fmt::Debug,
    path::Path,
    process::Command,
    sync::OnceLock,
};

use gtk4::{
    Builder, Image, Label, ListItem, Picture, gdk,
    gio::{self, prelude::FileExtManual},
    glib,
    prelude::WidgetExt,
};

use crate::{
    config::get_config,
    keybinds::Action,
    protos::generated_proto::query::query_response::Item,
    providers::{
        actionsmenu::ActionsMenu, archlinuxpkgs::ArchLinuxPkgs, bookmarks::Bookmarks, calc::Calc,
        clipboard::Clipboard, default_provider::DefaultProvider, dmenu::Dmenu,
        emergency::Emergency, files::Files, providerlist::Providerlist, symbols::Symbols,
        todo::Todo, unicode::Unicode,
    },
};

pub mod actionsmenu;
pub mod archlinuxpkgs;
pub mod bookmarks;
pub mod calc;
pub mod clipboard;
pub mod default_provider;
pub mod dmenu;
pub mod emergency;
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
                    unset: None,
                    action: "activate".to_string(),
                    default: Some(true),
                    bind: Some("Return".to_string()),
                    after: None,
                    label: None,
                }]
            })
    }

    fn get_keybind_hint(&self, actions: &[String]) -> Vec<Action> {
        let mut present = HashSet::new();

        let mut result: Vec<Action> = self
            .get_actions()
            .iter()
            .map(|a| {
                if a.action.ends_with(":keep") {
                    return match a.action.split_once(":") {
                        Some((first, _)) => {
                            let mut a = a.clone();
                            a.action = first.to_string();
                            a
                        }
                        None => a.clone(),
                    };
                }

                a.clone()
            })
            .filter(|v| {
                if actions.contains(&v.action) {
                    present.insert(v.action.clone());
                }

                actions.contains(&v.action)
            })
            .collect();

        if let Some(r) = get_config().providers.actions.get("fallback") {
            r.iter()
                .map(|a| {
                    if a.action.ends_with(":keep") {
                        return match a.action.split_once(":") {
                            Some((first, _)) => {
                                let mut a = a.clone();
                                a.action = first.to_string();
                                a
                            }
                            None => a.clone(),
                        };
                    }

                    a.clone()
                })
                .filter(|v| actions.contains(&v.action) && !present.contains(&v.action))
                .for_each(|v| {
                    result.push(v);
                });
        }

        if !actions.is_empty() && result.is_empty() {
            result.push(Action {
                unset: None,
                action: actions.first().unwrap().to_string(),
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

    fn get_item_grid_layout(&self) -> String {
        include_str!("../../resources/themes/default/item.xml").to_string()
    }

    fn text_transformer(&self, item: &Item, label: &Label) {
        if item.text.is_empty() {
            label.set_visible(false);
            return;
        }

        label.set_text(&item.text);
    }

    fn subtext_transformer(&self, item: &Item, label: &Label) {
        if item.subtext.is_empty() {
            label.set_visible(false);
            return;
        }

        label.set_text(&item.subtext);
    }

    fn image_transformer(&self, b: &Builder, i: &ListItem, item: &Item) {
        shared_image_transformer(b, i, item);
    }
}

pub fn shared_image_transformer(b: &Builder, _: &ListItem, item: &Item) {
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

pub static PROVIDERS: OnceLock<HashMap<String, Box<dyn Provider>>> = OnceLock::new();

pub fn setup_providers(elephant: bool) {
    let mut providers: HashMap<String, Box<dyn Provider>> = HashMap::new();
    providers.insert("dmenu".to_string(), Box::new(Dmenu::new()));
    providers.insert("actionmenu".to_string(), Box::new(ActionsMenu::new()));
    providers.insert("emergency".to_string(), Box::new(Emergency::new()));

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
            "bookmarks" => providers.insert("bookmarks".to_string(), Box::new(Bookmarks::new())),
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
