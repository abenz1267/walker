use gtk4::{Builder, Label, ListItem};

use crate::{protos::generated_proto::query::query_response::Item, providers::Provider};

#[derive(Debug)]
pub struct Symbols {
    name: &'static str,
}

impl Symbols {
    pub fn new() -> Self {
        Self { name: "symbols" }
    }
}

impl Provider for Symbols {
    fn get_name(&self) -> &str {
        self.name
    }

    fn get_item_layout(&self) -> String {
        include_str!("../../resources/themes/default/item_symbols.xml").to_string()
    }

    fn get_item_grid_layout(&self) -> String {
        include_str!("../../resources/themes/default/item_symbols_grid.xml").to_string()
    }

    fn image_transformer(&self, b: &Builder, _: &ListItem, item: &Item) {
        if let Some(image) = b.object::<Label>("ItemImage")
            && !item.icon.is_empty()
        {
            image.set_label(&item.icon);
        }
    }
}
