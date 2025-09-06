use chrono::DateTime;
use gtk4::{
    Label, Picture, gdk,
    gio::{self, prelude::FileExtManual},
    glib,
    prelude::WidgetExt,
};

use crate::{
    config::{Elephant, get_config},
    keybinds::{Action, AfterAction, Keybind},
    providers::Provider,
};

#[derive(Debug)]
pub struct Clipboard {
    item_keybinds: Vec<Keybind>,
    global_keybinds: Vec<Keybind>,
    default_action: String,
}

impl Clipboard {
    pub fn new() -> Self {
        let config = get_config();

        Self {
            default_action: config.providers.clipboard.default.clone(),
            global_keybinds: vec![Keybind {
                bind: config.providers.clipboard.toggle_images_only.clone(),
                action: Action {
                    label: "toggle images",
                    required_states: None,
                    action: "toggle_images".to_string(),
                    after: AfterAction::ClearReload,
                },
            }],
            item_keybinds: vec![
                Keybind {
                    bind: config.providers.clipboard.copy.clone(),
                    action: Action {
                        label: "copy",
                        required_states: None,
                        action: "copy".to_string(),
                        after: AfterAction::Close,
                    },
                },
                Keybind {
                    bind: config.providers.clipboard.delete.clone(),
                    action: Action {
                        label: "remove",
                        required_states: None,
                        action: "remove".to_string(),
                        after: AfterAction::ClearReload,
                    },
                },
                Keybind {
                    bind: config.providers.clipboard.edit.clone(),
                    action: Action {
                        label: "edit",
                        required_states: None,
                        action: "edit".to_string(),
                        after: AfterAction::Close,
                    },
                },
            ],
        }
    }
}

impl Provider for Clipboard {
    fn get_keybinds(&self) -> &Vec<Keybind> {
        &self.item_keybinds
    }

    fn get_global_keybinds(&self) -> Option<&Vec<Keybind>> {
        Some(&self.global_keybinds)
    }

    fn default_action(&self) -> &str {
        &self.default_action
    }

    fn get_item_layout(&self) -> String {
        include_str!("../../resources/themes/default/item_clipboard.xml").to_string()
    }

    fn text_transformer(&self, text: &str, label: &gtk4::Label) {
        label.set_label(&text.trim());
    }

    fn subtext_transformer(&self, text: &str, label: &gtk4::Label) {
        let Ok(dt) = DateTime::parse_from_rfc2822(&text) else {
            label.set_label(&text);
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
