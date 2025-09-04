use gtk4::{Builder, Label, ListItem};

use crate::{
    config::{Elephant, get_config},
    keybinds::{Action, Keybind},
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
            keybinds: vec![Keybind {
                bind: config.providers.unicode.copy.clone(),
                action: Action {
                    action: "copy",
                    after: crate::keybinds::AfterAction::Close,
                },
            }],
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

    fn get_keybind_hint(&self, cfg: &Elephant) -> String {
        format!("copy: {}", cfg.providers.unicode.copy)
    }

    fn get_default_item_layout(&self) -> &'static str {
        include_str!("../../resources/themes/default/item_unicode.xml")
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
