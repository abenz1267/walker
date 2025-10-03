use crate::providers::Provider;

#[derive(Debug)]
pub struct Providerlist {
    name: &'static str,
}

impl Providerlist {
    pub fn new() -> Self {
        Self {
            name: "providerlist",
        }
    }
}

impl Provider for Providerlist {
    fn get_name(&self) -> &str {
        self.name
    }

    fn get_item_layout(&self) -> String {
        include_str!("../../resources/themes/default/item_providerlist.xml").to_string()
    }
}
