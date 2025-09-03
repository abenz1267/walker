use crate::{config::get_config, keybinds::Keybind, providers::Provider};

#[derive(Debug)]
pub struct Websearch {
    keybinds: Vec<Keybind>,
}

impl Websearch {
    pub fn new() -> Self {
        let config = get_config();

        Self {
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
}
