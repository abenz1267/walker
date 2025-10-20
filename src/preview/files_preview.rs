use super::PreviewHandler;
use crate::protos::generated_proto::query::query_response::Item;
use crate::ui::window::get_selected_item;
use crate::{quit, with_window};
use gtk4::gdk::ContentProvider;
use gtk4::gio::{self, Cancellable, File};
use gtk4::glib::{self, Bytes};
use gtk4::{
    Box as GtkBox, Builder, ContentFit, DragSource, Image, Orientation, Picture, PolicyType,
    ScrolledWindow, Stack, TextView, Video, WrapMode, gdk_pixbuf, prelude::*,
};
use poppler::{Document, Page};
use std::cell::RefCell;
use std::collections::HashMap;
use std::fs;
use std::path::Path;
use std::rc::Rc;

#[derive(Debug)]
pub struct FilesPreviewHandler {
    cached_preview: RefCell<Option<FilePreview>>,
}

#[derive(Debug)]
pub struct FilePreview {
    pub box_widget: GtkBox,
    preview_area: Stack,
    current_file: String,
    image_cache: Rc<RefCell<HashMap<String, gtk4::gdk::Texture>>>,
    pub current_video_cancellable: Rc<RefCell<Option<Cancellable>>>,
}

impl FilesPreviewHandler {
    pub fn new() -> Self {
        Self {
            cached_preview: RefCell::new(None),
        }
    }
}

impl PreviewHandler for FilesPreviewHandler {
    fn clear_cache(&self) {
        let mut cached_preview = self.cached_preview.borrow_mut();
        if let Some(preview) = cached_preview.as_mut() {
            preview.clear_preview();
            preview.image_cache.borrow_mut().clear();
        }
        *cached_preview = None;
    }

