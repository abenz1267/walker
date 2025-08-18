use super::{LatestOnlyThrottler, PreviewHandler};
use crate::protos::generated_proto::query::query_response::Item;
use crate::{get_selected_item, quit, with_app, with_windows};
use gtk4::gdk::ContentProvider;
use gtk4::gio::File;
use gtk4::glib::clone::Downgrade;
use gtk4::glib::{self};
use gtk4::{
    Box as GtkBox, Builder, ContentFit, DragSource, Image, Orientation, Picture, PolicyType,
    ScrolledWindow, Stack, TextView, WrapMode,
};
use gtk4::{gio, prelude::*};
use std::fs;
use std::path::Path;
use std::rc::Rc;
use std::time::Duration;

#[derive(Debug)]
pub struct FilesPreviewHandler {
    throttler: Rc<LatestOnlyThrottler>,
}

impl FilesPreviewHandler {
    pub fn new() -> Self {
        Self {
            throttler: Rc::new(LatestOnlyThrottler::new(Duration::from_millis(5))),
        }
    }
}

impl PreviewHandler for FilesPreviewHandler {
    fn handle(&self, item: &Item, preview: &GtkBox, builder: &Builder) {
        let preview_clone = preview.clone();
        let builder_clone = builder.clone();
        let file_path = item.text.clone();

        let item_clone = item.clone();

        self.throttler.execute(&file_path, move |path| {
            if !Path::new(path).exists() {
                return;
            }

            if let Some(current) = get_selected_item() {
                if current != item_clone {
                    return;
                }
            } else {
                return;
            }

            let mut file_preview = match FilePreview::new_with_builder(&builder_clone)
                .or_else(|_| FilePreview::new())
            {
                Ok(preview) => preview,
                Err(_e) => {
                    return;
                }
            };

            if let Err(_e) = file_preview.preview_file(path) {
                return;
            }

            while let Some(child) = preview_clone.first_child() {
                child.unparent();
            }

            let drag_source = DragSource::new();

            let path_copy = path.to_string();
            drag_source.connect_prepare(move |_, _, _| {
                let file = File::for_path(&path_copy);
                let uri_string = format!("{}\n", file.uri());
                let b = glib::Bytes::from(uri_string.as_bytes());
                let cp = ContentProvider::for_bytes("text/uri-list", &b);
                Some(cp)
            });

            drag_source.connect_drag_begin(|_, _| {
                with_windows(|w| {
                    w[0].set_visible(false);
                });
            });

            drag_source.connect_drag_end(|_, _, _| {
                with_app(|app| {
                    quit(app);
                });
            });

            file_preview.box_widget.add_controller(drag_source);

            preview_clone.append(&file_preview.box_widget);

            if let Some(current) = get_selected_item() {
                if current == item_clone {
                    preview_clone.set_visible(true);
                } else {
                    preview_clone.set_visible(false);
                }
            } else {
                preview_clone.set_visible(false);
            }
        });
    }
}

pub struct FilePreview {
    pub box_widget: GtkBox,
    preview_area: Stack,
    current_file: String,
}

