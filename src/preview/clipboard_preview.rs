use super::PreviewHandler;
use crate::protos::generated_proto::query::query_response::Item;
use crate::ui::window::get_selected_item;
use gtk4::gio;
use gtk4::glib;
use gtk4::{
    Box as GtkBox, Builder, ContentFit, Image, Orientation, Picture, PolicyType, ScrolledWindow,
    Stack, TextView, WrapMode, gdk_pixbuf, prelude::*,
};
use std::path::Path;

#[derive(Debug)]
pub struct ClipboardPreviewHandler;

#[derive(Debug)]
pub struct ClipboardPreview {
    pub box_widget: GtkBox,
    preview_area: Stack,
    current_item_id: String,
}

impl ClipboardPreviewHandler {
    pub fn new() -> Self {
        Self
    }
}

impl PreviewHandler for ClipboardPreviewHandler {
    fn clear_cache(&self) {}

    fn handle(&self, item: &Item, preview: &GtkBox, builder: &Builder) {
        let Some(current) = get_selected_item() else {
            return;
        };

        if current != *item {
            return;
        }

        let clipboard_preview =
            ClipboardPreview::new_with_builder(builder).or_else(|_| ClipboardPreview::new());

        let Ok(mut clipboard_preview) = clipboard_preview else {
            return;
        };

        if clipboard_preview.preview_item(item).is_err() {
            return;
        }

        while let Some(child) = preview.first_child() {
            child.unparent();
        }
        preview.append(&clipboard_preview.box_widget);
        preview.set_visible(get_selected_item().is_some_and(|current| current == *item));
    }
}

impl ClipboardPreview {
    pub fn new_with_builder(builder: &Builder) -> Result<Self, Box<dyn std::error::Error>> {
        let box_widget = builder
            .object::<GtkBox>("PreviewBox")
            .ok_or("PreviewBox not found in builder")?;

        let preview_area = builder
            .object::<Stack>("PreviewStack")
            .ok_or("PreviewStack not found in builder")?;

        Ok(Self {
            box_widget,
            preview_area,
            current_item_id: String::new(),
        })
    }

    pub fn new() -> Result<Self, Box<dyn std::error::Error>> {
        let box_widget = GtkBox::new(Orientation::Vertical, 0);
        let preview_area = Stack::new();
        box_widget.append(&preview_area);

        Ok(Self {
            box_widget,
            preview_area,
            current_item_id: String::new(),
        })
    }

    pub fn preview_item(&mut self, item: &Item) -> Result<(), Box<dyn std::error::Error>> {
        self.current_item_id = item.identifier.clone();
        self.clear_preview();

        if !item.icon.is_empty() && Path::new(&item.icon).exists() {
            return self.preview_image(&item.icon);
        }

        self.preview_text(&item.text)
    }

    fn clear_preview(&self) {
        while let Some(child) = self.preview_area.first_child() {
            if let Some(picture) = child.downcast_ref::<Picture>() {
                picture.set_filename(Option::<&str>::None);
                picture.set_paintable(gtk4::gdk::Paintable::NONE);
            }

            if let Some(image) = child.downcast_ref::<Image>() {
                image.clear();
                image.set_icon_name(Option::<&str>::None);
            }

            if let Some(scrolled) = child.downcast_ref::<ScrolledWindow>()
                && let Some(scrolled_child) = scrolled.child()
            {
                if let Some(text_view) = scrolled_child.downcast_ref::<TextView>() {
                    text_view.buffer().set_text("");
                }
                if let Some(picture) = scrolled_child.downcast_ref::<Picture>() {
                    picture.set_filename(Option::<&str>::None);
                    picture.set_paintable(gtk4::gdk::Paintable::NONE);
                }
                if let Some(image) = scrolled_child.downcast_ref::<Image>() {
                    image.clear();
                    image.set_icon_name(Option::<&str>::None);
                }
            }

            self.preview_area.remove(&child);
        }
    }

    fn preview_image(&self, image_path: &str) -> Result<(), Box<dyn std::error::Error>> {
        let picture = Picture::new();
        picture.set_content_fit(ContentFit::Contain);

        let scrolled = ScrolledWindow::new();
        scrolled.set_child(Some(&picture));
        scrolled.set_policy(PolicyType::Automatic, PolicyType::Automatic);
        scrolled.set_size_request(400, 300);

        self.preview_area.add_child(&scrolled);
        self.preview_area.set_visible_child(&scrolled);

        let file = gio::File::for_path(image_path);
        picture.set_file(Some(&file));

        let image_path_clone = image_path.to_string();

        glib::MainContext::ref_thread_default().spawn_local(async move {
            let cancellable = gio::Cancellable::new();
            match Self::load_image_async(&image_path_clone, &cancellable).await {
                Ok(_) => {}
                Err(e) => {
                    eprintln!("Failed to cache image {}: {}", image_path_clone, e);
                }
            }
        });

        Ok(())
    }

    async fn load_image_async(
        image_path: &str,
        cancellable: &gio::Cancellable,
    ) -> Result<gtk4::gdk::Texture, Box<dyn std::error::Error>> {
        let file = gio::File::for_path(image_path);

        let (bytes, _) = file.load_bytes_future().await?;
        let stream = gio::MemoryInputStream::from_bytes(&bytes);

        let pixbuf = gdk_pixbuf::Pixbuf::from_stream(&stream, Some(cancellable))?;

        let max_width = 800;
        let max_height = 600;
        let (width, height) = (pixbuf.width(), pixbuf.height());
        let (new_width, new_height) = if width > max_width || height > max_height {
            let width_ratio = max_width as f64 / width as f64;
            let height_ratio = max_height as f64 / height as f64;
            let ratio = width_ratio.min(height_ratio);
            (
                (width as f64 * ratio) as i32,
                (height as f64 * ratio) as i32,
            )
        } else {
            (width, height)
        };

        let scaled_pixbuf = if new_width != width || new_height != height {
            pixbuf
                .scale_simple(new_width, new_height, gdk_pixbuf::InterpType::Bilinear)
                .ok_or("Failed to scale image")?
        } else {
            pixbuf
        };

        let texture = gtk4::gdk::Texture::for_pixbuf(&scaled_pixbuf);

        Ok(texture)
    }

    fn preview_text(&self, text: &str) -> Result<(), Box<dyn std::error::Error>> {
        let text_view = TextView::new();
        text_view.set_editable(false);
        text_view.set_monospace(true);
        text_view.set_wrap_mode(WrapMode::Word);
        text_view.set_size_request(400, 300);

        let buffer = text_view.buffer();

        let display_text = if text.len() > 10000 {
            format!("{}\n\n[Text truncated...]", &text[..10000])
        } else {
            text.to_string()
        };

        buffer.set_text(&display_text);

        let scrolled = ScrolledWindow::new();
        scrolled.set_child(Some(&text_view));
        scrolled.set_policy(PolicyType::Automatic, PolicyType::Automatic);
        scrolled.set_size_request(400, 300);

        self.preview_area.add_child(&scrolled);
        self.preview_area.set_visible_child(&scrolled);

        Ok(())
    }
}
