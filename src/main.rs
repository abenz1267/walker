mod config;
mod data;
mod keybinds;
mod preview;

mod protos;
use chrono::DateTime;
use gtk4::gdk::ContentProvider;
use gtk4::gio::File;
use gtk4::gio::prelude::{ApplicationCommandLineExt, FileExt, ListModelExt};
use gtk4::glib::clone::Downgrade;
use gtk4::glib::object::{CastNone, ObjectExt};
use gtk4::glib::subclass::types::ObjectSubclassIsExt;
use gtk4::glib::{OptionFlags, VariantTy};
use gtk4::prelude::{
    BoxExt, EditableExt, EntryExt, EventControllerExt, ListItemExt, SelectionModelExt,
};
use gtk4::{
    Box, DragSource, Entry, EventControllerMotion, Image, ListItem, ListScrollFlags, Picture,
    ScrolledWindow, gio,
};
use gtk4::{gio::ListStore, glib::object::Cast};
use notify::{Config, Event, RecommendedWatcher, RecursiveMode, Watcher};
use protos::generated_proto::query::query_response::Item;

use config::get_config;

use std::cell::Cell;
use std::path::Path;
use std::sync::mpsc;
use std::time::Duration;
use std::{
    cell::RefCell,
    sync::{Mutex, OnceLock},
};
use std::{env, fs, thread};

use gtk4::{
    Application, Builder, CssProvider, EventControllerKey, Label, ListView, SignalListItemFactory,
    SingleSelection, Window,
    gdk::Display,
    gio::{
        ApplicationFlags,
        prelude::{ApplicationExt, ApplicationExtManual},
    },
    glib::{self},
    prelude::{GtkApplicationExt, GtkWindowExt, WidgetExt},
};
use gtk4_layer_shell::{Edge, KeyboardMode, Layer, LayerShell};

use crate::config::DEFAULT_STYLE;
use crate::data::{SWITCHER_PROVIDER, activate, init_socket, input_changed, start_listening};
use crate::keybinds::{
    ACTION_SELECT_NEXT, ACTION_SELECT_PREVIOUS, ACTION_TOGGLE_EXACT, AFTER_CLEAR_RELOAD,
    AFTER_CLOSE, AFTER_RELOAD, get_modifiers, get_provider_bind,
};
use crate::{
    keybinds::{ACTION_CLOSE, get_bind, setup_binds},
    protos::generated_proto::query::{QueryResponse, query_response::Type},
};

static IS_VISIBLE: Mutex<bool> = Mutex::new(false);
static IS_SERVICE: OnceLock<bool> = OnceLock::new();

thread_local! {
static WINDOW: OnceLock<WindowData> = OnceLock::new();
}

#[derive(Debug, Clone)]
struct WindowData {
    mouse_x: Cell<f64>,
    mouse_y: Cell<f64>,
    app: Application,
    css_provider: CssProvider,
    window: Window,
    selection: SingleSelection,
    list: ListView,
    input: Entry,
    items: ListStore,
    placeholder: Option<Label>,
    keybinds: Option<Label>,
    scroll: ScrolledWindow,
}

fn with_window<F, R>(f: F) -> R
where
    F: FnOnce(&WindowData) -> R,
{
    WINDOW.with(|window| {
        let data = window.get().expect("Window not initialized");
        f(data)
    })
}

// GObject wrapper for QueryResponse
mod imp {
    use crate::protos::generated_proto::query::QueryResponse;

    use super::*;
    use gtk4::subclass::prelude::*;
    use std::cell::RefCell;

    #[derive(Debug, Default)]
    pub struct QueryResponseObject {
        pub response: RefCell<Option<QueryResponse>>,
    }

    #[glib::object_subclass]
    impl ObjectSubclass for QueryResponseObject {
        const NAME: &'static str = "QueryResponseObject";
        type Type = super::QueryResponseObject;
    }

    impl ObjectImpl for QueryResponseObject {}
}

glib::wrapper! {
    pub struct QueryResponseObject(ObjectSubclass<imp::QueryResponseObject>);
}

