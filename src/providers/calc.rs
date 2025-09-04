use std::path::Path;

use gtk4::{Builder, Image, ListItem, prelude::WidgetExt};

use crate::{
    config::{Elephant, get_config},
    keybinds::Keybind,
    protos::generated_proto::query::query_response::Item,
    providers::Provider,
};

#[derive(Debug)]
pub struct Calc {
    keybinds: Vec<Keybind>,
    default_action: String,
}

impl Calc {
    pub fn new() -> Self {
        let config = get_config();

        Self {
            default_action: config.providers.calc.default.clone(),
            keybinds: vec![
                Keybind {
                    bind: config.providers.calc.copy.clone(),
                    action: "copy".to_string(),
                    after: crate::keybinds::AfterAction::Close,
                },
                Keybind {
                    bind: config.providers.calc.delete.clone(),
                    action: "delete".to_string(),
                    after: crate::keybinds::AfterAction::Reload,
                },
                Keybind {
                    bind: config.providers.calc.save.clone(),
                    action: "save".to_string(),
                    after: crate::keybinds::AfterAction::Reload,
                },
            ],
        }
    }
}

impl Provider for Calc {
    fn get_keybinds(&self) -> &Vec<Keybind> {
        &self.keybinds
    }

    fn default_action(&self) -> &str {
        &self.default_action
    }

    fn get_keybind_hint(&self, cfg: &Elephant) -> String {
        format!(
            "copy: {} - save: {} - delete: {}",
            cfg.providers.calc.copy, cfg.providers.calc.save, cfg.providers.calc.delete
        )
    }

    fn get_default_item_layout(&self) -> &'static str {
        include_str!("../../resources/themes/default/item_calc.xml")
    }

    fn image_transformer(&self, b: &Builder, _: &ListItem, item: &Item) {
        let Some(image) = b.object::<Image>("ItemImage") else {
            return;
        };

        if !item.state.contains(&"current".to_string()) {
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
