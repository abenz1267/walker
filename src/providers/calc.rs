use crate::{
    config::{Elephant, get_config},
    keybinds::Keybind,
    providers::Provider,
};

#[derive(Debug)]
pub struct Calc {
    keybinds: Vec<Keybind>,
    default_action: String,
}

impl Calc {
    pub fn new() -> Self {
        let config = get_config();

        Self {
            default_action: config.providers.calc.default.clone(),
            keybinds: vec![
                Keybind {
                    bind: config.providers.calc.copy.clone(),
                    action: "copy".to_string(),
                    after: crate::keybinds::AfterAction::Close,
                },
                Keybind {
                    bind: config.providers.calc.delete.clone(),
                    action: "delete".to_string(),
                    after: crate::keybinds::AfterAction::Reload,
                },
                Keybind {
                    bind: config.providers.calc.save.clone(),
                    action: "save".to_string(),
                    after: crate::keybinds::AfterAction::Reload,
                },
            ],
        }
    }
}

impl Provider for Calc {
    fn get_keybinds(&self) -> &Vec<Keybind> {
        &self.keybinds
    }

    fn default_action(&self) -> &str {
        &self.default_action
    }

    fn get_keybind_hint(&self, cfg: &Elephant) -> String {
        format!(
            "copy: {} - save: {} - delete: {}",
            cfg.providers.calc.copy, cfg.providers.calc.save, cfg.providers.calc.delete
        )
    }
}