impl QueryResponseObject {
    pub fn new(response: QueryResponse) -> Self {
        let obj: Self = glib::Object::builder().build();
        obj.imp().response.replace(Some(response));
        obj
    }

    pub fn response(&self) -> QueryResponse {
        self.imp().response.borrow().as_ref().unwrap().clone()
    }
}

fn wait_for_file(path: &str) {
    while !Path::new(path).exists() {
        thread::sleep(Duration::from_millis(10));
    }
}

fn main() -> glib::ExitCode {
    let app = Application::builder()
        .application_id("dev.benz.walker")
        .flags(ApplicationFlags::HANDLES_COMMAND_LINE)
        .build();

    let hold_guard = RefCell::new(None);

    app.connect_handle_local_options(|_app, _dict| return -1);

    app.add_main_option(
        "version",
        b'v'.into(),
        OptionFlags::NONE,
        glib::OptionArg::None,
        "show version",
        None,
    );

    app.add_main_option(
        "provider",
        b'p'.into(),
        OptionFlags::NONE,
        glib::OptionArg::String,
        "exclusive provider to query",
        None,
    );

    app.connect_command_line(|app, cmd| {
        let options = cmd.options_dict();

        if options.contains("version") {
            cmd.print_literal("1.0.0-beta\n");
            return 0;
        }

        if options.contains("provider") {
            let mut provider = SWITCHER_PROVIDER.lock().unwrap();

            if let Some(val) = options.lookup_value("provider", Some(VariantTy::STRING)) {
                *provider = val.str().unwrap().to_string();
            }
        }

        app.activate();
        return 0;
    });

    app.connect_activate(move |app| {
        let visible = *IS_VISIBLE.lock().unwrap();

        let cfg = get_config();
        if cfg.close_when_open && visible {
            quit(app);
        } else {
            let provider = SWITCHER_PROVIDER.lock().unwrap();
            let p;

            if provider.is_empty() {
                p = "default".to_string();
            } else {
                p = provider.clone();
            }

            drop(provider);

            with_window(|w| {
                if let Some(placeholders) = &cfg.placeholders {
                    if let Some(placeholder) = placeholders.get(&p) {
                        w.input.set_placeholder_text(Some(&placeholder.input));
                        w.placeholder
                            .as_ref()
                            .map(|p| p.set_text(&placeholder.list));
                    }
                }

                w.input.emit_by_name::<()>("changed", &[]);
                w.input.grab_focus();

                w.window.present();
            });

            let mut visible = IS_VISIBLE.lock().unwrap();
            *visible = true;
        }
    });

    app.connect_startup(move |app| {
        *hold_guard.borrow_mut() = Some(app.hold());
        init_ui(app);
    });

    app.run()
}

fn init_ui(app: &Application) {
    if app.flags().contains(ApplicationFlags::IS_SERVICE) {
        IS_SERVICE.set(true).expect("failed to set IS_SERVICE");
    }

    println!("Waiting for elephant to start...");
    wait_for_file("/tmp/elephant.sock");
    println!("Elephant started!");

    config::load().unwrap();
    preview::load_previewers();
    setup_binds().unwrap();

    init_socket().unwrap();
    start_listening();

    let cfg = get_config();
    setup_window(app);
    setup_css(cfg.theme.clone());
    start_theme_watcher(cfg.theme.clone());

    with_window(|w| {
        setup_layer_shell(&w.window);
    });
}

