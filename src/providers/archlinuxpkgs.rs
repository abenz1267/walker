use crate::{
    config::{Elephant, get_config},
    keybinds::Keybind,
    providers::Provider,
};

#[derive(Debug)]
pub struct ArchLinuxPkgs {
    keybinds: Vec<Keybind>,
    default_action: String,
}

impl ArchLinuxPkgs {
    pub fn new() -> Self {
        let config = get_config();

        Self {
            default_action: config.providers.archlinuxpkgs.default.clone(),
            keybinds: vec![
                Keybind {
                    bind: config.providers.archlinuxpkgs.install.clone(),
                    action: "install".to_string(),
                    after: crate::keybinds::AfterAction::Close,
                },
                Keybind {
                    bind: config.providers.archlinuxpkgs.remove.clone(),
                    action: "remove".to_string(),
                    after: crate::keybinds::AfterAction::Close,
                },
            ],
        }
    }
}

impl Provider for ArchLinuxPkgs {
    fn get_keybinds(&self) -> &Vec<Keybind> {
        &self.keybinds
    }

    fn default_action(&self) -> &str {
        &self.default_action
    }

    fn get_keybind_hint(&self, cfg: &Elephant) -> String {
        format!(
            "install: {} - remove: {}",
            cfg.providers.archlinuxpkgs.install, cfg.providers.archlinuxpkgs.remove,
        )
    }
}
