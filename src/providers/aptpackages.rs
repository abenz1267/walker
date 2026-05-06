use crate::providers::Provider;

#[derive(Debug)]
pub struct AptPackages {
    name: &'static str,
}

impl AptPackages {
    pub fn new() -> Self {
        Self {
            name: "aptpackages",
        }
    }
}

impl Provider for AptPackages {
    fn get_name(&self) -> &str {
        self.name
    }

    fn get_item_layout(&self) -> String {
        include_str!("../../resources/themes/default/item_aptpackages.xml").to_string()
    }
}
