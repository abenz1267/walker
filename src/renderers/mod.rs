use crate::protos::generated_proto::query::query_response::Type;
use crate::ui::window::{quit, with_window};
use crate::{config::get_config, protos::generated_proto::query::query_response::Item};
use chrono::DateTime;
use gtk4::gdk::ContentProvider;
use gtk4::gio::File;
use gtk4::gio::prelude::FileExt;
use gtk4::glib::clone::Downgrade;
use gtk4::prelude::{ListItemExt, WidgetExt};
use gtk4::{Box, Builder, DragSource, Image, Label, ListItem, Picture, gio, glib};
use std::{env, path::Path};

pub fn create_desktopappications_item(l: &ListItem, i: &Item) {
    let b = Builder::new();
    let _ = b.add_from_string(include_str!("../../resources/themes/default/item.xml"));
    let itembox: Box = b.object("ItemBox").expect("failed to get ItemRoot");
    itembox.add_css_class(&i.provider);
    l.set_child(Some(&itembox));

    if let Some(text) = b.object::<Label>("ItemText") {
        text.set_label(&i.text);
    }

    if let Some(text) = b.object::<Label>("ItemSubtext") {
        if i.subtext.is_empty() {
            text.set_visible(false);
        } else {
            text.set_label(&i.subtext);
        }
    }

    if let Some(image) = b.object::<Image>("ItemImage") {
        if !i.icon.is_empty() {
            if Path::new(&i.icon).is_absolute() {
                image.set_from_file(Some(&i.icon));
            } else {
                image.set_icon_name(Some(&i.icon));
            }
        }
    }
}

pub fn create_clipboard_item(l: &ListItem, i: &Item) {
    let b = Builder::new();
    let _ = b.add_from_string(include_str!(
        "../../resources/themes/default/item_clipboard.xml"
    ));
    let itembox: Box = b.object("ItemBox").expect("failed to get ItemRoot");
    itembox.add_css_class(&i.provider);
    l.set_child(Some(&itembox));

    if let Some(text) = b.object::<Label>("ItemText") {
        text.set_label(&i.text.trim());
    }

    if let Some(text) = b.object::<Label>("ItemSubtext") {
        match DateTime::parse_from_rfc2822(&i.subtext) {
            Ok(dt) => {
                let formatted = dt
                    .format(&get_config().providers.clipboard.time_format)
                    .to_string();
                text.set_label(&formatted);
            }
            Err(_) => {
                text.set_label(&i.subtext);
            }
        }
    }

    if let Some(image) = b.object::<Picture>("ItemImage") {
        match i.type_.enum_value() {
            Ok(Type::FILE) => {
                image.set_filename(Some(&i.text));

                if let Some(text) = b.object::<Label>("ItemText") {
                    text.set_visible(false);
                }
            }
            Ok(Type::REGULAR) => {
                image.set_visible(false);
            }
            Err(_) => {
                println!("Unknown type!");
            }
        }
    }
}

pub fn create_symbols_item(l: &ListItem, i: &Item) {
    let b = Builder::new();
    let _ = b.add_from_string(include_str!(
        "../../resources/themes/default/item_symbols.xml"
    ));
    let itembox: Box = b.object("ItemBox").expect("failed to get ItemRoot");
    itembox.add_css_class(&i.provider);
    l.set_child(Some(&itembox));

    if let Some(text) = b.object::<Label>("ItemText") {
        text.set_label(&i.subtext);
    }

    if let Some(text) = b.object::<Label>("ItemSubtext") {
        text.set_label(&i.subtext);
    }

    if let Some(image) = b.object::<Label>("ItemImage") {
        if !i.text.is_empty() {
            image.set_label(&i.text);
        }
    }
}

pub fn create_calc_item(l: &ListItem, i: &Item) {
    let b = Builder::new();
    let _ = b.add_from_string(include_str!("../../resources/themes/default/item_calc.xml"));
    let itembox: Box = b.object("ItemBox").expect("failed to get ItemRoot");
    itembox.add_css_class(&i.provider);
    l.set_child(Some(&itembox));

    if let Some(text) = b.object::<Label>("ItemText") {
        text.set_label(&i.text);
    }

    if let Some(text) = b.object::<Label>("ItemSubtext") {
        text.set_label(&i.subtext);
    }

    if let Some(image) = b.object::<Image>("ItemImage") {
        if l.position() == 0 {
            if !i.icon.is_empty() {
                if Path::new(&i.icon).is_absolute() {
                    image.set_from_file(Some(&i.icon));
                } else {
                    image.set_icon_name(Some(&i.icon));
                }
            }
        } else {
            image.set_visible(false);
        }
    }
}

pub fn create_files_item(l: &ListItem, i: &Item) {
    let b = Builder::new();
    let _ = b.add_from_string(include_str!(
        "../../resources/themes/default/item_files.xml"
    ));
    let itembox: Box = b.object("ItemBox").expect("failed to get ItemBox");
    itembox.add_css_class(&i.provider);
    l.set_child(Some(&itembox));

    let text = i.text.clone();

    let drag_source = DragSource::new();

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
            quit(&w.app);
        });
    });

    itembox.add_controller(drag_source);

    if let Some(text) = b.object::<Label>("ItemText") {
        if let Ok(home) = env::var("HOME") {
            if let Some(stripped) = i.text.strip_prefix(&home) {
                text.set_label(stripped);
            }
        }
    }

    if let Some(image) = b.object::<Image>("ItemImage") {
        let file = gio::File::for_path(&i.text);
        let image_weak = Downgrade::downgrade(&image);

        file.query_info_async(
            "standard::icon",
            gio::FileQueryInfoFlags::NONE,
            glib::Priority::DEFAULT,
            gio::Cancellable::NONE,
            move |result| {
                if let Some(image) = image_weak.upgrade() {
                    match result {
                        Ok(info) => {
                            if let Some(icon) = info.icon() {
                                image.set_from_gicon(&icon);
                            }
                        }
                        Err(_) => {}
                    }
                }
            },
        );
    }
}

pub fn create_providerlist_item(l: &ListItem, i: &Item) {
    let b = Builder::new();
    let _ = b.add_from_string(include_str!(
        "../../resources/themes/default/item_providerlist.xml"
    ));
    let itembox: Box = b.object("ItemBox").expect("failed to get ItemRoot");
    itembox.add_css_class(&i.provider);
    l.set_child(Some(&itembox));

    if let Some(text) = b.object::<Label>("ItemText") {
        text.set_label(&i.text);
    }

    if let Some(image) = b.object::<Image>("ItemImage") {
        if !i.icon.is_empty() {
            if Path::new(&i.icon).is_absolute() {
                image.set_from_file(Some(&i.icon));
            } else {
                image.set_icon_name(Some(&i.icon));
            }
        }
    }
}