fn setup_window(app: &Application) {
    let builder = Builder::new();
    let _ = builder.add_from_string(include_str!("../resources/themes/default/layout.xml"));

    let window: Window = builder
        .object("Window")
        .expect("Couldn't get 'Window' from UI file");
    let input: Entry = builder.object("Input").unwrap();
    let scroll: ScrolledWindow = builder
        .object("Scroll")
        .expect("can't get scroll from layout");
    let list: ListView = builder.object("List").expect("can't get list from layout");
    let items = ListStore::new::<QueryResponseObject>();
    let placeholder: Option<Label> = builder.object("Placeholder");
    let keybinds: Option<Label> = builder.object("Keybinds");
    let selection = SingleSelection::new(Some(items.clone()));

    let ui = WindowData {
        scroll,
        mouse_x: 0.0.into(),
        mouse_y: 0.0.into(),
        app: app.clone(),
        css_provider: setup_css_provider(),
        window,
        selection,
        list,
        input,
        items,
        placeholder,
        keybinds,
    };

    WINDOW.with(|window| {
        window
            .set(ui.clone())
            .expect("failed initializing window data");
    });

    ui.window.set_application(Some(app));
    ui.window.set_css_classes(&vec![]);

    ui.input.connect_changed(move |input| {
        disable_mouse();

        let text = input.text().to_string();

        if !text.contains(&get_config().global_argument_delimiter) {
            input_changed(text);
        }
    });

    let controller = EventControllerKey::new();
    controller.set_propagation_phase(gtk4::PropagationPhase::Capture);

    controller.connect_key_pressed(move |_, k, _, m| {
        if let Some(action) = get_bind(k, m) {
            match action.action.as_str() {
                ACTION_CLOSE => quit(&ui.app),
                ACTION_SELECT_NEXT => select_next(),
                ACTION_SELECT_PREVIOUS => select_previous(),
                ACTION_TOGGLE_EXACT => toggle_exact(),
                _ => {}
            }

            return true.into();
        }

        let handled = with_window(|w| {
            let selection = &w.selection;
            let items = &w.selection;
            if items.n_items() == 0 {
                return false;
            }

            let selected_item = match selection.selected_item() {
                Some(item) => item,
                None => return false,
            };

            let response_obj = match selected_item.downcast::<QueryResponseObject>() {
                Ok(obj) => obj,
                Err(_) => return false,
            };

            let response = response_obj.response();
            let item = match response.item.as_ref() {
                Some(item) => item,
                None => return false,
            };
            let item_clone = item.clone();

            let mut provider = item.provider.clone();

            if provider.starts_with("menus:") {
                provider = "menus".to_string();
            }

            if let Some(action) = get_provider_bind(&provider, k, m) {
                if let Err(_) = activate(response, &w.input.text().to_string(), &action.action) {
                    return false;
                }

                let after = if item_clone.identifier.starts_with("keepopen:") {
                    AFTER_CLEAR_RELOAD
                } else {
                    action.after.as_str()
                };

                let mut dont_close = false;

                if let Some(keep_open) =
                    get_modifiers().get(get_config().keep_open_modifier.as_str())
                {
                    if *keep_open == m {
                        dont_close = true
                    }
                }

                match after {
                    AFTER_CLOSE => {
                        if dont_close {
                            select_next();
                        } else {
                            quit(&ui.app);
                        }
                        return true;
                    }
                    AFTER_CLEAR_RELOAD => {
                        with_window(|w| {
                            if w.input.text().is_empty() {
                                w.input.emit_by_name::<()>("changed", &[]);
                            } else {
                                w.input.set_text("");
                            }
                        });
                    }
                    AFTER_RELOAD => crate::data::input_changed(w.input.text().to_string()),
                    _ => {}
                }

                return true;
            }

            return false;
        });

        if handled {
            return true.into();
        }

        return false.into();
    });

    if let Some(p) = ui.placeholder {
        p.set_visible(false);
    }

    let builder_copy = builder.clone();
    ui.selection.set_autoselect(true);
    ui.selection.connect_items_changed(move |s, _, _, _| {
        handle_preview(&builder_copy);

        with_window(|w| {
            if s.n_items() == 0 {
                if let Some(p) = &w.placeholder {
                    p.set_visible(true);
                }

                w.scroll.set_visible(false);

                if let Some(k) = &w.keybinds {
                    clear_keybind_hint(k);
                }
            } else {
                if let Some(p) = &w.placeholder {
                    p.set_visible(false);
                }

                w.scroll.set_visible(true);

                if let Some(k) = &w.keybinds {
                    clear_keybind_hint(k);
                }
            }
        });
    });

    let builder_copy = builder.clone();

    if let Some(preview) = builder.object::<Box>("PreviewBox") {
        preview.set_visible(false);
    }

    ui.selection.connect_selection_changed(move |_, _, _| {
        with_window(|w| {
            handle_preview(&builder_copy);
            w.list
                .scroll_to(w.selection.selected(), ListScrollFlags::NONE, None);

            set_keybind_hint();
        });
    });

    let factory = SignalListItemFactory::new();
    factory.connect_unbind(|_, item| {
        let item = item
            .downcast_ref::<gtk4::ListItem>()
            .expect("failed casting to ListItem");

        let child = item.child();
        let itembox = child
            .and_downcast_ref::<gtk4::Box>()
            .expect("failed to cast to box");

        while let Some(child) = itembox.first_child() {
            itembox.remove(&child);
        }
    });

    factory.connect_bind(|_, item| {
        let item = item
            .downcast_ref::<gtk4::ListItem>()
            .expect("failed casting to ListItem");

        let itemitem = item.item();
        let response_obj = itemitem
            .and_downcast_ref::<QueryResponseObject>()
            .expect("The item has to be a QueryResponseObject");

        let response = response_obj.response();

        if let Some(i) = response.item.as_ref() {
            match i.provider.as_str() {
                "files" => create_files_item(&item, &i),
                "symbols" => create_symbols_item(&item, &i),
                "calc" => create_calc_item(&item, &i),
                "clipboard" => create_clipboard_item(&item, &i),
                "providerlist" => create_providerlist_item(&item, &i),
                _ => create_desktopappications_item(&item, &i),
            }
        }
    });

    ui.list.set_model(Some(&ui.selection));
    ui.list.set_factory(Some(&factory));

    if get_config().disable_mouse {
        ui.list.set_can_target(false);
        ui.input.set_can_target(false);
    } else {
        ui.list.set_single_click_activate(true);

        let motion = EventControllerMotion::new();
        motion.connect_motion(|_, x, y| {
            with_window(|w| {
                if w.mouse_x.get() == 0.0 || w.mouse_y.get() == 0.0 {
                    w.mouse_x.set(x);
                    w.mouse_y.set(y);
                    return;
                }

                if x != w.mouse_x.get() || y != w.mouse_y.get() {
                    if !w.list.can_target() {
                        w.list.set_can_target(true);
                    }
                }
            });
        });

        ui.window.add_controller(motion);
    }

    ui.window.add_controller(controller);
}

