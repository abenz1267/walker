use gtk4::{Builder, Label, ListItem};

use crate::{
    config::{Elephant, get_config},
    keybinds::{Action, Keybind},
    protos::generated_proto::query::query_response::Item,
    providers::Provider,
};

#[derive(Debug)]
pub struct Symbols {
    keybinds: Vec<Keybind>,
    default_action: String,
}

impl Symbols {
    pub fn new() -> Self {
        let config = get_config();

        Self {
            default_action: config.providers.symbols.default.clone(),
            keybinds: vec![Keybind {
                bind: config.providers.symbols.copy.clone(),
                action: Action {
                    action: "copy",
                    after: crate::keybinds::AfterAction::Close,
                },
            }],
        }
    }
}

impl Provider for Symbols {
    fn get_keybinds(&self) -> &Vec<Keybind> {
        &self.keybinds
    }

    fn default_action(&self) -> &str {
        &self.default_action
    }

    fn get_keybind_hint(&self, cfg: &Elephant) -> String {
        format!("copy: {}", cfg.providers.symbols.copy)
    }

    fn get_default_item_layout(&self) -> &'static str {
        include_str!("../../resources/themes/default/item_symbols.xml")
    }

    fn image_transformer(&self, b: &Builder, _: &ListItem, item: &Item) {
        if let Some(image) = b.object::<Label>("ItemImage")
            && !item.icon.is_empty()
        {
            image.set_label(&item.icon);
        }
    }
}
