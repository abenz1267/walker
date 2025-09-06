use crate::{
    config::{Elephant, get_config},
    keybinds::{Action, AfterAction, Keybind},
    providers::Provider,
};

#[derive(Debug)]
pub struct ArchLinuxPkgs {
    keybinds: Vec<Keybind>,
    default_action: String,
}

impl ArchLinuxPkgs {
    pub fn new() -> Self {
        let config = get_config();

        Self {
            default_action: config.providers.archlinuxpkgs.default.clone(),
            keybinds: vec![
                Keybind {
                    bind: config.providers.archlinuxpkgs.install.clone(),
                    action: Action {
                        label: "install",
                        required_states: Some(vec!["available"]),
                        action: "install".to_string(),
                        after: AfterAction::Close,
                    },
                },
                Keybind {
                    bind: config.providers.archlinuxpkgs.remove.clone(),
                    action: Action {
                        label: "remove",
                        action: "remove".to_string(),
                        required_states: Some(vec!["installed"]),
                        after: AfterAction::Close,
                    },
                },
            ],
        }
    }
}

impl Provider for ArchLinuxPkgs {
    fn get_keybinds(&self) -> &Vec<Keybind> {
        &self.keybinds
    }

    fn default_action(&self) -> &str {
        &self.default_action
    }

    fn get_item_layout(&self) -> String {
        include_str!("../../resources/themes/default/item_archlinuxpkgs.xml").to_string()
    }
}
