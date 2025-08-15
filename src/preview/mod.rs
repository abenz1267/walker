mod files_preview;
mod throttler;

pub use files_preview::FilesPreviewHandler;
pub use throttler::LatestOnlyThrottler;

use crate::protos::generated_proto::query::query_response::Item;
use gtk4::{Box as GtkBox, Builder};
use std::cell::RefCell;
use std::collections::HashMap;
use std::fmt::Debug;

pub trait PreviewHandler: Debug {
    fn handle(&self, item: &Item, preview: &GtkBox, builder: &Builder);
}

// Use thread_local instead of static since GTK widgets aren't Send + Sync
thread_local! {
    static PREVIEWERS: RefCell<HashMap<String, Box<dyn PreviewHandler>>> = RefCell::new(HashMap::new());
}

pub fn load_previewers() {
    PREVIEWERS.with(|previewers| {
        let mut previewers = previewers.borrow_mut();
        previewers.insert("files".to_string(), Box::new(FilesPreviewHandler::new()));
        // Add other handlers here as needed
    });
}

pub fn get_previewer<F, R>(provider: &str, f: F) -> Option<R>
where
    F: FnOnce(&dyn PreviewHandler) -> R,
{
    PREVIEWERS.with(|previewers| {
        previewers
            .borrow()
            .get(provider)
            .map(|handler| f(handler.as_ref()))
    })
}

pub fn handle_preview(provider: &str, item: &Item, preview: &GtkBox, builder: &Builder) {
    get_previewer(provider, |handler| {
        handler.handle(item, preview, builder);
    });
}

pub fn has_previewer(provider: &str) -> bool {
    PREVIEWERS.with(|previewers| previewers.borrow().contains_key(provider))
}

pub fn get_registered_providers() -> Vec<String> {
    PREVIEWERS.with(|previewers| previewers.borrow().keys().cloned().collect())
}
