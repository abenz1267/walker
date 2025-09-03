use crate::{config::get_config, keybinds::Keybind, providers::Provider};

#[derive(Debug)]
pub struct Dmenu {
    keybinds: Vec<Keybind>,
    default_action: String,
}

impl Dmenu {
    pub fn new() -> Self {
        let config = get_config();

        Self {
            default_action: config.providers.dmenu.default.clone(),
            keybinds: vec![Keybind {
                bind: config.providers.dmenu.select.clone(),
                action: "select".to_string(),
                after: crate::keybinds::AfterAction::Close,
            }],
        }
    }
}

impl Provider for Dmenu {
    fn get_keybinds(&self) -> &Vec<Keybind> {
        &self.keybinds
    }

    fn default_action(&self) -> &str {
        &self.default_action
    }
}
