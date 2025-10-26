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

    fn text_transformer(&self, item: &Item, label: &gtk4::Label) {
        if !item.icon.is_empty() {
            let Ok(dt) = DateTime::parse_from_rfc2822(&item.subtext) else {
                label.set_label(&item.subtext);
                return;
            };

            let formatted = dt
                .format(&get_config().providers.clipboard.time_format)
                .to_string();
            label.set_label(&formatted);

            return;
        }

        label.set_label(item.text.trim());
    }

    fn subtext_transformer(&self, item: &Item, label: &gtk4::Label) {
        if !item.icon.is_empty() {
            label.set_label("Image");
            return;
        }

        let Ok(dt) = DateTime::parse_from_rfc2822(&item.subtext) else {
            label.set_label(&item.subtext);
            return;
        };

        let formatted = dt
            .format(&get_config().providers.clipboard.time_format)
            .to_string();
        label.set_label(&formatted);
    }
}
