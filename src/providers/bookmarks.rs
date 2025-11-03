use std::path::Path;

use gtk4::{Builder, Image, Label, ListItem, prelude::WidgetExt};

use crate::{
    protos::generated_proto::query::query_response::Item,
    providers::{Provider, shared_image_transformer},
};

#[derive(Debug)]
pub struct Bookmarks {
    name: &'static str,
}

impl Bookmarks {
    pub fn new() -> Self {
        Self { name: "bookmarks" }
    }
}

impl Provider for Bookmarks {
    fn get_name(&self) -> &str {
        self.name
    }

    fn get_item_layout(&self) -> String {
        include_str!("../../resources/themes/default/item_bookmarks.xml").to_string()
    }

    fn image_transformer(&self, b: &Builder, i: &ListItem, item: &Item) {
        if item.state.contains(&"creating".to_string()) {
            let Some(image) = b.object::<Image>("ItemImageCreate") else {
                return;
            };

            if let Some(image) = b.object::<Image>("ItemImage") {
                image.set_visible(false);
            };

            if let Some(image) = b.object::<Label>("ItemImageFont") {
                image.set_visible(false);
            };

            let function = if !item.icon.is_empty() && Path::new(&item.icon).is_absolute() {
                Image::set_from_file
            } else if !item.icon.is_empty() {
                Image::set_icon_name
            } else {
                return;
            };

            function(&image, Some(&item.icon));
        } else {
            if let Some(image) = b.object::<Image>("ItemImageCreate") {
                image.set_visible(false);
            };

            shared_image_transformer(b, i, item);
        }
    }
}
