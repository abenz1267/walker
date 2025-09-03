use crate::{config::get_config, keybinds::Keybind, providers::Provider};

#[derive(Debug)]
pub struct Files {
    keybinds: Vec<Keybind>,
}

impl Files {
    pub fn new() -> Self {
        let config = get_config();

        Self {
            keybinds: vec![
                Keybind {
                    bind: config.providers.files.copy_file.clone(),
                    action: "copyfile".to_string(),
                    after: crate::keybinds::AfterAction::Close,
                },
                Keybind {
                    bind: config.providers.files.copy_path.clone(),
                    action: "copypath".to_string(),
                    after: crate::keybinds::AfterAction::Close,
                },
                Keybind {
                    bind: config.providers.files.open.clone(),
                    action: "open".to_string(),
                    after: crate::keybinds::AfterAction::Close,
                },
                Keybind {
                    bind: config.providers.files.open_dir.clone(),
                    action: "opendir".to_string(),
                    after: crate::keybinds::AfterAction::Close,
                },
            ],
        }
    }
}

impl Provider for Files {
    fn get_keybinds(&self) -> &Vec<Keybind> {
        &self.keybinds
    }
}
