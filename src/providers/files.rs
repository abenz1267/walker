use std::env;

use gtk4::{
    Builder, Image, ListItem,
    gio::{self, prelude::FileExt},
};

use crate::{
    config::get_config,
    keybinds::{Action, AfterAction, Keybind},
    protos::generated_proto::query::query_response::Item,
    providers::Provider,
};

#[derive(Debug)]
pub struct Files {
    keybinds: Vec<Keybind>,
    default_action: String,
}

impl Files {
    pub fn new() -> Self {
        let config = get_config();

        Self {
            default_action: config.providers.files.default.clone(),
            keybinds: vec![
                Keybind {
                    bind: config.providers.files.copy_file.clone(),
                    action: Action {
                        label: "copy",
                        action: "copy".to_string(),
                        after: AfterAction::Close,
                        required_states: None,
                    },
                },
                Keybind {
                    bind: config.providers.files.copy_path.clone(),
                    action: Action {
                        label: "copy path",
                        action: "copypath".to_string(),
                        after: AfterAction::Close,
                        required_states: None,
                    },
                },
                Keybind {
                    bind: config.providers.files.open.clone(),
                    action: Action {
                        label: "open",
                        action: "open".to_string(),
                        after: AfterAction::Close,
                        required_states: None,
                    },
                },
                Keybind {
                    bind: config.providers.files.open_dir.clone(),
                    action: Action {
                        label: "open dir",
                        action: "opendir".to_string(),
                        after: AfterAction::Close,
                        required_states: None,
                    },
                },
            ],
        }
    }
}

impl Provider for Files {
    fn get_keybinds(&self) -> &Vec<Keybind> {
        &self.keybinds
    }

    fn default_action(&self) -> &str {
        &self.default_action
    }

    fn get_item_layout(&self) -> String {
        include_str!("../../resources/themes/default/item_files.xml").to_string()
    }

    fn text_transformer(&self, text: &str, label: &gtk4::Label) {
        if let Ok(home) = env::var("HOME")
            && let Some(stripped) = text.strip_prefix(&home)
        {
            label.set_label(stripped);
        }
    }

    fn image_transformer(&self, b: &Builder, _: &ListItem, item: &Item) {
        let Some(image) = b.object::<Image>("ItemImage") else {
            return;
        };

        let file = gio::File::for_path(&item.text);

        let info = file.query_info(
            "standard::icon",
            gio::FileQueryInfoFlags::NONE,
            gio::Cancellable::NONE,
        );

        if let Ok(info) = info
            && let Some(icon) = info.icon()
        {
            image.set_from_gicon(&icon);
        }
    }
}
