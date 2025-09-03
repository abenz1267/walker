use crate::{config::get_config, keybinds::Keybind, providers::Provider};

#[derive(Debug)]
pub struct Runner {
    keybinds: Vec<Keybind>,
    default_action: String,
}

impl Runner {
    pub fn new() -> Self {
        let config = get_config();

        Self {
            default_action: config.providers.runner.default.clone(),
            keybinds: vec![
                Keybind {
                    bind: config.providers.runner.start.clone(),
                    action: "run".to_string(),
                    after: crate::keybinds::AfterAction::Close,
                },
                Keybind {
                    bind: config.providers.runner.start_terminal.clone(),
                    action: "runterminal".to_string(),
                    after: crate::keybinds::AfterAction::Close,
                },
            ],
        }
    }
}

impl Provider for Runner {
    fn get_keybinds(&self) -> &Vec<Keybind> {
        &self.keybinds
    }

    fn default_action(&self) -> &str {
        &self.default_action
    }
}
