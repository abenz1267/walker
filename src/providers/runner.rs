use crate::{
    config::{Elephant, get_config},
    keybinds::{Action, AfterAction, Keybind},
    providers::Provider,
};

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
                    action: Action {
                        action: "run",
                        after: AfterAction::Close,
                    },
                },
                Keybind {
                    bind: config.providers.runner.start_terminal.clone(),
                    action: Action {
                        action: "runterminal",
                        after: AfterAction::Close,
                    },
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

    fn get_keybind_hint(&self, cfg: &Elephant) -> String {
        format!(
            "run: {} - run in terminal: {}",
            cfg.providers.runner.start, cfg.providers.runner.start_terminal
        )
    }
}
