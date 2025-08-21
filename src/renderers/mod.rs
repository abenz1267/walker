use crate::state::with_state;
use crate::theme::Theme;
use crate::ui::window::{quit, with_window};
use crate::{config::get_config, protos::generated_proto::query::query_response::Item};
use chrono::DateTime;
use gtk4::gdk::ContentProvider;
use gtk4::gio::File;
use gtk4::gio::prelude::FileExt;
use gtk4::prelude::{ListItemExt, WidgetExt};
use gtk4::{Box, Builder, DragSource, Image, Label, ListItem, Picture, gio, glib};
use std::collections::HashMap;
use std::sync::OnceLock;
use std::{env, path::Path};

thread_local! {
    pub static TEXT_TRANSFORMERS: OnceLock<HashMap<String, fn(&str, &Label)>> = OnceLock::new();
    pub static SUBTEXT_TRANSFORMERS: OnceLock<HashMap<String, fn(&str, &Label)>> = OnceLock::new();
    pub static IMAGE_TRANSFORMERS: OnceLock<HashMap<String, fn(&str, &Builder, &ListItem, &Item)>> = OnceLock::new();
}

pub fn with_text_transformers<F, R>(f: F) -> R
where
    F: FnOnce(&HashMap<String, fn(&str, &Label)>) -> R,
{
    TEXT_TRANSFORMERS.with(|t| {
        let data = t.get().expect("Text transformers not initialized");
        f(data)
    })
}

pub fn with_image_transformers<F, R>(f: F) -> R
where
    F: FnOnce(&HashMap<String, fn(&str, &Builder, &ListItem, &Item)>) -> R,
{
    IMAGE_TRANSFORMERS.with(|t| {
        let data = t.get().expect("Image transformers not initialized");
        f(data)
    })
}

pub fn with_subtext_transformers<F, R>(f: F) -> R
where
    F: FnOnce(&HashMap<String, fn(&str, &Label)>) -> R,
{
    SUBTEXT_TRANSFORMERS.with(|t| {
        let data = t.get().expect("Subtext transformers not initialized");
        f(data)
    })
}

pub fn setup_item_transformers() {
    let mut text: HashMap<String, fn(&str, &Label)> = HashMap::new();

    text.insert("default".to_string(), default_text_transformer);
    text.insert("files".to_string(), files_text_transformer);
    text.insert("clipboard".to_string(), clipboard_text_transformer);

    TEXT_TRANSFORMERS.with(|t| {
        t.set(text).expect("Text transformers already initialized");
    });

    let mut subtext: HashMap<String, fn(&str, &Label)> = HashMap::new();

    subtext.insert("default".to_string(), default_subtext_transformer);
    subtext.insert("clipboard".to_string(), clipboard_subtext_transformer);

    SUBTEXT_TRANSFORMERS.with(|t| {
        t.set(subtext)
            .expect("Text transformers already initialized");
    });

    let mut image: HashMap<String, fn(&str, &Builder, &ListItem, &Item)> = HashMap::new();
    image.insert("default".to_string(), default_image_transformer);
    image.insert("clipboard".to_string(), clipboard_image_transformer);
    image.insert("symbols".to_string(), symbols_image_transformer);
    image.insert("calc".to_string(), calc_image_transformer);
    image.insert("files".to_string(), files_image_transformer);

    IMAGE_TRANSFORMERS.with(|t| {
        t.set(image).expect("Text transformers already initialized");
    });
}

fn default_image_transformer(img: &str, b: &Builder, _: &ListItem, _: &Item) {
    if let Some(image) = b.object::<Image>("ItemImage") {
        if !img.is_empty() {
            if Path::new(&img).is_absolute() {
                image.set_from_file(Some(&img));
            } else {
                image.set_icon_name(Some(&img));
            }
        }
    }
}

fn calc_image_transformer(img: &str, b: &Builder, li: &ListItem, _: &Item) {
    if let Some(image) = b.object::<Image>("ItemImage") {
        if li.position() == 0 {
            if !img.is_empty() {
                if Path::new(&img).is_absolute() {
                    image.set_from_file(Some(&img));
                } else {
                    image.set_icon_name(Some(&img));
                }
            }
        } else {
            image.set_visible(false);
        }
    }
}

