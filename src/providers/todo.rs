use std::path::Path;

use gtk4::{Builder, Image, ListItem, prelude::WidgetExt};

use crate::{protos::generated_proto::query::query_response::Item, providers::Provider};

#[derive(Debug)]
pub struct Todo {
    name: &'static str,
}

impl Todo {
    pub fn new() -> Self {
        Self { name: "todo" }
    }
}

impl Provider for Todo {
    fn get_name(&self) -> &str {
        self.name
    }

    fn get_item_layout(&self) -> String {
        include_str!("../../resources/themes/default/item_todo.xml").to_string()
    }

    fn image_transformer(&self, b: &Builder, _: &ListItem, item: &Item) {
        let Some(image) = b.object::<Image>("ItemImage") else {
            return;
        };

        if !item.state.contains(&"creating".to_string()) {
            image.set_visible(false);
            return;
        }

        let function = if !item.icon.is_empty() && Path::new(&item.icon).is_absolute() {
            Image::set_from_file
        } else if !item.icon.is_empty() {
            Image::set_icon_name
        } else {
            return;
        };

        function(&image, Some(&item.icon))
    }
}
