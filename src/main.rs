mod config;
mod data;
mod keybinds;
mod preview;

mod protos;
use gtk4::gdk::ContentProvider;
use gtk4::gio::File;
use gtk4::gio::prelude::{ApplicationCommandLineExt, FileExt, ListModelExt};
use gtk4::glib::clone::Downgrade;
use gtk4::glib::object::{CastNone, ObjectExt};
use gtk4::glib::subclass::types::ObjectSubclassIsExt;
use gtk4::glib::{OptionFlags, VariantTy};
use gtk4::prelude::{BoxExt, EditableExt, EventControllerExt, ListItemExt, SelectionModelExt};
use gtk4::{
    Box, DragSource, Entry, EventControllerMotion, Image, ListItem, ListScrollFlags,
    ScrolledWindow, gio,
};
use gtk4::{gio::ListStore, glib::object::Cast};
use protos::generated_proto::query::query_response::Item;

use config::get_config;

use std::path::Path;
use std::thread;
use std::time::Duration;
use std::{
    cell::RefCell,
    sync::{Mutex, OnceLock},
};

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
    static APP: RefCell<Option<Application>> = RefCell::new(None);
    static WINDOWS: RefCell<Option<Vec<Window>>> = RefCell::new(None);
    static STOREITEMS: RefCell<Option<ListStore>> = RefCell::new(None);
    static SELECTION: RefCell<Option<SingleSelection>> = RefCell::new(None);
    static LIST: RefCell<Option<ListView>> = RefCell::new(None);
    static INPUT: RefCell<Option<Entry>> = RefCell::new(None);
    static MOUSE_X: RefCell<Option<f64>> = RefCell::new(None);
    static MOUSE_Y: RefCell<Option<f64>> = RefCell::new(None);
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

    APP.with(|s| {
        *s.borrow_mut() = Some(app.clone());
    });

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

        if let Some(cfg) = get_config() {
            if cfg.close_when_open && visible {
                quit(app);
            } else {
                with_input(|i| {
                    i.emit_by_name::<()>("changed", &[]);
                });

                with_windows(|windows| {
                    windows[0].present();
                });

                let mut visible = IS_VISIBLE.lock().unwrap();
                *visible = true;
            }
        }
    });

    app.connect_startup(move |app| {
        *hold_guard.borrow_mut() = Some(app.hold());
        init_ui(app);
    });

    app.run()
}

fn init_ui(app: &Application) {
    IS_SERVICE.set(true).expect("failed to set IS_SERVICE");

    println!("Waiting for elephant to start...");
    wait_for_file("/tmp/elephant.sock");
    println!("Elephant started!");

    config::load().unwrap();
    preview::load_previewers();
    setup_binds().unwrap();

    init_socket().unwrap();
    start_listening();

    setup_windows(app);

    setup_css();

    with_windows(|windows| {
        windows.iter().for_each(|window| {
            setup_layer_shell(window);
        });
    });
}

