use crate::{config::get_config, keybinds::Keybind, providers::Provider};

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
                action: "copy".to_string(),
                after: crate::keybinds::AfterAction::Close,
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
}
