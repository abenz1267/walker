use crate::providers::Provider;

#[derive(Debug)]
pub struct Dmenu {
    name: &'static str,
}

impl Dmenu {
    pub fn new() -> Self {
        Self { name: "dmenu" }
    }
}

impl Provider for Dmenu {
    fn get_name(&self) -> &str {
        self.name
    }

    fn get_item_layout(&self) -> String {
        include_str!("../../resources/themes/default/item_dmenu.xml").to_string()
    }
}
