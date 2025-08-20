use gtk4::gio::ListStore;
use gtk4::{
    Application, Builder, CssProvider, Entry, GridView, Label, ScrolledWindow, SingleSelection,
    Window,
};
use std::cell::{Cell, RefCell};
use std::sync::OnceLock;

thread_local! {
    static STATE: OnceLock<AppState> = OnceLock::new();
    pub static CSS_PROVIDER: RefCell<Option<CssProvider>> = RefCell::new(None);
}

pub fn set_css_provider(provider: CssProvider) {
    CSS_PROVIDER.with(|p| {
        *p.borrow_mut() = Some(provider);
    });
}

pub fn has_css_provider() -> bool {
    CSS_PROVIDER.with(|p| p.borrow().is_some())
}

pub fn clear_css_provider() {
    CSS_PROVIDER.with(|p| {
        *p.borrow_mut() = None;
    });
}

pub fn with_css_provider<F, R>(f: F) -> Option<R>
where
    F: FnOnce(&CssProvider) -> R,
{
    CSS_PROVIDER.with(|p| p.borrow().as_ref().map(f))
}

#[derive(Debug, Clone)]
pub struct AppState {
    initial_height: Cell<i32>,
    initial_width: Cell<i32>,
    parameter_height: Cell<i32>,
    parameter_width: Cell<i32>,
    last_query: RefCell<String>,
    provider: RefCell<String>,
    theme: RefCell<String>,
    is_service: Cell<bool>,
    no_search: Cell<bool>,
    pub(crate) is_visible: Cell<bool>,
}

impl AppState {
    pub fn new() -> Self {
        Self {
            provider: RefCell::new(String::new()),
            theme: RefCell::new("default".to_string()),
            last_query: RefCell::new(String::new()),
            is_service: Cell::new(false),
            is_visible: Cell::new(false),
            no_search: Cell::new(false),
            initial_height: Cell::new(0),
            parameter_height: Cell::new(0),
            parameter_width: Cell::new(0),
            initial_width: Cell::new(0),
        }
    }

    pub fn get_theme(&self) -> String {
        self.theme.borrow().clone()
    }

    pub fn set_theme(&self, val: &str) {
        *self.theme.borrow_mut() = val.to_string();
    }

    pub fn get_provider(&self) -> String {
        self.provider.borrow().clone()
    }

    pub fn set_provider(&self, val: &str) {
        *self.provider.borrow_mut() = val.to_string();
    }

    pub fn get_last_query(&self) -> String {
        self.last_query.borrow().clone()
    }

    pub fn set_last_query(&self, val: &str) {
        *self.last_query.borrow_mut() = val.to_string();
    }

    pub fn set_is_service(&self, val: bool) {
        self.is_service.set(val);
    }

    pub fn is_visible(&self) -> bool {
        self.is_visible.get()
    }

    pub fn set_is_visible(&self, val: bool) {
        self.is_visible.set(val);
    }

    pub fn is_no_search(&self) -> bool {
        self.no_search.get()
    }

    pub fn set_no_search(&self, val: bool) {
        self.no_search.set(val);
    }

    pub fn set_initial_height(&self, val: i32) {
        self.initial_height.set(val);
    }

    pub fn set_initial_width(&self, val: i32) {
        self.initial_width.set(val);
    }

    pub fn get_initial_height(&self) -> i32 {
        return self.initial_height.get();
    }

    pub fn get_initial_width(&self) -> i32 {
        return self.initial_width.get();
    }

    pub fn set_parameter_height(&self, val: i32) {
        self.parameter_height.set(val);
    }

    pub fn set_parameter_width(&self, val: i32) {
        self.parameter_width.set(val);
    }

    pub fn get_parameter_height(&self) -> i32 {
        return self.parameter_height.get();
    }

    pub fn get_parameter_width(&self) -> i32 {
        return self.parameter_width.get();
    }
}

#[derive(Debug, Clone)]
pub struct WindowData {
    pub builder: Builder,
    pub preview_builder: RefCell<Option<Builder>>,
    pub mouse_x: Cell<f64>,
    pub mouse_y: Cell<f64>,
    pub app: Application,
    pub window: Window,
    pub selection: SingleSelection,
    pub list: GridView,
    pub input: Entry,
    pub items: ListStore,
    pub placeholder: Option<Label>,
    pub keybinds: Option<Label>,
    pub scroll: ScrolledWindow,
    pub search_container: gtk4::Box,
}

pub fn init_app_state() {
    let state = AppState::new();
    STATE.with(|s| {
        s.set(state.clone()).expect("failed initializing app state");
    });
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
