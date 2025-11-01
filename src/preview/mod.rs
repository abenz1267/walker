use crate::config::get_config;
use crate::protos::generated_proto::query::query_response::Item;
use crate::renderers::create_drag_source;
use crate::ui::window::get_selected_item;
use gtk4::gio::{self, Cancellable};
use gtk4::glib::{self, Bytes};
use gtk4::{
    Box as GtkBox, Builder, ContentFit, Image, Orientation, Picture, PolicyType, ScrolledWindow,
    Stack, TextView, Video, WrapMode, prelude::*,
};
use poppler::{Document, Page};
use std::cell::RefCell;
use std::path::Path;
use std::process::Command;
use std::rc::Rc;

#[derive(Debug)]
pub struct UnifiedPreviewHandler {
    cached_preview: RefCell<Option<PreviewWidget>>,
}

#[derive(Debug)]
pub struct PreviewWidget {
    pub box_widget: GtkBox,
    preview_area: Stack,
    current_content: String,
    pub current_video_cancellable: Rc<RefCell<Option<Cancellable>>>,
}

impl UnifiedPreviewHandler {
    pub fn new() -> Self {
        Self {
            cached_preview: RefCell::new(None),
        }
    }

    pub fn clear_cache(&self) {
        let mut cached_preview = self.cached_preview.borrow_mut();
        if let Some(preview) = cached_preview.as_mut() {
            preview.clear_preview();
        }
        *cached_preview = None;
    }

    pub fn handle(&self, item: &Item, preview: &GtkBox, builder: &Builder) {
        // Check if preview is disabled for this provider
        let config = get_config();
        if config.providers.ignore_preview.contains(&item.provider) {
            return;
        }

        // Only show preview if this item is currently selected
        let Some(current) = get_selected_item() else {
            return;
        };

        if current != *item {
            return;
        }

        // Get or create preview widget
        let mut cached_preview = self.cached_preview.borrow_mut();
        if let Some(existing) = cached_preview.as_ref()
            && existing.current_content
                != format!("{}{}{}", item.preview_type, item.preview, item.text)
        {
            *cached_preview = None;
        }

        if cached_preview.is_none()
            && let Ok(preview_widget) =
                PreviewWidget::new_with_builder(&builder).or_else(|_| PreviewWidget::new())
        {
            *cached_preview = Some(preview_widget);
        } else if cached_preview.is_none() {
            return;
        }

        let preview_widget = cached_preview.as_mut().unwrap();

        // Handle preview based on preview_type
        let result = match item.preview_type.as_str() {
            "text" => preview_widget.preview_text(&item.preview),
            "file" => {
                if item.preview.is_empty() {
                    preview_widget.preview_file(&item.text)
                } else {
                    preview_widget.preview_file(&item.preview)
                }
            }
            "command" => preview_widget.preview_command(&item.preview),
            _ => {
                return;
            }
        };

        if result.is_err() {
            return;
        }

        // Clear existing preview and add new one
        while let Some(child) = preview.first_child() {
            child.unparent();
        }

        preview.append(&preview_widget.box_widget);
        preview.set_visible(get_selected_item().is_some_and(|current| current == *item));
    }
}

pub fn handle_preview(item: &Item, preview: &GtkBox, builder: &Builder) {
    thread_local! {
        static PREVIEW_HANDLER: std::cell::RefCell<UnifiedPreviewHandler> = std::cell::RefCell::new(UnifiedPreviewHandler::new());
    }

    PREVIEW_HANDLER.with(|handler| {
        handler.borrow().handle(item, preview, builder);
    });
}

pub fn clear_all_caches() {
    thread_local! {
        static PREVIEW_HANDLER: std::cell::RefCell<UnifiedPreviewHandler> = std::cell::RefCell::new(UnifiedPreviewHandler::new());
    }

    PREVIEW_HANDLER.with(|handler| {
        handler.borrow().clear_cache();
    });
}

