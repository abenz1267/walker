use crate::{config::get_config, keybinds::Keybind, providers::Provider};

#[derive(Debug)]
pub struct Providerlist {
    keybinds: Vec<Keybind>,
    default_action: String,
}

impl Providerlist {
    pub fn new() -> Self {
        let config = get_config();

        Self {
            default_action: config.providers.providerlist.default.clone(),
            keybinds: vec![Keybind {
                bind: config.providers.providerlist.activate.clone(),
                action: "activate".to_string(),
                after: crate::keybinds::AfterAction::ClearReload,
            }],
        }
    }
}

impl Provider for Providerlist {
    fn get_keybinds(&self) -> &Vec<Keybind> {
        &self.keybinds
    }

    fn default_action(&self) -> &str {
        &self.default_action
    }
}
