use gtk4::gio::ListStore;
use gtk4::{
    Application, Builder, CssProvider, Entry, GridView, Label, ScrolledWindow, SingleSelection,
    Window,
};
use std::cell::{Cell, RefCell};
use std::collections::HashSet;
use std::sync::OnceLock;

thread_local! {
    static STATE: OnceLock<AppState> = OnceLock::new();
    pub static CSS_PROVIDER: RefCell<Option<CssProvider>> = RefCell::new(None);
}

pub fn set_css_provider(provider: CssProvider) {
    CSS_PROVIDER.with(|p| *p.borrow_mut() = Some(provider));
}

pub fn with_css_provider<F, R>(f: F) -> Option<R>
where
    F: FnOnce(&CssProvider) -> R,
{
    CSS_PROVIDER.with(|p| p.borrow().as_ref().map(f))
}

#[derive(Debug, Clone)]
pub struct AppState {
    dmenu_keep_open: Cell<bool>,
    dmenu_exit_after: Cell<bool>,
    initial_height: Cell<i32>,
    initial_width: Cell<i32>,
    dmenu_current: Cell<i64>,
    parameter_height: Cell<i32>,
    parameter_width: Cell<i32>,
    last_query: RefCell<String>,
    placeholder: RefCell<String>,
    initial_placeholder: RefCell<String>,
    available_themes: RefCell<HashSet<String>>,
    provider: RefCell<String>,
    theme: RefCell<String>,
    is_service: Cell<bool>,
    no_search: Cell<bool>,
    input_only: Cell<bool>,
    is_dmenu: Cell<bool>,
    is_param_close: Cell<bool>,
    current_prefix: RefCell<String>,
    pub(crate) is_visible: Cell<bool>,
}

impl AppState {
    pub fn new() -> Self {
        Self {
            provider: RefCell::new(String::new()),
            available_themes: RefCell::new(HashSet::new()),
            theme: RefCell::new("".to_string()),
            current_prefix: RefCell::new("".to_string()),
            placeholder: RefCell::new("".to_string()),
            initial_placeholder: RefCell::new("".to_string()),
            last_query: RefCell::new(String::new()),
            is_service: Cell::new(false),
            is_param_close: Cell::new(false),
            is_visible: Cell::new(false),
            input_only: Cell::new(false),
            no_search: Cell::new(false),
            is_dmenu: Cell::new(false),
            dmenu_keep_open: Cell::new(false),
            dmenu_exit_after: Cell::new(false),
            initial_height: Cell::new(0),
            parameter_height: Cell::new(0),
            parameter_width: Cell::new(0),
            initial_width: Cell::new(0),
            dmenu_current: Cell::new(0),
        }
    }

    pub fn get_theme(&self) -> String {
        self.theme.borrow().clone()
    }

    pub fn set_theme(&self, val: &str) {
        *self.theme.borrow_mut() = val.to_string();
    }

    pub fn get_current_prefix(&self) -> String {
        self.current_prefix.borrow().clone()
    }

    pub fn set_current_prefix(&self, val: &str) {
        *self.current_prefix.borrow_mut() = val.to_string();
    }

    pub fn get_provider(&self) -> String {
        self.provider.borrow().clone()
    }

    pub fn set_provider(&self, val: &str) {
        *self.provider.borrow_mut() = val.to_string();
    }

    pub fn get_initial_placeholder(&self) -> String {
        self.initial_placeholder.borrow().clone()
    }

    pub fn set_initial_placeholder(&self, val: &str) {
        *self.initial_placeholder.borrow_mut() = val.to_string();
    }

    pub fn get_placeholder(&self) -> String {
        self.placeholder.borrow().clone()
    }

    pub fn set_placeholder(&self, val: &str) {
        *self.placeholder.borrow_mut() = val.to_string();
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

    pub fn is_input_only(&self) -> bool {
        self.input_only.get()
    }

    pub fn set_input_only(&self, val: bool) {
        self.input_only.set(val);
    }

    pub fn is_param_close(&self) -> bool {
        self.is_param_close.get()
    }

    pub fn set_param_close(&self, val: bool) {
        self.is_param_close.set(val);
    }

    pub fn is_dmenu_keep_open(&self) -> bool {
        self.dmenu_keep_open.get()
    }

    pub fn set_dmenu_keep_open(&self, val: bool) {
        self.dmenu_keep_open.set(val);
    }

    pub fn is_dmenu_exit_after(&self) -> bool {
        self.dmenu_exit_after.get()
    }

    pub fn set_dmenu_exit_after(&self, val: bool) {
        self.dmenu_exit_after.set(val);
    }

    pub fn is_dmenu(&self) -> bool {
        self.is_dmenu.get()
    }

    pub fn set_is_dmenu(&self, val: bool) {
        self.is_dmenu.set(val);
    }

    pub fn is_no_search(&self) -> bool {
        self.no_search.get()
    }

    pub fn is_service(&self) -> bool {
        self.is_service.get()
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

    pub fn get_dmenu_current(&self) -> i64 {
        return self.dmenu_current.get();
    }

    pub fn set_dmenu_current(&self, val: i64) {
        return self.dmenu_current.set(val);
    }

    pub fn add_theme(&self, val: String) {
        self.available_themes.borrow_mut().insert(val);
    }

    pub fn has_theme(&self, val: String) -> bool {
        return self.available_themes.borrow().contains(&val);
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
    pub input: Option<Entry>,
    pub items: ListStore,
    pub placeholder: Option<Label>,
    pub keybinds: Option<Label>,
    pub scroll: ScrolledWindow,
    pub search_container: Option<gtk4::Box>,
    pub content_container: gtk4::Box,
}

pub fn init_app_state() {
    let state = AppState::new();
    STATE.with(|s| s.set(state.clone()).expect("failed initializing app state"));
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