fn create_desktopappications_item(l: &ListItem, i: &Item) {
    let b = Builder::new();
    let _ = b.add_from_string(include_str!("../resources/themes/default/item.xml"));
    let itembox: Box = b.object("ItemBox").expect("failed to get ItemRoot");
    itembox.add_css_class(&i.provider);
    l.set_child(Some(&itembox));

    if let Some(text) = b.object::<Label>("ItemText") {
        text.set_label(&i.text);
    }

    if let Some(text) = b.object::<Label>("ItemSubtext") {
        if i.subtext.is_empty() {
            text.set_visible(false);
        } else {
            text.set_label(&i.subtext);
        }
    }

    if let Some(image) = b.object::<Image>("ItemImage") {
        if !i.icon.is_empty() {
            if Path::new(&i.icon).is_absolute() {
                image.set_from_file(Some(&i.icon));
            } else {
                image.set_icon_name(Some(&i.icon));
            }
        }
    }
}

fn create_clipboard_item(l: &ListItem, i: &Item) {
    let b = Builder::new();
    let _ = b.add_from_string(include_str!(
        "../resources/themes/default/item_clipboard.xml"
    ));
    let itembox: Box = b.object("ItemBox").expect("failed to get ItemRoot");
    itembox.add_css_class(&i.provider);
    l.set_child(Some(&itembox));

    if let Some(text) = b.object::<Label>("ItemText") {
        text.set_label(&i.text.trim());
    }

    if let Some(text) = b.object::<Label>("ItemSubtext") {
        match DateTime::parse_from_rfc2822(&i.subtext) {
            Ok(dt) => {
                let formatted = dt
                    .format(&get_config().providers.clipboard.time_format)
                    .to_string();
                text.set_label(&formatted);
            }
            Err(_) => {
                text.set_label(&i.subtext);
            }
        }
    }

    if let Some(image) = b.object::<Picture>("ItemImage") {
        match i.type_.enum_value() {
            Ok(Type::FILE) => {
                image.set_filename(Some(&i.text));

                if let Some(text) = b.object::<Label>("ItemText") {
                    text.set_visible(false);
                }
            }
            Ok(Type::REGULAR) => {
                image.set_visible(false);
            }
            Err(_) => {
                println!("Unknown type!");
            }
        }
    }
}

