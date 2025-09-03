use crate::{config::get_config, keybinds::Keybind, providers::Provider};

#[derive(Debug)]
pub struct Websearch {
    keybinds: Vec<Keybind>,
    default_action: String,
}

impl Websearch {
    pub fn new() -> Self {
        let config = get_config();

        Self {
            default_action: config.providers.websearch.default.clone(),
            keybinds: vec![Keybind {
                bind: config.providers.websearch.search.clone(),
                action: "search".to_string(),
                after: crate::keybinds::AfterAction::Close,
            }],
        }
    }
}

impl Provider for Websearch {
    fn get_keybinds(&self) -> &Vec<Keybind> {
        &self.keybinds
    }

    fn default_action(&self) -> &str {
        &self.default_action
    }
}
