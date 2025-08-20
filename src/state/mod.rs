use gtk4::gio::ListStore;
use gtk4::{
    Application, Builder, CssProvider, Entry, Label, ListView, ScrolledWindow, SingleSelection,
    Window,
};
use std::cell::{Cell, RefCell};
use std::sync::OnceLock;

thread_local! {
    static STATE: OnceLock<AppState> = OnceLock::new();
}

#[derive(Debug, Clone)]
pub struct AppState {
    last_query: RefCell<String>,
    provider: RefCell<String>,
    is_service: Cell<bool>,
    pub(crate) is_visible: Cell<bool>,
}

impl AppState {
    pub fn new() -> Self {
        Self {
            provider: RefCell::new(String::new()),
            last_query: RefCell::new(String::new()),
            is_service: Cell::new(false),
            is_visible: Cell::new(false),
        }
    }

    pub fn get_provider(&self) -> String {
        self.provider.borrow().clone()
    }

    pub fn set_provider(&self, new_provider: &str) {
        *self.provider.borrow_mut() = new_provider.to_string();
    }

    pub fn get_last_query(&self) -> String {
        self.last_query.borrow().clone()
    }

    pub fn set_last_query(&self, new_provider: &str) {
        *self.last_query.borrow_mut() = new_provider.to_string();
    }

    pub fn set_is_service(&self, is_service: bool) {
        self.is_service.set(is_service);
    }

    pub fn is_visible(&self) -> bool {
        self.is_visible.get()
    }

    pub fn set_is_visible(&self, is_visible: bool) {
        self.is_visible.set(is_visible);
    }
}

#[derive(Debug, Clone)]
pub struct WindowData {
    pub builder: Builder,
    pub preview_builder: RefCell<Option<Builder>>,
    pub mouse_x: Cell<f64>,
    pub mouse_y: Cell<f64>,
    pub app: Application,
    pub css_provider: CssProvider,
    pub window: Window,
    pub selection: SingleSelection,
    pub list: ListView,
    pub input: Entry,
    pub items: ListStore,
    pub placeholder: Option<Label>,
    pub keybinds: Option<Label>,
    pub scroll: ScrolledWindow,
}

pub fn init_app_state() -> AppState {
    let state = AppState::new();
    STATE.with(|s| {
        s.set(state.clone()).expect("failed initializing app state");
    });
    state
}

pub fn with_state<F, R>(f: F) -> R
where
    F: FnOnce(&AppState) -> R,
{
    STATE.with(|state| {
        let data = state.get().expect("AppState not initialized");
        f(data)
    })
}
