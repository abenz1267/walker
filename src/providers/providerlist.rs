use crate::{
    config::{Elephant, get_config},
    keybinds::{Action, Keybind},
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
                    action: "activate",
                    after: crate::keybinds::AfterAction::ClearReload,
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

    fn get_keybind_hint(&self, cfg: &Elephant) -> String {
        format!("select: {}", cfg.providers.providerlist.activate)
    }

    fn get_item_layout(&self) -> &'static str {
        include_str!("../../resources/themes/default/item_providerlist.xml")
    }
}