fn create_symbols_item(l: &ListItem, i: &Item) {
    let b = Builder::new();
    let _ = b.add_from_string(include_str!("../resources/themes/default/item_symbols.xml"));
    let itembox: Box = b.object("ItemBox").expect("failed to get ItemRoot");
    itembox.add_css_class(&i.provider);
    l.set_child(Some(&itembox));

    if let Some(text) = b.object::<Label>("ItemText") {
        text.set_label(&i.subtext);
    }

    if let Some(text) = b.object::<Label>("ItemSubtext") {
        text.set_label(&i.subtext);
    }

    if let Some(image) = b.object::<Label>("ItemImage") {
        if !i.text.is_empty() {
            image.set_label(&i.text);
        }
    }
}

fn create_calc_item(l: &ListItem, i: &Item) {
    let b = Builder::new();
    let _ = b.add_from_string(include_str!("../resources/themes/default/item_calc.xml"));
    let itembox: Box = b.object("ItemBox").expect("failed to get ItemRoot");
    itembox.add_css_class(&i.provider);
    l.set_child(Some(&itembox));

    if let Some(text) = b.object::<Label>("ItemText") {
        text.set_label(&i.text);
    }

    if let Some(text) = b.object::<Label>("ItemSubtext") {
        text.set_label(&i.subtext);
    }

    if let Some(image) = b.object::<Image>("ItemImage") {
        if l.position() == 0 {
            if !i.icon.is_empty() {
                if Path::new(&i.icon).is_absolute() {
                    image.set_from_file(Some(&i.icon));
                } else {
                    image.set_icon_name(Some(&i.icon));
                }
            }
        } else {
            image.set_visible(false);
        }
    }
}