impl FilePreview {
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
            current_file: String::new(),
        })
    }

    pub fn new() -> Result<Self, Box<dyn std::error::Error>> {
        let box_widget = GtkBox::new(Orientation::Vertical, 0);
        let preview_area = Stack::new();

        box_widget.append(&preview_area);

        Ok(Self {
            box_widget,
            preview_area,
            current_file: String::new(),
        })
    }

    pub fn preview_file(&mut self, file_path: &str) -> Result<(), Box<dyn std::error::Error>> {
        self.current_file = file_path.to_string();

        while let Some(child) = self.preview_area.first_child() {
            child.unparent();
        }

        let mime_type = self.detect_mime_type(file_path)?;

        let result = if mime_type.starts_with("image/") {
            self.preview_image(file_path)
        } else if mime_type.starts_with("text/") {
            self.preview_text(file_path)
        } else {
            self.preview_generic(file_path)
        };

        result
    }

    fn detect_mime_type(&self, file_path: &str) -> Result<String, Box<dyn std::error::Error>> {
        let path = Path::new(file_path);
        if let Some(extension) = path.extension() {
            if let Some(ext_str) = extension.to_str() {
                let mime_type = match ext_str {
                    "go" => Some("text/x-go"),
                    "rs" => Some("text/x-rust"),
                    "py" => Some("text/x-python"),
                    "js" => Some("text/javascript"),
                    "html" => Some("text/html"),
                    "css" => Some("text/css"),
                    "json" => Some("text/json"),
                    "xml" => Some("text/xml"),
                    "md" => Some("text/markdown"),
                    "txt" => Some("text/plain"),
                    _ => None,
                };

                if let Some(mime) = mime_type {
                    return Ok(mime.to_string());
                }

                if let Some(mime_type) = mime_guess::from_ext(ext_str).first() {
                    return Ok(mime_type.to_string());
                }
            }
        }

        Ok("application/octet-stream".to_string())
    }

    fn preview_image(&self, file_path: &str) -> Result<(), Box<dyn std::error::Error>> {
        let picture = Picture::new();
        picture.set_filename(Some(file_path));
        picture.set_content_fit(ContentFit::Contain);

        let scrolled = ScrolledWindow::new();
        scrolled.set_child(Some(&picture));
        scrolled.set_policy(PolicyType::Automatic, PolicyType::Automatic);

        self.preview_area.add_child(&scrolled);
        self.preview_area.set_visible_child(&scrolled);

        Ok(())
    }

    fn preview_text(&self, file_path: &str) -> Result<(), Box<dyn std::error::Error>> {
        let content = fs::read(file_path)?;

        let max_size = 1024 * 1024; // 1MB
        let display_content = if content.len() > max_size {
            let mut truncated = content[..max_size].to_vec();
            truncated.extend_from_slice(b"\n\n[File truncated...]");
            truncated
        } else {
            content
        };

        let text_view = TextView::new();
        text_view.set_editable(false);
        text_view.set_monospace(true);
        text_view.set_wrap_mode(WrapMode::Word);
        text_view.set_size_request(300, 200);

        let buffer = text_view.buffer();
        if let Ok(content_str) = String::from_utf8(display_content) {
            buffer.set_text(&content_str);
        } else {
            buffer.set_text("[Binary file - cannot display as text]");
        }

        let scrolled = ScrolledWindow::new();
        scrolled.set_child(Some(&text_view));
        scrolled.set_policy(PolicyType::Automatic, PolicyType::Automatic);
        scrolled.set_size_request(300, 250);

        self.preview_area.add_child(&scrolled);
        self.preview_area.set_visible_child(&scrolled);

        Ok(())
    }

    fn preview_generic(&self, file_path: &str) -> Result<(), Box<dyn std::error::Error>> {
        let container = GtkBox::new(Orientation::Vertical, 10);
        container.set_halign(gtk4::Align::Center);
        container.set_valign(gtk4::Align::Center);
        container.set_margin_top(20);
        container.set_margin_bottom(20);
        container.set_margin_start(20);
        container.set_margin_end(20);
        container.set_size_request(250, 200);

        let file = gio::File::for_path(file_path);
        let icon = Image::from_icon_name("text-x-generic");
        icon.set_icon_size(gtk4::IconSize::Large);
        let icon_weak = Downgrade::downgrade(&icon);

        file.query_info_async(
            "standard::icon",
            gio::FileQueryInfoFlags::NONE,
            glib::Priority::DEFAULT,
            gio::Cancellable::NONE,
            move |result| {
                if let Some(image) = icon_weak.upgrade() {
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

        container.append(&icon);

        self.preview_area.add_child(&container);
        self.preview_area.set_visible_child(&container);

        Ok(())
    }
}