    fn handle(&self, item: &Item, preview: &GtkBox, builder: &Builder) {
        let file_path = if item.preview.is_empty() {
            item.text.as_str()
        } else {
            item.preview.as_str()
        };

        if !Path::new(file_path).exists() {
            return;
        }

        let Some(current) = get_selected_item() else {
            return;
        };

        if current != *item {
            return;
        }

        let mut cached_preview = self.cached_preview.borrow_mut();
        if let Some(existing) = cached_preview.as_ref()
            && existing.current_file != item.text
        {
            *cached_preview = None;
        }

        if cached_preview.is_none()
            && let Ok(preview) =
                FilePreview::new_with_builder(&builder).or_else(|_| FilePreview::new())
        {
            *cached_preview = Some(preview);
        } else if cached_preview.is_none() {
            return;
        }

        let file_preview = cached_preview.as_mut().unwrap();
        if file_preview.preview_file(file_path).is_err() {
            return;
        }

        while let Some(child) = preview.first_child() {
            child.unparent();
        }

        let existing_controllers: Vec<_> = preview
            .observe_controllers()
            .into_iter()
            .filter_map(Result::ok)
            .collect();
        for controller in existing_controllers {
            if let Ok(drag_source) = controller.downcast::<DragSource>() {
                preview.remove_controller(&drag_source);
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

        drag_source.connect_drag_begin(|_, _| with_window(|w| w.window.set_visible(false)));
        drag_source.connect_drag_end(|_, _, _| with_window(|w| quit(&w.app, false)));

        file_preview.box_widget.set_can_target(false);
        preview.add_controller(drag_source);
        preview.append(&file_preview.box_widget);
        preview.set_visible(get_selected_item().is_some_and(|current| current == *item));
    }
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
            image_cache: Rc::new(RefCell::new(HashMap::new())),
            current_video_cancellable: Rc::new(RefCell::new(None)),
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
            image_cache: Rc::new(RefCell::new(HashMap::new())),
            current_video_cancellable: Rc::new(RefCell::new(None)),
        })
    }

    pub fn preview_file(&mut self, file_path: &str) -> Result<(), Box<dyn std::error::Error>> {
        self.current_file = file_path.to_string();
        self.clear_preview();

        let Some(guess) = new_mime_guess::from_path(file_path).first() else {
            return self.preview_generic(file_path);
        };

        let function = match (guess.type_(), guess.subtype()) {
            (mime::IMAGE, _) => Self::preview_image,
            (mime::APPLICATION, mime::PDF) => Self::preview_pdf,
            (mime::TEXT, _) => Self::preview_text,
            (mime::VIDEO, _) => Self::preview_video,
            _ => Self::preview_generic,
        };

        function(self, file_path)
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

            while let Some(container) = child.downcast_ref::<gtk4::Box>()
                && let Some(nested_child) = container.first_child()
            {
                if let Some(nested_picture) = nested_child.downcast_ref::<Picture>() {
                    nested_picture.set_paintable(gtk4::gdk::Paintable::NONE);
                }
                if let Some(nested_image) = nested_child.downcast_ref::<Image>() {
                    nested_image.clear();
                    nested_image.set_icon_name(Option::<&str>::None);
                }

                container.remove(&nested_child);
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

    fn preview_pdf(&self, file_path: &str) -> Result<(), Box<dyn std::error::Error>> {
        let uri = format!("file://{file_path}");
        let document =
            Document::from_file(&uri, None).map_err(|e| format!("Failed to load PDF: {e}"))?;

        let pdf = GtkBox::new(Orientation::Vertical, 0);

        if let Some(page) = document.page(0) {
            match self.render_pdf_page(&page) {
                Ok(page_widget) => pdf.append(&page_widget),
                Err(e) => eprintln!("Failed to render PDF page: {e}"),
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

        let render_scale = (display_scale * 1.5).min(2.0);
        let render_width = ((width * render_scale) as i32).min(1600);
        let render_height = ((height * render_scale) as i32).min(2400);

        let estimated_size = (render_width * render_height * 4) as usize;
        if estimated_size > 15 * 1024 * 1024 {
            return Err("PDF page too large for preview".into());
        }

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
            let mut rgba_data = surface.data()?.to_vec();

            for chunk in rgba_data.chunks_exact_mut(4) {
                chunk.swap(0, 2);
            }

            Bytes::from_owned(rgba_data)
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
        picture.set_content_fit(ContentFit::Contain);

        let scrolled = ScrolledWindow::new();
        scrolled.set_child(Some(&picture));
        scrolled.set_policy(PolicyType::Automatic, PolicyType::Automatic);

        self.preview_area.add_child(&scrolled);
        self.preview_area.set_visible_child(&scrolled);

        if let Some(cached_texture) = self.image_cache.borrow().get(file_path) {
            picture.set_paintable(Some(cached_texture));
            return Ok(());
        }

        let file = gio::File::for_path(file_path);
        picture.set_file(Some(&file));

        // load and cache in background for future use
        let file_path_clone = file_path.to_string();
        let cache_clone = self.image_cache.clone();

        glib::MainContext::ref_thread_default().spawn_local(async move {
            let cancellable = gio::Cancellable::new();
            match Self::load_image_async(&file_path_clone, &cancellable, cache_clone).await {
                Ok(_) => {}
                Err(e) => {
                    eprintln!("Failed to cache image {}: {}", file_path_clone, e);
                }
            }
        });

        Ok(())
    }

    async fn load_image_async(
        file_path: &str,
        cancellable: &gio::Cancellable,
        cache: Rc<RefCell<HashMap<String, gtk4::gdk::Texture>>>,
    ) -> Result<gtk4::gdk::Texture, Box<dyn std::error::Error>> {
        if let Some(cached_texture) = cache.borrow().get(file_path) {
            return Ok(cached_texture.clone());
        }

        let file = gio::File::for_path(file_path);

        let (bytes, _) = file.load_bytes_future().await?;
        let stream = gio::MemoryInputStream::from_bytes(&bytes);

        let pixbuf = gdk_pixbuf::Pixbuf::from_stream(&stream, Some(cancellable))?;

        // Downscale to small size for fast preview (max 360x360)
        let max_size = 360;
        let (width, height) = (pixbuf.width(), pixbuf.height());
        let (new_width, new_height) = if width > max_size || height > max_size {
            let ratio = (max_size as f64) / (width.max(height) as f64);
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

        let mut cache_mut = cache.borrow_mut();
        if cache_mut.len() >= 50 {
            if let Some(key) = cache_mut.keys().next().cloned() {
                cache_mut.remove(&key);
            }
        }
        cache_mut.insert(file_path.to_string(), texture.clone());

        Ok(texture)
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
        buffer.set_text(
            String::from_utf8(display_content)
                .as_ref()
                .map(String::as_str)
                .unwrap_or("[Binary file - cannot display as text]"),
        );

        let scrolled = ScrolledWindow::new();
        scrolled.set_child(Some(&text_view));
        scrolled.set_policy(PolicyType::Automatic, PolicyType::Automatic);
        scrolled.set_size_request(300, 250);

        self.preview_area.add_child(&scrolled);
        self.preview_area.set_visible_child(&scrolled);
        Ok(())
    }

    fn preview_video(&self, file_path: &str) -> Result<(), Box<dyn std::error::Error>> {
        // Cancel previous load
        if let Some(c) = self.current_video_cancellable.borrow_mut().take() {
            c.cancel();
        }

        let cancellable = Cancellable::new();
        *self.current_video_cancellable.borrow_mut() = Some(cancellable.clone());

        let scrolled = self.get_or_create_scrolled();
        scrolled.set_size_request(128, 72);
        scrolled.set_child(None::<gtk4::Widget>.as_ref());

        let placeholder = GtkBox::new(Orientation::Vertical, 10);
        placeholder.set_halign(gtk4::Align::Center);
        placeholder.set_valign(gtk4::Align::Center);
        let icon = Image::from_icon_name("video-x-generic");
        icon.set_pixel_size(64);
        placeholder.append(&icon);
        scrolled.set_child(Some(&placeholder));

        self.preview_area.set_visible_child(&scrolled);

        //added a 200ms debounce to make fast scrolling smoother
        let file_path_clone = file_path.to_string();
        let scrolled_clone = scrolled.clone();
        let placeholder_clone = placeholder.clone();
        let cancellable_clone = cancellable.clone();

        glib::timeout_add_local(std::time::Duration::from_millis(200), move || {
            if cancellable_clone.is_cancelled() {
                return glib::ControlFlow::Break;
            }
            if !scrolled_clone.is_visible() {
                return glib::ControlFlow::Break;
            }

            let file = gio::File::for_path(&file_path_clone);
            let video = Video::for_file(Some(&file));
            video.set_autoplay(true);

            // Replace placeholder with video
            placeholder_clone.set_visible(false);
            scrolled_clone.set_child(Some(&video));

            glib::ControlFlow::Break
        });

        Ok(())
    }

    fn get_or_create_scrolled(&self) -> ScrolledWindow {
        if let Some(child) = self.preview_area.first_child() {
            if let Ok(scrolled) = child.downcast::<ScrolledWindow>() {
                return scrolled;
            }
        }
        let scrolled = ScrolledWindow::new();
        scrolled.set_policy(PolicyType::Automatic, PolicyType::Automatic);
        scrolled.set_size_request(300, 250);
        scrolled.set_halign(gtk4::Align::Fill);
        scrolled.set_valign(gtk4::Align::Fill);
        scrolled.set_hexpand(true);
        scrolled.set_vexpand(true);
        self.preview_area.add_child(&scrolled);
        scrolled
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

        let icon = Image::from_icon_name("text-x-generic");
        icon.set_icon_size(gtk4::IconSize::Large);
        icon.add_css_class("preview-generic-icon");

        // Try to get file-specific icon, but fallback gracefully to avoid memory issues
        let file = gio::File::for_path(file_path);
        if let Ok(info) = file.query_info(
            "standard::icon",
            gio::FileQueryInfoFlags::NONE,
            gio::Cancellable::NONE,
        ) && let Some(file_icon) = info.icon()
        {
            icon.set_from_gicon(&file_icon);
        }

        container.append(&icon);

        self.preview_area.add_child(&container);
        self.preview_area.set_visible_child(&container);
        Ok(())
    }
}
