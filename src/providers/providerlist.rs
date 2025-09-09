use crate::{
    config::get_config,
    keybinds::{Action, AfterAction, Keybind},
    providers::Provider,
};

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
                action: Action {
                    label: "select",
                    required_states: None,
                    action: "activate".to_string(),
                    after: AfterAction::ClearReload,
                },
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

    fn get_item_layout(&self) -> String {
        include_str!("../../resources/themes/default/item_providerlist.xml").to_string()
    }
}
