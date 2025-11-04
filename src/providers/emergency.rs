use crate::providers::Provider;

#[derive(Debug)]
pub struct Emergency {
    name: &'static str,
}

impl Emergency {
    pub fn new() -> Self {
        Self { name: "emergency" }
    }
}

impl Provider for Emergency {
    fn get_name(&self) -> &str {
        self.name
    }

    fn get_item_layout(&self) -> String {
        include_str!("../../resources/themes/default/item_dmenu.xml").to_string()
    }
}
