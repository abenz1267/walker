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
                        label: "run",
                        required_states: None,
                        action: "run".to_string(),
                        after: AfterAction::Close,
                    },
                },
                Keybind {
                    bind: config.providers.runner.start_terminal.clone(),
                    action: Action {
                        label: "run in terminal",
                        required_states: None,
                        action: "runterminal".to_string(),
                        after: AfterAction::Close,
                    },
                },
                Keybind {
                    bind: config.providers.desktopapplications.remove_history.clone(),
                    action: Action {
                        label: "erase history",
                        action: "erase_history".to_string(),
                        after: AfterAction::Reload,
                        required_states: Some(vec!["history"]),
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
}
