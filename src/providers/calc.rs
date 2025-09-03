use crate::{config::get_config, keybinds::Keybind, providers::Provider};

#[derive(Debug)]
pub struct Calc {
    keybinds: Vec<Keybind>,
}

impl Calc {
    pub fn new() -> Self {
        let config = get_config();

        Self {
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
}
