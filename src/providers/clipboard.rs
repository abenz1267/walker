use crate::{config::get_config, keybinds::Keybind, providers::Provider};

#[derive(Debug)]
pub struct Clipboard {
    keybinds: Vec<Keybind>,
    default_action: String,
}

impl Clipboard {
    pub fn new() -> Self {
        let config = get_config();

        Self {
            default_action: config.providers.clipboard.default.clone(),
            keybinds: vec![
                Keybind {
                    bind: config.providers.clipboard.copy.clone(),
                    action: "copy".to_string(),
                    after: crate::keybinds::AfterAction::Close,
                },
                Keybind {
                    bind: config.providers.clipboard.delete.clone(),
                    action: "remove".to_string(),
                    after: crate::keybinds::AfterAction::Reload,
                },
                Keybind {
                    bind: config.providers.clipboard.edit.clone(),
                    action: "edit".to_string(),
                    after: crate::keybinds::AfterAction::Close,
                },
                Keybind {
                    bind: config.providers.clipboard.toggle_images_only.clone(),
                    action: "toggle_images".to_string(),
                    after: crate::keybinds::AfterAction::ClearReloadKeepPrefix,
                },
            ],
        }
    }
}

impl Provider for Clipboard {
    fn get_keybinds(&self) -> &Vec<Keybind> {
        &self.keybinds
    }

    fn default_action(&self) -> &str {
        &self.default_action
    }
}
