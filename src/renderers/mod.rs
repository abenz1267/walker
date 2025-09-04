use crate::protos::generated_proto::query::query_response::Item;
use crate::providers::PROVIDERS;
use crate::state::get_dmenu_current;
use crate::theme::Theme;
use crate::ui::window::{quit, with_window};
use gtk4::gdk::ContentProvider;
use gtk4::gio::File;
use gtk4::gio::prelude::FileExt;
use gtk4::prelude::{ListItemExt, WidgetExt};
use gtk4::{Box, Builder, DragSource, Label, ListItem, glib};
use std::path::Path;

pub fn create_item(list_item: &ListItem, item: &Item, theme: &Theme) {
    let b = Builder::new();

    if let Some(s) = theme.items.get(&item.provider) {
        let _ = b.add_from_string(s);
    } else {
        let _ = b.add_from_string(theme.items.get("default").unwrap());
    }

    let itembox: Box = b.object("ItemBox").expect("failed to get ItemBox");
    itembox.add_css_class(&item.provider.replace("menus:", "menus-"));

    item.state.iter().for_each(|i| itembox.add_css_class(i));

    if get_dmenu_current() != 0 && get_dmenu_current() as u32 == list_item.position() + 1 {
        itembox.add_css_class("current");
    }

    list_item.set_child(Some(&itembox));

    if Path::new(&item.text).is_absolute() {
        itembox.add_controller(create_drag_source(&item.text));
    }

    let p = PROVIDERS.get().unwrap().get(&item.provider).unwrap();

    if let Some(text) = b.object::<Label>("ItemText") {
        p.text_transformer(&item.text, &text);
    }

    if let Some(text) = b.object::<Label>("ItemSubtext") {
        p.subtext_transformer(&item.subtext, &text);
    }

    p.image_transformer(&b, &list_item, &item);
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
        with_window(|w| w.window.set_visible(false));
    });

    drag_source.connect_drag_end(|_, _, _| {
        with_window(|w| quit(&w.app, false));
    });

    drag_source
}
