use crate::{
    config::{Elephant, get_config},
    keybinds::Keybind,
    providers::Provider,
};

#[derive(Debug)]
pub struct DesktopApplications {
    keybinds: Vec<Keybind>,
    default_action: String,
}

impl DesktopApplications {
    pub fn new() -> Self {
        let config = get_config();

        Self {
            default_action: config.providers.desktopapplications.default.clone(),
            keybinds: vec![Keybind {
                bind: config.providers.desktopapplications.start.clone(),
                action: "".to_string(),
                after: crate::keybinds::AfterAction::Close,
            }],
        }
    }
}

impl Provider for DesktopApplications {
    fn get_keybinds(&self) -> &Vec<Keybind> {
        &self.keybinds
    }

    fn default_action(&self) -> &str {
        &self.default_action
    }

    fn get_keybind_hint(&self, cfg: &Elephant) -> String {
        format!("start: {}", cfg.providers.desktopapplications.start)
    }
}