fn create_files_item(l: &ListItem, i: &Item) {
    let b = Builder::new();
    let _ = b.add_from_string(include_str!("../resources/themes/default/item_files.xml"));
    let itembox: Box = b.object("ItemBox").expect("failed to get ItemBox");
    itembox.add_css_class(&i.provider);
    l.set_child(Some(&itembox));

    let text = i.text.clone();

    let drag_source = DragSource::new();

    drag_source.connect_prepare(move |_, _, _| {
        let file = File::for_path(&text);
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

    itembox.add_controller(drag_source);

    if let Some(text) = b.object::<Label>("ItemText") {
        if let Ok(home) = env::var("HOME") {
            if let Some(stripped) = i.text.strip_prefix(&home) {
                text.set_label(stripped);
            }
        }
    }

    if let Some(image) = b.object::<Image>("ItemImage") {
        let file = gio::File::for_path(&i.text);
        let image_weak = Downgrade::downgrade(&image);

        file.query_info_async(
            "standard::icon",
            gio::FileQueryInfoFlags::NONE,
            glib::Priority::DEFAULT,
            gio::Cancellable::NONE,
            move |result| {
                if let Some(image) = image_weak.upgrade() {
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
    }
}

fn create_providerlist_item(l: &ListItem, i: &Item) {
    let b = Builder::new();
    let _ = b.add_from_string(include_str!(
        "../resources/themes/default/item_providerlist.xml"
    ));
    let itembox: Box = b.object("ItemBox").expect("failed to get ItemRoot");
    itembox.add_css_class(&i.provider);
    l.set_child(Some(&itembox));

    if let Some(text) = b.object::<Label>("ItemText") {
        text.set_label(&i.text);
    }

    if let Some(image) = b.object::<Image>("ItemImage") {
        if !i.icon.is_empty() {
            if Path::new(&i.icon).is_absolute() {
                image.set_from_file(Some(&i.icon));
            } else {
                image.set_icon_name(Some(&i.icon));
            }
        }
    }
}

fn quit(app: &Application) {
    if app.flags().contains(ApplicationFlags::IS_SERVICE) {
        app.active_window().unwrap().set_visible(false);

        let mut provider = SWITCHER_PROVIDER.lock().unwrap();
        *provider = "".to_string();

        glib::idle_add_once(|| {
            with_window(|w| {
                w.input.set_text("");
                w.input.emit_by_name::<()>("changed", &[]);
            });
        });

        let mut visible = IS_VISIBLE.lock().unwrap();
        *visible = false;
    } else {
        app.quit();
    }
}

fn select_next() {
    disable_mouse();

    with_window(|w| {
        let selection = &w.selection;
        if get_config().selection_wrap {
            let current = selection.selected();
            let n_items = selection.n_items();
            if n_items > 0 {
                let next = if current + 1 >= n_items {
                    0
                } else {
                    current + 1
                };
                selection.set_selected(next);
            }
        } else {
            let current = selection.selected();
            let n_items = selection.n_items();
            if current + 1 < n_items {
                selection.set_selected(current + 1);
            }
        }
    });
}

fn toggle_exact() {
    with_window(|w| {
        let i = &w.input;
        let cfg = get_config();
        if i.text().starts_with(&cfg.exact_search_prefix) {
            if let Some(t) = i.text().strip_prefix(&cfg.exact_search_prefix) {
                i.set_text(t);
                i.set_position(-1);
            }
        } else {
            i.set_text(&format!("{}{}", cfg.exact_search_prefix, i.text()));
            i.set_position(-1);
        }
    });
}

fn select_previous() {
    disable_mouse();

    with_window(|w| {
        let selection = &w.selection;
        if get_config().selection_wrap {
            let current = selection.selected();
            let n_items = selection.n_items();
            if n_items > 0 {
                let prev = if current == 0 {
                    n_items - 1
                } else {
                    current - 1
                };
                selection.set_selected(prev);
            }
        } else {
            let current = selection.selected();
            if current > 0 {
                selection.set_selected(current - 1);
            }
        }
    });
}

fn setup_css(theme: String) {
    let css: String;

    if theme == "default" {
        css = DEFAULT_STYLE.to_string();
    } else {
        let mut path = dirs::config_dir()
            .ok_or("Could not find config directory")
            .unwrap();

        path.push("walker");
        path.push("themes");
        path.push(theme);
        path.push("style.css");

        css = fs::read_to_string(&path).unwrap();
    }

    with_window(|w| {
        w.css_provider.load_from_string(&css);
    });
}

fn setup_css_provider() -> CssProvider {
    let css_provider = CssProvider::new();

    gtk4::style_context_add_provider_for_display(
        &Display::default().expect("Could not connect to a display."),
        &css_provider,
        gtk4::STYLE_PROVIDER_PRIORITY_USER,
    );

    return css_provider;
}

fn start_theme_watcher(theme_name: String) {
    let mut path = dirs::config_dir()
        .ok_or("Could not find config directory")
        .unwrap();

    path.push("walker");
    path.push("themes");
    path.push(&theme_name);
    path.push("style.css");

    thread::spawn(move || {
        let (tx, rx) = mpsc::channel();

        let mut watcher = RecommendedWatcher::new(
            move |result: Result<Event, notify::Error>| {
                if let Err(_) = tx.send(result) {
                    return;
                }
            },
            Config::default(),
        )
        .expect("Failed to create watcher");

        if let Err(_) = watcher.watch(&path, RecursiveMode::NonRecursive) {
            return;
        }

        let theme_name_for_callback = theme_name.clone();

        for result in rx {
            match result {
                Ok(_event) => {
                    let theme_name_clone = theme_name_for_callback.clone();
                    glib::idle_add_once(move || {
                        setup_css(theme_name_clone);
                    });
                }
                Err(error) => println!("Watch error: {:?}", error),
            }
        }
    });
}

fn setup_layer_shell(win: &Window) {
    if !gtk4_layer_shell::is_supported() {
        let titlebar = Box::new(gtk4::Orientation::Vertical, 0);
        win.set_titlebar(Some(&titlebar));
        return;
    }

    let cfg = get_config();
    win.init_layer_shell();
    win.set_namespace(Some("walker"));
    win.set_exclusive_zone(-1);
    win.set_layer(Layer::Overlay);
    win.set_keyboard_mode(KeyboardMode::OnDemand);

    win.set_anchor(Edge::Left, true);
    win.set_anchor(Edge::Right, true);
    win.set_anchor(Edge::Top, true);
    win.set_anchor(Edge::Bottom, true);
}

fn get_selected_item() -> Option<Item> {
    let result = with_window(|w| {
        w.selection
            .selected_item()
            .and_then(|item| item.downcast::<QueryResponseObject>().ok())
            .and_then(|obj| obj.response().item.as_ref().cloned())
    });

    return result;
}

fn handle_preview(builder: &Builder) {
    if let Some(preview) = builder.object::<Box>("Preview") {
        if let Some(item) = get_selected_item() {
            if crate::preview::has_previewer(&item.provider) {
                let builder = Builder::new();
                let _ = builder
                    .add_from_string(include_str!("../resources/themes/default/preview.xml"));

                crate::preview::handle_preview(&item.provider, &item, &preview, &builder);

                preview.set_visible(true);
            } else {
                preview.set_visible(false);
            }
        } else {
            preview.set_visible(false);
        }
    }
}

fn disable_mouse() {
    with_window(|w| {
        w.mouse_x.set(0.0);
        w.mouse_y.set(0.0);
        w.list.set_can_target(false);
    });
}

fn set_keybinds_desktopapplications(k: &Label) {
    let text = format!(
        "start: {}",
        get_config().providers.desktopapplications.start
    );
    k.set_text(&text);
}

fn set_keybinds_clipboard(k: &Label) {
    let cfg = get_config();
    let text = format!(
        "copy: {} - delete: {}",
        cfg.providers.clipboard.copy, cfg.providers.clipboard.delete
    );
    k.set_text(&text);
}

fn set_keybinds_menus(k: &Label) {
    let cfg = get_config();
    let text = format!("activate: {}", cfg.providers.menus.activate);
    k.set_text(&text);
}

fn set_keybinds_calc(k: &Label) {
    let cfg = get_config();
    let text = format!(
        "copy: {} - save: {} - delete: {}",
        cfg.providers.calc.copy, cfg.providers.calc.save, cfg.providers.calc.delete
    );
    k.set_text(&text);
}

fn set_keybinds_symbols(k: &Label) {
    let cfg = get_config();
    let text = format!("copy: {}", cfg.providers.symbols.copy,);
    k.set_text(&text);
}

fn set_keybinds_providerlist(k: &Label) {
    let cfg = get_config();
    let text = format!("select: {}", cfg.providers.providerlist.activate);
    k.set_text(&text);
}

fn set_keybinds_runner(k: &Label) {
    let cfg = get_config();
    let text = format!(
        "run: {} - run in terminal: {}",
        cfg.providers.runner.start, cfg.providers.runner.start_terminal
    );
    k.set_text(&text);
}

fn set_keybinds_files(k: &Label) {
    let cfg = get_config();
    let text = format!(
        "open: {} - open dir: {} - copy: {} - copy path: {}",
        cfg.providers.files.open,
        cfg.providers.files.open_dir,
        cfg.providers.files.copy_file,
        cfg.providers.files.copy_path
    );
    k.set_text(&text);
}

fn set_keybind_hint() {
    with_window(|w| {
        if let Some(k) = &w.keybinds {
            if let Some(item) = get_selected_item() {
                match item.provider.as_str() {
                    "desktopapplications" => set_keybinds_desktopapplications(k),
                    "files" => set_keybinds_files(k),
                    "symbols" => set_keybinds_symbols(k),
                    "calc" => set_keybinds_calc(k),
                    "runner" => set_keybinds_runner(k),
                    "providerlist" => set_keybinds_providerlist(k),
                    "clipboard" => set_keybinds_clipboard(k),
                    provider if provider.starts_with("menus:") => set_keybinds_menus(k),
                    _ => clear_keybind_hint(k),
                }
            } else {
                clear_keybind_hint(k);
            }
        }
    });
}

fn clear_keybind_hint(k: &Label) {
    k.set_text("");
}
