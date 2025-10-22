use crate::{
    config::get_config, protos::generated_proto::query::query_response::Item, providers::Provider,
};

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

    fn subtext_transformer(&self, item: &Item, label: &gtk4::Label) {
        let cfg = get_config();

        if let Some(prefix) = cfg
            .providers
            .prefixes
            .iter()
            .find(|p| p.provider == item.identifier)
        {
            label.set_text(format!("( {} )", &prefix.prefix).as_str());
        }
    }

    fn get_item_layout(&self) -> String {
        include_str!("../../resources/themes/default/item_providerlist.xml").to_string()
    }
}
