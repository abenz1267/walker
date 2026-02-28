use crate::providers::Provider;

#[derive(Debug)]
pub struct Calc {
    name: &'static str,
}

impl Calc {
    pub fn new() -> Self {
        Self { name: "calc" }
    }
}

impl Provider for Calc {
    fn get_name(&self) -> &str {
        self.name
    }

    fn get_item_layout(&self) -> String {
        include_str!("../../resources/themes/default/item_calc.xml").to_string()
    }
}
