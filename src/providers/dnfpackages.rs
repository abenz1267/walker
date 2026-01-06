use crate::providers::Provider;

#[derive(Debug)]
pub struct DnfPackages {
    name: &'static str,
}

impl DnfPackages {
    pub fn new() -> Self {
        Self {
            name: "dnfpackages",
        }
    }
}

impl Provider for DnfPackages {
    fn get_name(&self) -> &str {
        self.name
    }

    fn get_item_layout(&self) -> String {
        include_str!("../../resources/themes/default/item_dnfpackages.xml").to_string()
    }
}
