use chrono::DateTime;
use chrono_humanize::HumanTime;

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
        if item.preview_type == "file" {
            label.set_label("Image");
            return;
        }

        label.set_label(item.text.trim());
    }

    fn subtext_transformer(&self, item: &Item, label: &gtk4::Label) {
        let Ok(dt) = DateTime::parse_from_rfc2822(&item.subtext) else {
            label.set_label(&item.subtext);
            return;
        };

        let fmt = &get_config().providers.clipboard.time_format;
        let mut text = match fmt.as_str() {
            "relative" => format!("{}", HumanTime::from(dt)),
            _ => dt.format(fmt).to_string(),
        };

        if item.state.contains(&String::from("pinned")) {
            text = format!("{} ✦", text);
        };

        label.set_label(&text);
    }
}
