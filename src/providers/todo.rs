use std::path::Path;

use gtk4::{Builder, Image, ListItem, prelude::WidgetExt};

use crate::{
    config::get_config,
    keybinds::{Action, AfterAction, Keybind},
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
                    action: Action {
                        label: "save",
                        action: "save".to_string(),
                        after: AfterAction::ClearReload,
                        required_states: None,
                    },
                },
                Keybind {
                    bind: config.providers.todo.delete.clone(),
                    action: Action {
                        label: "delete",
                        action: "delete".to_string(),
                        after: AfterAction::ClearReload,
                        required_states: None,
                    },
                },
                Keybind {
                    bind: config.providers.todo.mark_active.clone(),
                    action: Action {
                        label: "active",
                        action: "active".to_string(),
                        after: AfterAction::ClearReload,
                        required_states: None,
                    },
                },
                Keybind {
                    bind: config.providers.todo.mark_done.clone(),
                    action: Action {
                        label: "done",
                        action: "done".to_string(),
                        after: AfterAction::ClearReload,
                        required_states: None,
                    },
                },
                Keybind {
                    bind: config.providers.todo.clear.clone(),
                    action: Action {
                        label: "clear",
                        action: "clear".to_string(),
                        after: AfterAction::ClearReload,
                        required_states: None,
                    },
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

    fn get_item_layout(&self) -> String {
        include_str!("../../resources/themes/default/item_todo.xml").to_string()
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
