use crate::{config::get_config, keybinds::Keybind, providers::Provider};

#[derive(Debug)]
pub struct Unicode {
    keybinds: Vec<Keybind>,
}

impl Unicode {
    pub fn new() -> Self {
        let config = get_config();

        Self {
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
}
