use crate::{
    config::get_config,
    keybinds::{Action, AfterAction, Keybind},
    providers::Provider,
};

#[derive(Debug)]
pub struct Websearch {
    keybinds: Vec<Keybind>,
    default_action: String,
}

impl Websearch {
    pub fn new() -> Self {
        let config = get_config();

        Self {
            default_action: config.providers.websearch.default.clone(),
            keybinds: vec![Keybind {
                bind: config.providers.websearch.search.clone(),
                action: Action {
                    action: "search".to_string(),
                    after: AfterAction::Close,
                },
            }],
        }
    }
}

impl Provider for Websearch {
    fn get_keybinds(&self) -> &Vec<Keybind> {
        &self.keybinds
    }

    fn default_action(&self) -> &str {
        &self.default_action
    }

    fn get_keybind_hint(&self, cfg: &crate::config::Elephant) -> String {
        format!("search: {}", cfg.providers.websearch.search)
    }
}
