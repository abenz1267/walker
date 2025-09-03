use crate::{config::get_config, keybinds::Keybind, providers::Provider};

#[derive(Debug)]
pub struct DesktopApplications {
    keybinds: Vec<Keybind>,
}

impl DesktopApplications {
    pub fn new() -> Self {
        let config = get_config();

        Self {
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
}
