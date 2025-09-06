use crate::{
    config::{Elephant, get_config},
    keybinds::{Action, AfterAction, Keybind},
    providers::Provider,
};

#[derive(Debug)]
pub struct Menus {
    keybinds: Vec<Keybind>,
    default_action: String,
}

impl Menus {
    pub fn new() -> Self {
        let config = get_config();

        Self {
            default_action: config.providers.menus.default.clone(),
            keybinds: vec![
                Keybind {
                    bind: config.providers.menus.activate.clone(),
                    action: Action {
                        label: "select",
                        required_states: None,
                        action: "activate".to_string(),
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

impl Provider for Menus {
    fn get_keybinds(&self) -> &Vec<Keybind> {
        &self.keybinds
    }

    fn default_action(&self) -> &str {
        &self.default_action
    }
}
