use chrono::DateTime;
use gtk4::{
    Label, Picture, gdk,
    gio::{self, prelude::FileExtManual},
    glib,
    prelude::WidgetExt,
};

use crate::{
    config::get_config, protos::generated_proto::query::query_response::Item, providers::Provider,
};

#[derive(Debug)]
pub struct Clipboard {
    name: &'static str,
}

impl Clipboard {
    pub fn new() -> Self {
        Self { name: "clipboard" }
    }
}

impl Provider for Clipboard {
    fn get_name(&self) -> &str {
        self.name
    }

    fn get_item_layout(&self) -> String {
        include_str!("../../resources/themes/default/item_clipboard.xml").to_string()
    }

    fn text_transformer(&self, text: &str, label: &gtk4::Label) {
        label.set_label(text.trim());
    }

    fn subtext_transformer(&self, item: &Item, label: &gtk4::Label) {
        let Ok(dt) = DateTime::parse_from_rfc2822(&item.subtext) else {
            label.set_label(&item.subtext);
            return;
        };

        let formatted = dt
            .format(&get_config().providers.clipboard.time_format)
            .to_string();
        label.set_label(&formatted);
    }

    fn image_transformer(
        &self,
        b: &gtk4::Builder,
        _: &gtk4::ListItem,
        item: &crate::protos::generated_proto::query::query_response::Item,
    ) {
        let Some(image) = b.object::<Picture>("ItemImage") else {
            return;
        };

        if item.icon.is_empty() {
            image.set_visible(false);
            return;
        }

        let icon = item.icon.clone();

        glib::spawn_future_local(async move {
            let Ok((bytes, _)) = gio::File::for_path(&icon).load_contents_future().await else {
                return;
            };

            let texture = gdk::Texture::from_bytes(&glib::Bytes::from(&bytes)).unwrap();
            image.set_paintable(Some(&texture));
        });

        if let Some(text) = b.object::<Label>("ItemText") {
            text.set_visible(false);
        }
    }
}
