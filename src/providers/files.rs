use crate::{
    config::{Elephant, get_config},
    keybinds::Keybind,
    providers::Provider,
};

#[derive(Debug)]
pub struct Files {
    keybinds: Vec<Keybind>,
    default_action: String,
}

impl Files {
    pub fn new() -> Self {
        let config = get_config();

        Self {
            default_action: config.providers.files.default.clone(),
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

    fn default_action(&self) -> &str {
        &self.default_action
    }

    fn get_keybind_hint(&self, cfg: &Elephant) -> String {
        format!(
            "open: {} - open dir: {} - copy: {} - copy path: {}",
            cfg.providers.files.open,
            cfg.providers.files.open_dir,
            cfg.providers.files.copy_file,
            cfg.providers.files.copy_path
        )
    }

    fn get_item_layout(&self) -> String {
        include_str!("../../resources/themes/default/item_files.xml").to_string()
    }
}
