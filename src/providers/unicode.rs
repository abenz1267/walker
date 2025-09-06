use gtk4::{Builder, Label, ListItem};

use crate::{
    config::{Elephant, get_config},
    keybinds::{Action, AfterAction, Keybind},
    protos::generated_proto::query::query_response::Item,
    providers::Provider,
};

#[derive(Debug)]
pub struct Unicode {
    keybinds: Vec<Keybind>,
    default_action: String,
}

impl Unicode {
    pub fn new() -> Self {
        let config = get_config();

        Self {
            default_action: config.providers.unicode.default.clone(),
            keybinds: vec![
                Keybind {
                    bind: config.providers.unicode.copy.clone(),
                    action: Action {
                        label: "copy",
                        required_states: None,
                        action: "copy".to_string(),
                        after: AfterAction::Close,
                    },
                },
                Keybind {
                    bind: config.providers.desktopapplications.remove_history.clone(),
                    action: Action {
                        label: "erase history",
                        action: "erase_history".to_string(),
                        after: AfterAction::Reload,
                        required_states: Some(vec!["history"]),
                    },
                },
            ],
        }
    }
}

impl Provider for Unicode {
    fn get_keybinds(&self) -> &Vec<Keybind> {
        &self.keybinds
    }

    fn default_action(&self) -> &str {
        &self.default_action
    }

    fn get_item_layout(&self) -> String {
        include_str!("../../resources/themes/default/item_unicode.xml").to_string()
    }

    fn image_transformer(&self, b: &Builder, _: &ListItem, item: &Item) {
        if let Some(image) = b.object::<Label>("ItemImage")
            && !item.icon.is_empty()
            && let Ok(code_point) = u32::from_str_radix(&item.icon, 16)
            && let Some(unicode_char) = char::from_u32(code_point)
        {
            image.set_label(&format!("{unicode_char}"));
        }
    }
}
