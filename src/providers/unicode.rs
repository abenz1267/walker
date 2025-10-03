use gtk4::{Builder, Label, ListItem};

use crate::{protos::generated_proto::query::query_response::Item, providers::Provider};

#[derive(Debug)]
pub struct Unicode {
    name: &'static str,
}

impl Unicode {
    pub fn new() -> Self {
        Self { name: "unicode" }
    }
}

impl Provider for Unicode {
    fn get_name(&self) -> &str {
        self.name
    }

    fn get_item_layout(&self) -> String {
        include_str!("../../resources/themes/default/item_unicode.xml").to_string()
    }

    fn image_transformer(&self, b: &Builder, _: &ListItem, item: &Item) {
        if let Some(image) = b.object::<Label>("ItemImage")
            && !item.icon.is_empty()
            && let Ok(code_point) = u32::from_str_radix(&item.icon, 16)
            && let Some(unicode_char) = char::from_u32(code_point)
        {
            image.set_label(&format!("{unicode_char}"));
        }
    }
}
