use std::env;
use std::path::Path;

use gtk4::{
    Builder, Image, Label, ListItem,
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

    fn text_transformer(&self, item: &Item, label: &Label) {
        let text = Path::new(&item.text)
            .file_name()
            .and_then(|f| f.to_str())
            .unwrap();

        label.set_text(text);
    }

    fn subtext_transformer(&self, item: &Item, label: &Label) {
        let subtext = Path::new(&item.text)
            .parent()
            .and_then(|p| p.to_str())
            .map(|parent_folder| {
                if let Ok(home) = env::var("HOME") {
                    if let Some(stripped) = parent_folder.strip_prefix(&home) {
                        return format!("~{}", stripped);
                    }
                }
                parent_folder.to_string()
            })
            .unwrap_or_default();

        label.set_text(&subtext);
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