fn setup_windows(app: &Application) {
    // TODO: create window per layout?
    let mut windows: Vec<Window> = Vec::new();

    let builder = Builder::new();
    let _ = builder.add_from_string(include_str!("../resources/layout_default.xml"));

    let window: Window = builder
        .object("Window")
        .expect("Couldn't get 'Window' from UI file");
    window.set_application(Some(app));
    window.set_css_classes(&vec![]);

    windows.push(window.clone());

    let app_clone = app.clone();

    let input: Entry = builder.object("Input").unwrap();

    INPUT.with(|s| {
        *s.borrow_mut() = Some(input.clone());
    });

    let input_clone = input.clone();
    let builder_copy = builder.clone();
    input.connect_changed(move |input| {
        disable_mouse();

        let text = input.text().to_string();

        if let Some(cfg) = get_config() {
            if !text.contains(&cfg.global_argument_delimiter) {
                input_changed(text);
            }
        }
    });

    let controller = EventControllerKey::new();
    controller.set_propagation_phase(gtk4::PropagationPhase::Capture);

    controller.connect_key_pressed(move |_, k, _, m| {
        if let Some(action) = get_bind(k, m) {
            match action.action.as_str() {
                ACTION_CLOSE => quit(&app_clone),
                ACTION_SELECT_NEXT => select_next(),
                ACTION_SELECT_PREVIOUS => select_previous(),
                ACTION_TOGGLE_EXACT => toggle_exact(),
                _ => {}
            }

            return true.into();
        }

        let handled = with_store(|items| {
            with_selection(|selection| {
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
                    if let Err(_) =
                        activate(response, &input_clone.text().to_string(), &action.action)
                    {
                        return false;
                    }

                    let after = if item_clone.identifier.starts_with("keepopen:") {
                        AFTER_CLEAR_RELOAD
                    } else {
                        action.after.as_str()
                    };

                    let mut dont_close = false;

                    if let Some(cfg) = get_config() {
                        if let Some(keep_open) =
                            get_modifiers().get(cfg.keep_open_modifier.as_str())
                        {
                            if *keep_open == m {
                                dont_close = true
                            }
                        }
                    }

                    match after {
                        AFTER_CLOSE => {
                            if dont_close {
                                select_next();
                            } else {
                                quit(&app_clone);
                            }
                            return true;
                        }
                        AFTER_CLEAR_RELOAD => {
                            with_input(|i| {
                                if i.text().is_empty() {
                                    i.emit_by_name::<()>("changed", &[]);
                                } else {
                                    i.set_text("");
                                }
                            });
                        }
                        AFTER_RELOAD => crate::data::input_changed(input_clone.text().to_string()),
                        _ => {}
                    }

                    return true;
                }

                return false;
            })
            .unwrap_or(false)
        })
        .unwrap_or(false);

        if handled {
            return true.into();
        }

        return false.into();
    });

    let scroll: ScrolledWindow = builder
        .object("Scroll")
        .expect("can't get scroll from layout");

    let list: ListView = builder.object("List").expect("can't get list from layout");

    LIST.with(|s| {
        *s.borrow_mut() = Some(list.clone());
    });

    let items = ListStore::new::<QueryResponseObject>();
    STOREITEMS.with(|s| {
        *s.borrow_mut() = Some(items.clone());
    });

    let placeholder: Label = builder.object("Placeholder").expect("no placeholder found");
    let selection = SingleSelection::new(Some(items.clone()));

    SELECTION.with(|s| {
        *s.borrow_mut() = Some(selection.clone());
    });

    if selection.n_items() == 0 {
        placeholder.set_visible(true);
    } else {
        placeholder.set_visible(false);
    }

    selection.set_autoselect(true);
    selection.connect_items_changed(move |s, _, _, _| {
        handle_preview(&builder_copy);

        if s.n_items() == 0 {
            placeholder.set_visible(true);
            scroll.set_visible(false);
        } else {
            placeholder.set_visible(false);
            scroll.set_visible(true);
        }
    });

    let builder_copy = builder.clone();

    if let Some(preview) = builder.object::<Box>("PreviewBox") {
        preview.set_visible(false);
    }

    selection.connect_selection_changed(move |_, _, _| {
        with_list(|list| {
            with_selection(|selection| {
                handle_preview(&builder_copy);
                list.scroll_to(selection.selected(), ListScrollFlags::NONE, None);
            });
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

    list.set_model(Some(&selection));
    list.set_factory(Some(&factory));
    list.set_single_click_activate(true);

    let motion = EventControllerMotion::new();
    motion.connect_motion(|_, x, y| {
        with_mouse_x(|mx| {
            with_mouse_y(|my| {
                if *mx == 0.0 || *my == 0.0 {
                    *mx = x;
                    *my = y;
                    return;
                }

                if x != *mx || y != *my {
                    with_list(|l| {
                        if !l.can_target() {
                            l.set_can_target(true);
                        }
                    });
                }
            });
        });
    });

    MOUSE_X.with(|s| {
        *s.borrow_mut() = Some(0.0);
    });

    MOUSE_Y.with(|s| {
        *s.borrow_mut() = Some(0.0);
    });

    window.add_controller(motion);
    window.add_controller(controller);

    WINDOWS.with(|s| {
        *s.borrow_mut() = Some(windows.clone());
    });
}

fn create_desktopappications_item(l: &ListItem, i: &Item) {
    let b = Builder::new();
    let _ = b.add_from_string(include_str!("../resources/item_default.xml"));
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
    let _ = b.add_from_string(include_str!("../resources/item_clipboard.xml"));
    let itembox: Box = b.object("ItemBox").expect("failed to get ItemRoot");
    itembox.add_css_class(&i.provider);
    l.set_child(Some(&itembox));

    if let Some(text) = b.object::<Label>("ItemText") {
        text.set_label(&i.text.trim());
    }

    if let Some(text) = b.object::<Label>("ItemSubtext") {
        text.set_label(&i.subtext);
    }

    if let Some(image) = b.object::<Image>("ItemImage") {
        match i.type_.enum_value() {
            Ok(Type::FILE) => {
                image.set_from_file(Some(&i.text));

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
    let _ = b.add_from_string(include_str!("../resources/item_symbols.xml"));
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
    let _ = b.add_from_string(include_str!("../resources/item_calc.xml"));
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
    let _ = b.add_from_string(include_str!("../resources/item_files.xml"));
    let itembox: Box = b.object("ItemBox").expect("failed to get ItemBox");
    itembox.add_css_class(&i.provider);
    l.set_child(Some(&itembox));

    let drag_source = DragSource::new();
    let text = i.text.clone();

    drag_source.connect_prepare(move |_, _, _| {
        let file = File::for_path(&text);
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

    itembox.add_controller(drag_source);

    if let Some(text) = b.object::<Label>("ItemText") {
        text.set_label(&i.text);
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
    let _ = b.add_from_string(include_str!("../resources/item_providerlist.xml"));
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
            with_input(|i| {
                i.set_text("");
                i.emit_by_name::<()>("changed", &[]);
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

    with_selection(|selection| {
        if let Some(cfg) = get_config() {
            if cfg.selection_wrap {
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
        }
    });
}

fn toggle_exact() {
    with_input(|i| {
        if let Some(cfg) = get_config() {
            if i.text().starts_with(&cfg.exact_search_prefix) {
                if let Some(t) = i.text().strip_prefix(&cfg.exact_search_prefix) {
                    i.set_text(t);
                    i.set_position(-1);
                }
            } else {
                i.set_text(&format!("{}{}", cfg.exact_search_prefix, i.text()));
                i.set_position(-1);
            }
        }
    });
}

fn select_previous() {
    disable_mouse();

    with_selection(|selection| {
        if let Some(cfg) = get_config() {
            if cfg.selection_wrap {
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
        }
    });
}

fn setup_css() {
    let css_provider = CssProvider::new();
    css_provider.load_from_string(include_str!("../resources/style_default.css"));

    gtk4::style_context_add_provider_for_display(
        &Display::default().expect("Could not connect to a display."),
        &css_provider,
        gtk4::STYLE_PROVIDER_PRIORITY_APPLICATION,
    );
}

fn setup_layer_shell(win: &Window) {
    if !gtk4_layer_shell::is_supported() {
        let titlebar = Box::new(gtk4::Orientation::Vertical, 0);
        win.set_titlebar(Some(&titlebar));
        return;
    }

    if let Some(cfg) = get_config() {
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
}

pub fn with_selection<F, R>(f: F) -> Option<R>
where
    F: FnOnce(&SingleSelection) -> R,
{
    SELECTION.with(|s| s.borrow().as_ref().map(f))
}

pub fn with_list<F, R>(f: F) -> Option<R>
where
    F: FnOnce(&ListView) -> R,
{
    LIST.with(|s| s.borrow().as_ref().map(f))
}

pub fn with_store<F, R>(f: F) -> Option<R>
where
    F: FnOnce(&ListStore) -> R,
{
    STOREITEMS.with(|s| s.borrow().as_ref().map(f))
}

pub fn with_app<F, R>(f: F) -> Option<R>
where
    F: FnOnce(&Application) -> R,
{
    APP.with(|s| s.borrow().as_ref().map(f))
}

pub fn with_input<F, R>(f: F) -> Option<R>
where
    F: FnOnce(&Entry) -> R,
{
    INPUT.with(|s| s.borrow().as_ref().map(f))
}

pub fn with_windows<F, R>(f: F) -> Option<R>
where
    F: FnOnce(&Vec<Window>) -> R,
{
    WINDOWS.with(|s| s.borrow().as_ref().map(f))
}

pub fn with_mouse_x<F, R>(f: F) -> Option<R>
where
    F: FnOnce(&mut f64) -> R,
{
    MOUSE_X.with(|s| {
        let mut borrow = s.borrow_mut();
        borrow.as_mut().map(f)
    })
}

pub fn with_mouse_y<F, R>(f: F) -> Option<R>
where
    F: FnOnce(&mut f64) -> R,
{
    MOUSE_Y.with(|s| {
        let mut borrow = s.borrow_mut();
        borrow.as_mut().map(f)
    })
}

fn get_selected_item() -> Option<Item> {
    let result = with_selection(|selection| {
        selection
            .selected_item()?
            .downcast::<QueryResponseObject>()
            .ok()?
            .response()
            .item
            .as_ref()
            .cloned()
    });

    result.flatten()
}

fn handle_preview(builder: &Builder) {
    if let Some(preview) = builder.object::<Box>("Preview") {
        if let Some(item) = get_selected_item() {
            if crate::preview::has_previewer(&item.provider) {
                let builder = Builder::new();
                let _ = builder.add_from_string(include_str!("../resources/preview_default.xml"));

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
    with_mouse_x(|x| {
        with_mouse_y(|y| {
            with_list(|l| {
                *y = 0.0;
                *x = 0.0;
                l.set_can_target(false);
            })
        })
    });
}
