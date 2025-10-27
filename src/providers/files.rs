use std::env;

use gtk4::{
    Builder, Image, ListItem,
    gio::{self, prelude::FileExt},
};

use crate::{protos::generated_proto::query::query_response::Item, providers::Provider};

#[derive(Debug)]
pub struct Files {
    name: &'static str,
}

impl Files {
    pub fn new() -> Self {
        Self { name: "files" }
    }
}

impl Provider for Files {
    fn get_name(&self) -> &str {
        self.name
    }

    fn get_item_layout(&self) -> String {
        include_str!("../../resources/themes/default/item_files.xml").to_string()
    }

    fn text_transformer(&self, item: &Item, label: &gtk4::Label) {
        if let Ok(home) = env::var("HOME")
            && let Some(stripped) = item.text.strip_prefix(&home)
        {
            label.set_label(stripped);
        }
    }

    fn image_transformer(&self, b: &Builder, _: &ListItem, item: &Item) {
        let Some(image) = b.object::<Image>("ItemImage") else {
            return;
        };

        let file = gio::File::for_path(&item.text);

        let info = file.query_info(
            "standard::icon",
            gio::FileQueryInfoFlags::NONE,
            gio::Cancellable::NONE,
        );

        if let Ok(info) = info
            && let Some(icon) = info.icon()
        {
            image.set_from_gicon(&icon);
        }
    }
}
