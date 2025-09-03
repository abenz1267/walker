use crate::{config::get_config, keybinds::Keybind, providers::Provider};

#[derive(Debug)]
pub struct Todo {
    keybinds: Vec<Keybind>,
    default_action: String,
}

impl Todo {
    pub fn new() -> Self {
        let config = get_config();

        Self {
            default_action: config.providers.todo.default.clone(),
            keybinds: vec![
                Keybind {
                    bind: config.providers.todo.save.clone(),
                    action: "save".to_string(),
                    after: crate::keybinds::AfterAction::ClearReloadKeepPrefix,
                },
                Keybind {
                    bind: config.providers.todo.delete.clone(),
                    action: "delete".to_string(),
                    after: crate::keybinds::AfterAction::ClearReloadKeepPrefix,
                },
                Keybind {
                    bind: config.providers.todo.mark_active.clone(),
                    action: "active".to_string(),
                    after: crate::keybinds::AfterAction::ClearReloadKeepPrefix,
                },
                Keybind {
                    bind: config.providers.todo.mark_done.clone(),
                    action: "done".to_string(),
                    after: crate::keybinds::AfterAction::ClearReloadKeepPrefix,
                },
                Keybind {
                    bind: config.providers.todo.clear.clone(),
                    action: "clear".to_string(),
                    after: crate::keybinds::AfterAction::ClearReloadKeepPrefix,
                },
            ],
        }
    }
}

impl Provider for Todo {
    fn get_keybinds(&self) -> &Vec<Keybind> {
        &self.keybinds
    }

    fn default_action(&self) -> &str {
        &self.default_action
    }
}
