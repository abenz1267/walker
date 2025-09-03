use crate::{config::get_config, keybinds::Keybind, providers::Provider};

#[derive(Debug)]
pub struct Runner {
    keybinds: Vec<Keybind>,
}

impl Runner {
    pub fn new() -> Self {
        let config = get_config();

        Self {
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
}
