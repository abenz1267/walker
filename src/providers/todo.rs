use std::path::Path;

use gtk4::{Builder, Image, ListItem, prelude::WidgetExt};

use crate::{
    config::{Elephant, get_config},
    keybinds::Keybind,
    protos::generated_proto::query::query_response::Item,
    providers::Provider,
};

#[derive(Debug)]
pub struct Todo {
    keybinds: Vec<Keybind>,
    default_action: String,
}

impl Todo {
    pub fn new() -> Self {
        let config = get_config();

        Self {
            default_action: config.providers.todo.default.clone(),
            keybinds: vec![
                Keybind {
                    bind: config.providers.todo.save.clone(),
                    action: "save".to_string(),
                    after: crate::keybinds::AfterAction::ClearReloadKeepPrefix,
                },
                Keybind {
                    bind: config.providers.todo.delete.clone(),
                    action: "delete".to_string(),
                    after: crate::keybinds::AfterAction::ClearReloadKeepPrefix,
                },
                Keybind {
                    bind: config.providers.todo.mark_active.clone(),
                    action: "active".to_string(),
                    after: crate::keybinds::AfterAction::ClearReloadKeepPrefix,
                },
                Keybind {
                    bind: config.providers.todo.mark_done.clone(),
                    action: "done".to_string(),
                    after: crate::keybinds::AfterAction::ClearReloadKeepPrefix,
                },
                Keybind {
                    bind: config.providers.todo.clear.clone(),
                    action: "clear".to_string(),
                    after: crate::keybinds::AfterAction::ClearReloadKeepPrefix,
                },
            ],
        }
    }
}

impl Provider for Todo {
    fn get_keybinds(&self) -> &Vec<Keybind> {
        &self.keybinds
    }

    fn default_action(&self) -> &str {
        &self.default_action
    }

    fn get_keybind_hint(&self, cfg: &Elephant) -> String {
        format!(
            "mark active: {} - mark done: {} - delete: {} - clear: {}",
            cfg.providers.todo.mark_active,
            cfg.providers.todo.mark_done,
            cfg.providers.todo.delete,
            cfg.providers.todo.clear
        )
    }

    fn get_default_item_layout(&self) -> &'static str {
        include_str!("../../resources/themes/default/item_todo.xml")
    }

    fn image_transformer(&self, b: &Builder, _: &ListItem, item: &Item) {
        let Some(image) = b.object::<Image>("ItemImage") else {
            return;
        };

        if !item.state.contains(&"creating".to_string()) {
            image.set_visible(false);
            return;
        }

        let function = if !item.icon.is_empty() && Path::new(&item.icon).is_absolute() {
            Image::set_from_file
        } else if !item.icon.is_empty() {
            Image::set_icon_name
        } else {
            return;
        };
        function(&image, Some(&item.icon))
    }
}
