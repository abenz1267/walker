use crate::{
    config::{Elephant, get_config},
    keybinds::Keybind,
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
                action: "copy".to_string(),
                after: crate::keybinds::AfterAction::Close,
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
        format!("copy: {}", cfg.providers.unicode.copy,)
    }

    fn get_item_layout(&self) -> String {
        include_str!("../../resources/themes/default/item_unicode.xml").to_string()
    }
}
