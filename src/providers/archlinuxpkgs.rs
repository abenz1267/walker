use crate::providers::Provider;

#[derive(Debug)]
pub struct ArchLinuxPkgs {
    name: &'static str,
}

impl ArchLinuxPkgs {
    pub fn new() -> Self {
        Self {
            name: "archlinuxpkgs",
        }
    }
}

impl Provider for ArchLinuxPkgs {
    fn get_name(&self) -> &str {
        self.name
    }

    fn get_item_layout(&self) -> String {
        include_str!("../../resources/themes/default/item_archlinuxpkgs.xml").to_string()
    }
}