impl PreviewWidget {
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
            current_content: String::new(),
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
            current_content: String::new(),
            current_video_cancellable: Rc::new(RefCell::new(None)),
        })
    }

    pub fn preview_text(&mut self, text: &str) -> Result<(), Box<dyn std::error::Error>> {
        self.current_content = format!("text{}", text);
        self.clear_preview();

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

    pub fn preview_file(&mut self, file_path: &str) -> Result<(), Box<dyn std::error::Error>> {
        self.current_content = format!("file{}", file_path);
        self.clear_preview();

        if Path::new(file_path).is_absolute() {
            self.box_widget
                .add_controller(create_drag_source(file_path));
        }

        if !Path::new(file_path).exists() {
            return Err(format!("File does not exist: {}", file_path).into());
        }

        let Some(guess) = new_mime_guess::from_path(file_path).first() else {
            return self.preview_generic(file_path);
        };

        let function = match (guess.type_(), guess.subtype()) {
            (mime::IMAGE, _) => Self::preview_image,
            (mime::APPLICATION, mime::PDF) => Self::preview_pdf,
            (mime::TEXT, _) => Self::preview_text_file,
            (mime::VIDEO, _) => Self::preview_video,
            _ => Self::preview_generic,
        };

        function(self, file_path)
    }

    pub fn preview_command(&mut self, command: &str) -> Result<(), Box<dyn std::error::Error>> {
        self.current_content = format!("command{}", command);
        self.clear_preview();

        // Execute command and capture output
        let output = Command::new("sh").arg("-c").arg(command).output()?;

        let stdout = String::from_utf8_lossy(&output.stdout);
        let stderr = String::from_utf8_lossy(&output.stderr);

        let combined_output = if stderr.is_empty() {
            stdout.to_string()
        } else {
            format!("{}\n\nSTDERR:\n{}", stdout, stderr)
        };

        let text_view = TextView::new();
        text_view.set_editable(false);
        text_view.set_monospace(true);
        text_view.set_wrap_mode(WrapMode::Word);
        text_view.set_size_request(400, 300);

        let buffer = text_view.buffer();
        buffer.set_text(&combined_output);

        let scrolled = ScrolledWindow::new();
        scrolled.set_child(Some(&text_view));
        scrolled.set_policy(PolicyType::Automatic, PolicyType::Automatic);
        scrolled.set_size_request(400, 300);

        self.preview_area.add_child(&scrolled);
        self.preview_area.set_visible_child(&scrolled);

        Ok(())
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

    fn preview_image(&self, file_path: &str) -> Result<(), Box<dyn std::error::Error>> {
        let picture = Picture::new();
        picture.set_content_fit(ContentFit::Contain);

        let scrolled = ScrolledWindow::new();
        scrolled.set_child(Some(&picture));
        scrolled.set_policy(PolicyType::Automatic, PolicyType::Automatic);

        self.preview_area.add_child(&scrolled);
        self.preview_area.set_visible_child(&scrolled);


        let file = gio::File::for_path(file_path);
        picture.set_file(Some(&file));


        Ok(())
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
        if estimated_size > 10 * 1024 * 1024 {
            return Err("PDF page too large for preview".into());
        }

        let mut surface =
            cairo::ImageSurface::create(cairo::Format::ARgb32, render_width, render_height)?;

        let bytes = {
            let ctx = cairo::Context::new(&surface)?;

            ctx.set_antialias(cairo::Antialias::Best);
            ctx.scale(render_scale, render_scale);

            ctx.set_source_rgb(1.0, 1.0, 1.0);
            ctx.paint()?;

            page.render(&ctx);

            ctx.target().flush();
            drop(ctx);
            
            surface.flush();
            
            let surface_data = surface.data()?;
            let mut rgba_data = Vec::with_capacity(surface_data.len());
            rgba_data.extend_from_slice(&surface_data);
            
            drop(surface_data);
            
            for chunk in rgba_data.chunks_exact_mut(4) {
                chunk.swap(0, 2);
            }

            Bytes::from_owned(rgba_data)
        };

        drop(surface);

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

    fn preview_video(&self, file_path: &str) -> Result<(), Box<dyn std::error::Error>> {
        // Cancel previous load
        if let Some(c) = self.current_video_cancellable.borrow_mut().take() {
            c.cancel();
        }

        let cancellable = Cancellable::new();
        *self.current_video_cancellable.borrow_mut() = Some(cancellable.clone());

        let scrolled = self.get_or_create_scrolled();
        scrolled.set_size_request(128, 72);
        
        // Properly cleanup existing child
        if let Some(existing_child) = scrolled.child() {
            if let Some(video) = existing_child.downcast_ref::<Video>() {
                video.set_file(None::<&gio::File>);
            }
            scrolled.set_child(None::<gtk4::Widget>.as_ref());
        }

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
        let cancellable_clone = cancellable.clone();

        glib::timeout_add_local(std::time::Duration::from_millis(200), move || {
            if cancellable_clone.is_cancelled() {
                return glib::ControlFlow::Break;
            }
            if !scrolled_clone.is_visible() {
                return glib::ControlFlow::Break;
            }

            // Clean up placeholder properly before replacing
            if let Some(_child) = scrolled_clone.child() {
                scrolled_clone.set_child(None::<gtk4::Widget>.as_ref());
            }

            let file = gio::File::for_path(&file_path_clone);
            let video = Video::for_file(Some(&file));
            video.set_autoplay(true);

            scrolled_clone.set_child(Some(&video));

            glib::ControlFlow::Break
        });

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

    fn preview_text_file(&self, file_path: &str) -> Result<(), Box<dyn std::error::Error>> {
        use std::io::{BufRead, BufReader};
        
        let file = std::fs::File::open(file_path)?;
        let mut reader = BufReader::new(file);
        
        let max_size = 512 * 1024; // 512KB
        let mut content = String::with_capacity(max_size.min(64 * 1024));
        let mut total_read = 0;
        let mut buffer = String::new();
        
        while total_read < max_size {
            buffer.clear();
            let bytes_read = reader.read_line(&mut buffer)?;
            if bytes_read == 0 {
                break; // EOF
            }
            
            if total_read + bytes_read > max_size {
                let remaining = max_size - total_read;
                content.push_str(&buffer[..remaining]);
                content.push_str("\n\n[File truncated...]");
                break;
            }
            
            content.push_str(&buffer);
            total_read += bytes_read;
        }
        
        let text_view = TextView::new();
        text_view.set_editable(false);
        text_view.set_monospace(true);
        text_view.set_wrap_mode(WrapMode::Word);
        text_view.set_size_request(300, 200);
        let buffer = text_view.buffer();
        buffer.set_text(&content);

        let scrolled = ScrolledWindow::new();
        scrolled.set_child(Some(&text_view));
        scrolled.set_policy(PolicyType::Automatic, PolicyType::Automatic);
        scrolled.set_size_request(300, 250);

        self.preview_area.add_child(&scrolled);
        self.preview_area.set_visible_child(&scrolled);
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

}