fn files_image_transformer(_: &str, b: &Builder, _: &ListItem, item: &Item) {
    if let Some(image) = b.object::<Image>("ItemImage") {
        let file = gio::File::for_path(&item.text);

        let info = file.query_info(
            "standard::icon",
            gio::FileQueryInfoFlags::NONE,
            gio::Cancellable::NONE,
        );

        if let Ok(info) = info {
            if let Some(icon) = info.icon() {
                image.set_from_gicon(&icon);
            }
        }
    }
}

fn symbols_image_transformer(img: &str, b: &Builder, _: &ListItem, _: &Item) {
    if let Some(image) = b.object::<Label>("ItemImage") {
        if !img.is_empty() {
            image.set_label(&img);
        }
    }
}

fn clipboard_image_transformer(img: &str, b: &Builder, _: &ListItem, _: &Item) {
    if let Some(image) = b.object::<Picture>("ItemImage") {
        image.set_filename(Option::<&str>::None);

        if !img.is_empty() {
            if Path::new(&img).is_absolute() {
                image.set_filename(Some(&img));
            } else {
                image.set_filename(Some(&img));
            }

            if let Some(text) = b.object::<Label>("ItemText") {
                text.set_visible(false);
            }
        } else {
            image.set_visible(false);
        }
    }
}

fn files_text_transformer(text: &str, label: &Label) {
    if let Ok(home) = env::var("HOME") {
        if let Some(stripped) = text.strip_prefix(&home) {
            label.set_label(stripped);
        }
    }
}

fn clipboard_text_transformer(text: &str, label: &Label) {
    label.set_label(&text.trim());
}

fn default_text_transformer(text: &str, label: &Label) {
    if text.is_empty() {
        label.set_visible(false);
    } else {
        label.set_text(&text);
    }
}

fn default_subtext_transformer(text: &str, label: &Label) {
    if text.is_empty() {
        label.set_visible(false);
    } else {
        label.set_text(&text);
    }
}

fn clipboard_subtext_transformer(text: &str, label: &Label) {
    match DateTime::parse_from_rfc2822(&text) {
        Ok(dt) => {
            let formatted = dt
                .format(&get_config().providers.clipboard.time_format)
                .to_string();
            label.set_label(&formatted);
        }
        Err(_) => {
            label.set_label(&text);
        }
    }
}

pub fn create_item(list_item: &ListItem, item: &Item, theme: &Theme) {
    let b = Builder::new();

    if let Some(s) = theme.items.get(&item.provider) {
        let _ = b.add_from_string(s);
    } else {
        let _ = b.add_from_string(theme.items.get("default").unwrap());
    }

    let itembox: Box = b.object("ItemBox").expect("failed to get ItemBox");
    itembox.add_css_class(&item.provider);

    with_state(|s| {
        if s.get_dmenu_current() != 0 && s.get_dmenu_current() as u32 == list_item.position() + 1 {
            itembox.add_css_class("current");
        }
    });

    list_item.set_child(Some(&itembox));

    if is_absolute_path(&item.text) {
        itembox.add_controller(create_drag_source(&item.text));
    }

    if let Some(text) = b.object::<Label>("ItemText") {
        with_text_transformers(|t| {
            if let Some(t) = t.get(&item.provider) {
                t(&item.text, &text);
            } else {
                t.get("default").unwrap()(&item.text, &text);
            }
        });
    }

    if let Some(text) = b.object::<Label>("ItemSubtext") {
        with_subtext_transformers(|t| {
            if let Some(t) = t.get(&item.provider) {
                t(&item.subtext, &text);
            } else {
                t.get("default").unwrap()(&item.subtext, &text);
            }
        });
    }

    with_image_transformers(|t| {
        if let Some(t) = t.get(&item.provider) {
            t(&item.icon, &b, &list_item, &item);
        } else {
            t.get("default").unwrap()(&item.icon, &b, &list_item, &item);
        }
    });
}

fn is_absolute_path(path: &str) -> bool {
    Path::new(path).is_absolute()
}

pub fn create_drag_source(text: &str) -> DragSource {
    let drag_source = DragSource::new();
    let text = text.to_string();
    drag_source.connect_prepare(move |_, _, _| {
        let file = File::for_path(&text);
        let uri_string = format!("{}\n", file.uri());
        let b = glib::Bytes::from(uri_string.as_bytes());

        let cp = ContentProvider::for_bytes("text/uri-list", &b);

        Some(cp)
    });

    drag_source.connect_drag_begin(|_, _| {
        with_window(|w| {
            w.window.set_visible(false);
        });
    });

    drag_source.connect_drag_end(|_, _, _| {
        with_window(|w| {
            quit(&w.app, false);
        });
    });

    drag_source
}
