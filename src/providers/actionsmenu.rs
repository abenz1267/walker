use crate::providers::Provider;

#[derive(Debug)]
pub struct ActionsMenu {
    name: &'static str,
}

impl ActionsMenu {
    pub fn new() -> Self {
        Self {
            name: "actionsmenu",
        }
    }
}

impl Provider for ActionsMenu {
    fn get_name(&self) -> &str {
        self.name
    }

    fn get_item_layout(&self) -> String {
        include_str!("../../resources/themes/default/item_actionsmenu.xml").to_string()
    }
}
