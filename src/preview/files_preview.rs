use super::PreviewHandler;
use crate::protos::generated_proto::query::query_response::Item;
use crate::ui::window::get_selected_item;
use crate::{quit, with_window};
use gtk4::gdk::ContentProvider;
use gtk4::gio::File;
use gtk4::glib::clone::Downgrade;
use gtk4::glib::{self, Bytes};
use gtk4::{
    Box as GtkBox, Builder, ContentFit, DragSource, Image, Orientation, Picture, PolicyType,
    ScrolledWindow, Stack, TextView, WrapMode,
};
use gtk4::{gio, prelude::*};
use poppler::{Document, Page};
use std::cell::RefCell;
use std::fs;
use std::path::Path;

#[derive(Debug)]
pub struct FilesPreviewHandler {
    cached_preview: RefCell<Option<FilePreview>>,
}

impl FilesPreviewHandler {
    pub fn new() -> Self {
        Self {
            cached_preview: RefCell::new(None),
        }
    }
}

impl PreviewHandler for FilesPreviewHandler {
    fn handle(&self, item: &Item, preview: &GtkBox, builder: &Builder) {
        let preview_clone = preview.clone();
        let builder_clone = builder.clone();
        let file_path = item.text.clone();

        let item_clone = item.clone();

        if !Path::new(&file_path).exists() {
            return;
        }

        if let Some(current) = get_selected_item() {
            if current != item_clone {
                return;
            }
        } else {
            return;
        }

        let mut cached_preview = self.cached_preview.borrow_mut();
        if cached_preview.is_none() {
            match FilePreview::new_with_builder(&builder_clone).or_else(|_| FilePreview::new()) {
                Ok(preview) => {
                    *cached_preview = Some(preview);
                }
                Err(_e) => {
                    return;
                }
            }
        }

        let file_preview = cached_preview.as_mut().unwrap();
        if let Err(_e) = file_preview.preview_file(&file_path) {
            return;
        }

        while let Some(child) = preview_clone.first_child() {
            child.unparent();
        }

        let existing_controllers: Vec<_> = preview_clone
            .observe_controllers()
            .into_iter()
            .filter_map(|result| result.ok())
            .collect();
        for controller in existing_controllers {
            if let Ok(drag_source) = controller.downcast::<DragSource>() {
                preview_clone.remove_controller(&drag_source);
            }
        }

        let drag_source = DragSource::new();

        let path_copy = file_path.to_string();
        drag_source.connect_prepare(move |_, _, _| {
            let file = File::for_path(&path_copy);
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

        file_preview.box_widget.set_can_target(false);
        preview_clone.add_controller(drag_source);

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
    }
}

#[derive(Debug)]
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

        self.clear_preview();

        let mime_type = self.detect_mime_type(file_path)?;

        let result = if mime_type.starts_with("image/") {
            self.preview_image(file_path)
        } else if mime_type == "application/pdf" {
            self.preview_pdf(file_path)
        } else if mime_type.starts_with("text/") {
            self.preview_text(file_path)
        } else {
            self.preview_generic(file_path)
        };

        result
    }

    fn clear_preview(&self) {
        while let Some(child) = self.preview_area.first_child() {
            self.preview_area.remove(&child);
        }
    }

    fn detect_mime_type(&self, file_path: &str) -> Result<String, Box<dyn std::error::Error>> {
        let path = Path::new(file_path);
        if let Some(extension) = path.extension() {
            if let Some(ext_str) = extension.to_str() {
                let mime_type = match ext_str {
                    "pdf" => Some("application/pdf"),
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

    fn preview_pdf(&self, file_path: &str) -> Result<(), Box<dyn std::error::Error>> {
        let uri = format!("file://{}", file_path);
        let document =
            Document::from_file(&uri, None).map_err(|e| format!("Failed to load PDF: {}", e))?;

        let pdf = GtkBox::new(Orientation::Vertical, 0);

        if let Some(page) = document.page(0) {
            match self.render_pdf_page(&page) {
                Ok(page_widget) => {
                    pdf.append(&page_widget);
                }
                Err(e) => {
                    eprintln!("Failed to render PDF page: {}", e);
                }
            }
        }

        std::mem::drop(document);

        let scrolled = ScrolledWindow::new();
        scrolled.set_child(Some(&pdf));
        scrolled.set_policy(PolicyType::Never, PolicyType::Automatic);
        scrolled.set_width_request(800);

        self.preview_area.add_child(&scrolled);
        self.preview_area.set_visible_child(&scrolled);

        Ok(())
    }

    fn render_pdf_page(&self, page: &Page) -> Result<GtkBox, Box<dyn std::error::Error>> {
        let page_container = GtkBox::new(Orientation::Vertical, 5);
        page_container.set_halign(gtk4::Align::Fill);
        page_container.set_hexpand(true);

        let (width, height) = page.size();

        let target_width = 800.0;
        let display_scale = target_width / width;

        let render_scale = display_scale * 2.0;
        let render_width = (width * render_scale) as i32;
        let render_height = (height * render_scale) as i32;

        let mut surface =
            cairo::ImageSurface::create(cairo::Format::ARgb32, render_width, render_height)?;

        {
            let ctx = cairo::Context::new(&surface)?;

            ctx.set_antialias(cairo::Antialias::Best);
            ctx.scale(render_scale, render_scale);

            ctx.set_source_rgb(1.0, 1.0, 1.0);
            ctx.paint()?;

            page.render(&ctx);

            ctx.target().flush();
            std::mem::drop(ctx);
        }

        surface.flush();

        let bytes = {
            let surface_data = surface.data()?;
            let mut rgba_data = Vec::with_capacity(surface_data.len());

            for chunk in surface_data.chunks_exact(4) {
                rgba_data.push(chunk[2]); // R
                rgba_data.push(chunk[1]); // G
                rgba_data.push(chunk[0]); // B
                rgba_data.push(chunk[3]); // A
            }

            Bytes::from(&rgba_data)
        };

        std::mem::drop(surface);

        let texture = gtk4::gdk::MemoryTexture::new(
            render_width,
            render_height,
            gtk4::gdk::MemoryFormat::R8g8b8a8,
            &bytes,
            (render_width * 4) as usize,
        );

        let picture = Picture::for_paintable(&texture);
        picture.set_content_fit(ContentFit::Cover);
        picture.set_halign(gtk4::Align::Start);
        picture.set_valign(gtk4::Align::Start);
        picture.set_width_request(800);

        let display_height = (height * display_scale) as i32;
        picture.set_height_request(display_height);

        page_container.append(&picture);

        Ok(page_container)
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

        let info = file.query_info(
            "standard::icon",
            gio::FileQueryInfoFlags::NONE,
            gio::Cancellable::NONE,
        );

        if let Ok(info) = info {
            if let Some(image) = icon_weak.upgrade() {
                if let Some(icon) = info.icon() {
                    image.set_from_gicon(&icon);
                }
            }
        }

        container.append(&icon);

        self.preview_area.add_child(&container);
        self.preview_area.set_visible_child(&container);

        Ok(())
    }
}
