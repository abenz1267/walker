use crate::{
    config::get_config,
    keybinds::{Action, AfterAction, Keybind},
    providers::Provider,
};

#[derive(Debug)]
pub struct Bluetooth {
    keybinds: Vec<Keybind>,
    default_action: String,
    global_keybinds: Vec<Keybind>,
}

impl Bluetooth {
    pub fn new() -> Self {
        let config = get_config();

        Self {
            default_action: config.providers.desktopapplications.default.clone(),
            global_keybinds: vec![Keybind {
                bind: config.providers.bluetooth.find.clone(),
                action: Action {
                    label: "find",
                    required_states: None,
                    action: "find".to_string(),
                    after: AfterAction::ClearReload,
                },
            }],
            keybinds: vec![
                Keybind {
                    bind: config.providers.bluetooth.connect.clone(),
                    action: Action {
                        label: "connect",
                        action: "connect".to_string(),
                        after: AfterAction::Nothing,
                        required_states: Some(vec!["can_connect"]),
                    },
                },
                Keybind {
                    bind: config.providers.bluetooth.disconnect.clone(),
                    action: Action {
                        label: "disconnect",
                        action: "disconnect".to_string(),
                        after: AfterAction::Nothing,
                        required_states: Some(vec!["can_disconnect"]),
                    },
                },
                Keybind {
                    bind: config.providers.bluetooth.remove.clone(),
                    action: Action {
                        label: "remove",
                        action: "remove".to_string(),
                        after: AfterAction::Nothing,
                        required_states: Some(vec!["can_remove"]),
                    },
                },
                Keybind {
                    bind: config.providers.bluetooth.pair.clone(),
                    action: Action {
                        label: "pair",
                        action: "pair".to_string(),
                        after: AfterAction::Nothing,
                        required_states: Some(vec!["can_pair"]),
                    },
                },
                Keybind {
                    bind: config.providers.bluetooth.trust.clone(),
                    action: Action {
                        label: "trust",
                        action: "trust".to_string(),
                        after: AfterAction::Nothing,
                        required_states: Some(vec!["can_trust"]),
                    },
                },
                Keybind {
                    bind: config.providers.bluetooth.untrust.clone(),
                    action: Action {
                        label: "untrust",
                        action: "untrust".to_string(),
                        after: AfterAction::Nothing,
                        required_states: Some(vec!["can_untrust"]),
                    },
                },
            ],
        }
    }
}

impl Provider for Bluetooth {
    fn get_keybinds(&self) -> &Vec<Keybind> {
        &self.keybinds
    }

    fn get_global_keybinds(&self) -> Option<&Vec<Keybind>> {
        Some(&self.global_keybinds)
    }

    fn default_action(&self) -> &str {
        &self.default_action
    }
}
