use crate::{
    GLOBAL_DMENU_SENDER, QueryResponseObject,
    config::get_config,
    data::{activate, input_changed, set_state},
    keybinds::{
        ACTION_CLOSE, ACTION_QUICK_ACTIVATE, ACTION_RESUME_LAST_QUERY, ACTION_SELECT_DOWN,
        ACTION_SELECT_LEFT, ACTION_SELECT_NEXT, ACTION_SELECT_PAGE_DOWN, ACTION_SELECT_PAGE_UP,
        ACTION_SELECT_PREVIOUS, ACTION_SELECT_RIGHT, ACTION_SELECT_UP, ACTION_TOGGLE_EXACT, Action,
        AfterAction, get_bind, get_provider_bind, get_provider_global_bind,
    },
    protos::generated_proto::query::QueryResponse,
    providers::{PROVIDERS, Provider},
    renderers::create_item,
    send_message,
    state::{
        get_current_prefix, get_error, get_initial_height, get_initial_max_height,
        get_initial_max_width, get_initial_min_height, get_initial_min_width,
        get_initial_placeholder, get_initial_width, get_last_query, get_prefix_provider,
        get_provider, get_theme, get_theme_grid_columns, is_connected, is_dmenu,
        is_dmenu_exit_after, is_dmenu_keep_open, is_emergency, is_grid, is_no_hints, is_service,
        query, set_async_after, set_current_prefix, set_current_set, set_dmenu_current,
        set_dmenu_exit_after, set_dmenu_keep_open, set_error, set_hide_qa, set_index,
        set_initial_height, set_initial_max_height, set_initial_max_width, set_initial_min_height,
        set_initial_min_width, set_initial_placeholder, set_initial_width, set_input_only,
        set_is_dmenu, set_is_grid, set_is_stay_open_explicit_provider, set_is_visible,
        set_last_query, set_no_hints, set_no_search, set_param_close, set_parameter_height,
        set_parameter_max_height, set_parameter_max_width, set_parameter_min_height,
        set_parameter_min_width, set_parameter_width, set_placeholder, set_provider, set_query,
        set_theme,
    },
    theme::{Theme, setup_layer_shell, with_themes},
};
use gtk4::{
    Application, Builder, Button, CustomFilter, Entry, EventControllerKey, EventControllerMotion,
    FilterListModel, GestureClick, Label, PropagationPhase, ScrolledWindow, SignalListItemFactory,
    SingleSelection, Window,
    prelude::{BoxExt, ButtonExt},
};
use gtk4::{Box, ListScrollFlags};
use gtk4::{
    CssProvider,
    prelude::{EditableExt, EventControllerExt, ListItemExt, SelectionModelExt},
};
use gtk4::{
    GridView,
    glib::object::{CastNone, ObjectExt},
};
use gtk4::{gdk, prelude::WidgetExt};
use gtk4::{gio::ListStore, glib::object::Cast};
use gtk4::{
    gio::prelude::{ApplicationExt, ListModelExt},
    prelude::GtkApplicationExt,
};
use gtk4::{
    glib::Object,
    prelude::{EntryExt, GtkWindowExt},
};
use std::{
    cell::{Cell, OnceCell, RefCell},
    collections::HashMap,
    process::{self, Command, Stdio},
};

thread_local! {
    pub static WINDOWS: OnceCell<HashMap<String, WindowData>> = const { OnceCell::new() };
    pub static CSS_PROVIDER: RefCell<Option<CssProvider>> = const { RefCell::new(None) };
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

#[derive(Debug)]
pub struct WindowData {
    pub sid: Option<gdk::glib::SignalHandlerId>,
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
    pub elephant_hint: Label,
    pub keybinds: gtk4::Box,
    pub global_keybinds: gtk4::Box,
    pub item_keybinds: gtk4::Box,
    pub scroll: ScrolledWindow,
    pub search_container: Option<gtk4::Box>,
    pub preview_container: Option<gtk4::Box>,
    pub content_container: gtk4::Box,
    pub box_wrapper: gtk4::Box,
    pub error: gtk4::Label,
}

pub fn with_window<F, R>(f: F) -> R
where
    F: FnOnce(&WindowData) -> R,
{
    WINDOWS.with(|windows| {
        let windows_map = windows.get().unwrap();
        let theme = get_theme();

        windows_map
            .get(&theme)
            .or_else(|| windows_map.get("default"))
            .map(f)
            .unwrap_or_else(|| {
                println!("default theme not found");
                process::exit(130);
            })
    })
}

pub fn setup_theme_window(app: &Application, val: &Theme) -> Result<WindowData, String> {
    let builder = Builder::new();
    let _ = builder.add_from_string(&val.layout);

    let window: Window = match builder.object("Window") {
        Some(w) => w,
        None => return Err("missing 'Window' object".into()),
    };

    let scroll: ScrolledWindow = match builder.object("Scroll") {
        Some(w) => w,
        None => return Err("missing 'Scroll' object".into()),
    };

    let list: GridView = match builder.object("List") {
        Some(w) => w,
        None => return Err("missing 'List' object".into()),
    };

    let elephant_hint: Label = match builder.object("ElephantHint") {
        Some(w) => w,
        None => return Err("missing 'ElephantHint' object".into()),
    };

    let error: Label = match builder.object("Error") {
        Some(w) => w,
        None => return Err("missing 'Error' object".into()),
    };

    let box_wrapper: gtk4::Box = match builder.object("BoxWrapper") {
        Some(w) => w,
        None => return Err("missing 'BoxWrapper' object".into()),
    };

    let content_container: gtk4::Box = match builder.object("ContentContainer") {
        Some(w) => w,
        None => return Err("missing 'ContentContainer' object".into()),
    };

    let keybinds: gtk4::Box = match builder.object("Keybinds") {
        Some(w) => w,
        None => return Err("missing 'Keybinds' object".into()),
    };

    let global_keybinds: gtk4::Box = match builder.object("GlobalKeybinds") {
        Some(w) => w,
        None => return Err("missing 'GlobalKeybinds' object".into()),
    };

    let item_keybinds: gtk4::Box = match builder.object("ItemKeybinds") {
        Some(w) => w,
        None => return Err("missing 'ItemKeybinds' object".into()),
    };

    let input: Option<Entry> = builder.object("Input");
    let placeholder: Option<Label> = builder.object("Placeholder");

    let filter = CustomFilter::new({
        move |entry| {
            let item = entry.downcast_ref::<QueryResponseObject>().unwrap();

            let q = query();

            if (is_dmenu() || is_emergency()) && !q.is_empty() {
                let f = 18 * q.len();

                if item.dmenu_score() < f as u32 {
                    return false;
                }
            }

            true
        }
    });

    let items = ListStore::new::<QueryResponseObject>();
    let filter_model = FilterListModel::new(Some(items.clone()), Some(filter.clone()));
    let selection = SingleSelection::new(Some(filter_model.clone()));
    let search_container: Option<Box> = builder.object("SearchContainer");
    let preview_container: Option<Box> = builder.object("Preview");

    let mut ui = WindowData {
        error,
        sid: None,
        box_wrapper,
        preview_container,
        elephant_hint,
        content_container,
        search_container,
        builder,
        preview_builder: std::cell::RefCell::new(None),
        scroll,
        mouse_x: 0.0.into(),
        mouse_y: 0.0.into(),
        app: app.clone(),
        window,
        selection,
        list,
        input,
        items,
        placeholder,
        keybinds,
        global_keybinds,
        item_keybinds,
    };

    if let Some(p) = &ui.preview_container {
        p.set_visible(false);
    }

    setup_window_behavior(&ui, app);

    if let Some(input) = &ui.input {
        ui.sid = Some(setup_input_handling(input))
    }

    setup_keyboard_handling(&ui);
    setup_list_behavior(&ui);
    setup_mouse_handling(&ui);

    ui.window.set_application(Some(app));
    ui.window.set_css_classes(&[]);

    setup_layer_shell(&ui.window);

    Ok(ui)
}

pub fn setup_window(app: &Application) {
    let mut windows: HashMap<String, WindowData> = HashMap::new();

    with_themes(|t| {
        for (key, val) in t {
            match setup_theme_window(app, val) {
                Ok(res) => {
                    windows.insert(key.to_string(), res);
                }
                Err(error) => set_error(format!("Theme [{key}]: {error}")),
            }
        }
    });

    WINDOWS.with(|s| s.set(windows).expect("failed initializing windows"));
}

pub fn check_error() {
    with_window(|w| {
        if !get_error().is_empty() {
            w.error.set_text(&get_error());
            w.error.set_visible(true);
        } else {
            w.error.set_text("");
            w.error.set_visible(false);
        }
    });
}

fn setup_window_behavior(ui: &WindowData, app: &Application) {
    if let Some(p) = &ui.placeholder {
        p.set_visible(false);
    }

    ui.selection.set_autoselect(true);

    ui.selection.connect_items_changed(move |_, _, _, _| {
        if is_dmenu() {
            handle_changed_items();
        }
    });

    if let Some(preview) = ui.builder.object::<Box>("PreviewBox") {
        preview.set_visible(false);
    }

    ui.selection.connect_selection_changed(move |_, _, _| {
        with_window(|w| {
            handle_preview();

            w.list
                .scroll_to(w.selection.selected(), ListScrollFlags::NONE, None);

            set_keybind_hint();
        });
    });

    let app_copy = app.clone();

    ui.list.connect_activate(move |_, _| {
        activate_default(&app_copy);
    });

    let config = get_config();

    ui.list
        .set_single_click_activate(config.single_click_activation);

    let app_copy = app.clone();

    if config.click_to_close && !config.disable_mouse {
        let gc = GestureClick::new();
        gc.set_propagation_phase(PropagationPhase::Target);
        gc.connect_pressed(move |_, _, _, _| {
            quit(&app_copy, true);
        });

        ui.window.add_controller(gc);
    }
}

fn activate_default(app: &Application) {
    with_window(|w| {
        let query = w.input.as_ref().map(Entry::text).unwrap_or_default();

        if let Some(item) = get_selected_item() {
            let provider = item.provider.clone();

            let providers = PROVIDERS.get().unwrap();
            let p = providers.get(&provider).unwrap();

            let actions = p.get_keybind_hint(&item.actions);

            let action = if item.actions.len() == 1 {
                actions
                    .iter()
                    .find(|a| a.action == *item.actions.first().unwrap())
                    .unwrap()
            } else {
                actions.iter().find(|a| a.default.unwrap_or(false)).unwrap()
            };

            activate(get_selected_query_response(), &provider, &query, action);

            let after = action.after.as_ref().unwrap_or(&AfterAction::Close).clone();
            handle_after(&after, app, query.to_string());
        }
    });
}

fn setup_input_handling(input: &Entry) -> gdk::glib::SignalHandlerId {
    input.connect_changed(move |input| {
        disable_mouse();

        let text = input.text().to_string();

        let cfg = get_config();

        let delimiter = if let Some(item) = get_selected_query_response() {
            if let Some(d) = cfg.providers.argument_delimiter.get(&item.item.provider) {
                d
            } else {
                &cfg.global_argument_delimiter
            }
        } else {
            &cfg.global_argument_delimiter
        };

        if !text.contains(delimiter) {
            input_changed(&text);
        }
    })
}

fn setup_keyboard_handling(ui: &WindowData) {
    let controller = EventControllerKey::new();
    controller.set_propagation_phase(gtk4::PropagationPhase::Capture);

    let app = ui.app.clone();

    controller.connect_key_pressed(move |_, mut k, _, m| {
        if k == gdk::Key::KP_Enter {
            k = gdk::Key::Return;
        }

        let handled = with_window(|w| {
            if !is_connected() && !is_dmenu() {
                if let Some(action) = get_bind(k, m, is_grid())
                    && action.action == ACTION_CLOSE
                {
                    quit(&app, true);
                }

                if !is_emergency() {
                    return true;
                }
            }

        if !is_connected() && let Some(e) = &get_config().emergencies && k == gdk::Key::Return
            && let Some(i) = get_selected_item()
                && let Some(item) = e.iter().find(|e| e.text == i.text) {
                    Command::new("sh")
                        .arg("-c")
                        .arg(&item.command)
                        .stdin(Stdio::null())  // Detach from stdin
                        .stdout(Stdio::null()) // Detach from stdout
                        .stderr(Stdio::null()) // Detach from stderr
                        .spawn().expect("failed to run emergency command");

                    quit(&app, true);
                };

            let selection = &w.selection;


            if is_dmenu() && k == gdk::Key::Return && selection.selected_item().is_none() {
                let mut text = w
                    .input
                    .as_ref()
                    .map(Entry::text)
                    .unwrap_or_default()
                    .to_string();

                if text.is_empty() {
                    text = "CNCLD".to_string();
                }

                if is_service() {
                    send_message(text);
                } else {
                    println!("{text}");
                }

                quit(&app, false);
                return true;
            }

            let mut keybind_action: Option<Action> = None;

            let mut provider = if !get_provider().is_empty() {
                get_provider()
            } else {
                get_prefix_provider()
            };

            let mut after: Option<AfterAction> = None;

            if !provider.is_empty()
                && let Some(action) = get_provider_global_bind(&provider, k, m)
            {
                keybind_action = Some(action.clone());
                after = Some(action.after.unwrap_or(AfterAction::Close));

                if action.action.starts_with("set:")
                    && let Some((_, set)) = action.action.split_once(":")
                {
                    set_current_set(set.to_string());
                    set_provider(String::new());
                }

                if action.action.starts_with("provider:")
                    && let Some((_, provider)) = action.action.split_once(":")
                {
                    set_provider(provider.to_string());
                }
            }

            let mut response: Option<QueryResponse> = None;

            if keybind_action.is_none()
                && let Some(r) = get_selected_query_response()
            {
                response = Some(r.clone());
                let Some(item) = r.item.as_ref() else {
                    return false;
                };

                provider = item.provider.clone();

                if let Some(action) = get_provider_bind(&item.provider, k, m, &item.actions) {
                    after = Some(action.after.as_ref().unwrap_or(&AfterAction::Close).clone());
                    keybind_action = Some(action);
                }

                if is_dmenu_keep_open() && !is_dmenu_exit_after() {
                    after = Some(AfterAction::Nothing)
                }
            }


            let is_grid = is_grid();

            if keybind_action.is_none()
                || (keybind_action.as_ref().unwrap().action == "menus:parent"
                    && !get_prefix_provider().is_empty())
            {
                if let Some(action) = get_bind(k, m, is_grid) {
                    match action.action.as_str() {
                        ACTION_CLOSE => quit(&app, true),
                        ACTION_SELECT_NEXT if !is_grid => select_next(),
                        ACTION_SELECT_PREVIOUS if !is_grid => select_previous(),
                        ACTION_SELECT_LEFT if is_grid => select_previous(),
                        ACTION_SELECT_RIGHT if is_grid => select_next(),
                        ACTION_SELECT_UP if is_grid => select_up(),
                        ACTION_SELECT_DOWN if is_grid => select_down(),
                        ACTION_TOGGLE_EXACT => toggle_exact(),
                        ACTION_RESUME_LAST_QUERY => resume_last_query(),
                        ACTION_SELECT_PAGE_DOWN => select_page_down(),
                        ACTION_SELECT_PAGE_UP => select_page_up(),
                        action if action.starts_with(ACTION_QUICK_ACTIVATE) => {
                            if let Some((_, after)) = action.split_once(":") {
                                let i: u32 = after.parse().unwrap();
                                quick_activate(&app, i)
                            }
                        }
                        _ => (),
                    }

                    return true;
                }

                return false;
            }

            let query = w.input.as_ref().map(Entry::text).unwrap_or_default();

            if let Some(a) = keybind_action {
                if !a.action.starts_with("set:") && !a.action.starts_with("provider:") {
                    if provider == "windows" && get_config().force_keyboard_focus {
                        println!("windows might potentially not be focused if force_keyboard_focus is true");
                    }

                    activate(response, provider.as_str(), &query, &a);
                }
            } else {
                return false;
            }

            if let Some(a) = after {
                handle_after(&a, &app, query.to_string());
            }

            true
        });

        if handled {
            return true.into();
        }

        false.into()
    });

    ui.window.add_controller(controller);
}

fn handle_after(a: &AfterAction, app: &Application, query: String) {
    match a {
        AfterAction::Close => {
            quit(app, false);
        }
        AfterAction::KeepOpen => {
            select_next();
        }
        AfterAction::ClearReload => {
            with_window(|w| {
                if let Some(input) = &w.input {
                    if input.text().is_empty() {
                        input.emit_by_name::<()>("changed", &[]);
                    } else {
                        set_input_text(&get_current_prefix());
                    }
                }
            });
        }
        AfterAction::AsyncReload => set_async_after(Some(AfterAction::AsyncReload)),
        AfterAction::AsyncClearReload => set_async_after(Some(AfterAction::AsyncClearReload)),
        AfterAction::Reload => {
            crate::data::input_changed(&query);
        }
        _ => {}
    }
}

fn setup_list_behavior(ui: &WindowData) {
    let factory = SignalListItemFactory::new();

    factory.connect_unbind(|_, item| {
        let item = item
            .downcast_ref::<gtk4::ListItem>()
            .expect("failed casting to ListItem");

        item.set_child(None::<&gtk4::Widget>);
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

        with_themes(|t| {
            if let Some(i) = response.item.as_ref() {
                if let Some(theme) = t.get(&get_theme()) {
                    create_item(item, i, theme);
                } else {
                    create_item(item, i, t.get("default").unwrap());
                }
            }
        });
    });

    ui.list.set_model(Some(&ui.selection));
    ui.list.set_factory(Some(&factory));
}

fn setup_mouse_handling(ui: &WindowData) {
    if get_config().disable_mouse {
        ui.list.set_can_target(false);

        if let Some(input) = &ui.input {
            input.set_can_target(false);
        }
        return;
    }

    let motion = EventControllerMotion::new();
    motion.connect_motion(|_, x, y| {
        with_window(|w| {
            if w.mouse_x.get() == 0.0 || w.mouse_y.get() == 0.0 {
                w.mouse_x.set(x);
                w.mouse_y.set(y);
                return;
            }

            if (x != w.mouse_x.get() || y != w.mouse_y.get()) && !w.list.can_target() {
                w.list.set_can_target(true);
            }
        });
    });

    ui.window.add_controller(motion);
}

fn reset_provider_states() {
    if is_connected() && PROVIDERS.get().unwrap().contains_key("clipboard") {
        set_state("clipboard", "show_combined");
    }

    if is_connected() && PROVIDERS.get().unwrap().contains_key("todo") {
        set_state("todo", "search");
    }

    if is_connected() && PROVIDERS.get().unwrap().contains_key("bookmarks") {
        set_state("bookmarks", "search");
    }
}

pub fn quit(app: &Application, cancelled: bool) {
    reset_provider_states();

    if GLOBAL_DMENU_SENDER.read().unwrap().is_some() {
        send_message("CNCLD".to_string());
    }

    if !app
        .flags()
        .contains(gtk4::gio::ApplicationFlags::IS_SERVICE)
    {
        if cancelled {
            process::exit(130);
        }

        app.quit();
        return;
    }

    app.active_window().unwrap().set_visible(false);

    with_window(|w| {
        while let Some(preview) = w.builder.object::<Box>("Preview")
            && let Some(child) = preview.first_child()
        {
            child.unparent();
        }

        w.preview_builder.borrow_mut().take();
    });

    // Clear all preview caches
    crate::preview::clear_all_caches();

    set_current_prefix(String::new());
    set_provider(String::new());
    set_is_stay_open_explicit_provider(false);
    set_parameter_height(None);
    set_parameter_width(None);
    set_parameter_min_height(None);
    set_parameter_min_width(None);
    set_parameter_max_height(None);
    set_parameter_max_width(None);
    set_no_search(false);
    set_no_hints(false);
    set_placeholder(String::new());
    set_is_visible(false);
    set_dmenu_current(0);
    set_is_dmenu(false);
    set_input_only(false);
    set_param_close(false);
    set_hide_qa(false);
    set_query("");
    set_current_set(String::new());
    set_index(false);

    if is_dmenu_exit_after() {
        set_dmenu_exit_after(false);
        set_dmenu_keep_open(false);
    }

    gtk4::glib::idle_add_once(|| {
        with_window(|w| {
            if let Some(input) = &w.input {
                set_last_query(input.text().to_string());

                if !get_initial_placeholder().is_empty() {
                    input.set_placeholder_text(Some(&get_initial_placeholder()));
                    set_initial_placeholder(String::new());
                }
            };

            if let Some(search_container) = &w.search_container {
                search_container.set_visible(true);
            }

            w.item_keybinds.set_visible(true);
            w.keybinds.set_visible(true);

            w.content_container.set_visible(true);

            if let Some(val) = get_initial_height() {
                w.box_wrapper.set_height_request(val);
                set_initial_height(None);
            }

            if let Some(val) = get_initial_width() {
                w.box_wrapper.set_width_request(val);
                set_initial_width(None);
            }

            if let Some(val) = get_initial_max_width() {
                w.scroll.set_max_content_width(val);
                set_initial_max_width(None);
            }

            if let Some(val) = get_initial_min_width() {
                w.scroll.set_min_content_width(val);
                set_initial_min_width(None);
            }

            if let Some(val) = get_initial_max_height() {
                w.scroll.set_max_content_height(val);
                set_initial_max_height(None);
            }

            if let Some(val) = get_initial_min_height() {
                w.scroll.set_min_content_height(val);
                set_initial_min_height(None);
            }

            set_theme(get_config().theme.clone());
        });
    });
}

pub fn select_up() {
    with_window(|w| {
        let selection = &w.selection;
        let current = selection.selected();

        if current < w.list.max_columns() {
            return;
        }

        selection.set_selected(current - w.list.max_columns());
    });
}

pub fn select_down() {
    with_window(|w| {
        let selection = &w.selection;
        let current = selection.selected();
        let n_items = selection.n_items();

        if current + w.list.max_columns() >= n_items {
            selection.set_selected(n_items - 1);
            return;
        }

        selection.set_selected(current + w.list.max_columns());
    });
}

pub fn select_next() {
    disable_mouse();

    with_window(|w| {
        let selection = &w.selection;
        if !get_config().selection_wrap {
            let current = selection.selected();
            let n_items = selection.n_items();
            if current + 1 < n_items {
                selection.set_selected(current + 1);
            }
            return;
        }

        let current = selection.selected();
        let n_items = selection.n_items();
        if n_items == 0 {
            return;
        }

        let next = if current + 1 >= n_items {
            0
        } else {
            current + 1
        };
        selection.set_selected(next);
    });
}

pub fn select_previous() {
    disable_mouse();

    with_window(|w| {
        let selection = &w.selection;
        if !get_config().selection_wrap {
            let current = selection.selected();
            if current > 0 {
                selection.set_selected(current - 1);
            }
            return;
        }

        let current = selection.selected();
        let n_items = selection.n_items();
        if n_items == 0 {
            return;
        }

        let prev = if current == 0 {
            n_items - 1
        } else {
            current - 1
        };
        selection.set_selected(prev);
    });
}

fn quick_activate(app: &Application, i: u32) {
    with_window(|w| {
        w.selection.set_selected(i);
    });

    activate_default(app);
}

pub fn resume_last_query() {
    if !get_last_query().is_empty() {
        set_input_text(&get_last_query());
    }
}

pub fn toggle_exact() {
    with_window(|w| {
        let Some(i) = &w.input else { return };

        let cfg = get_config();
        let prefix = get_current_prefix();
        let input_text = i.text();

        let text = input_text.strip_prefix(&prefix).unwrap_or(&input_text);

        let toggled = match text.strip_prefix(&cfg.exact_search_prefix) {
            Some(t) => t.to_string(),
            None => format!("{}{}", cfg.exact_search_prefix, text),
        };

        set_input_text(&format!("{}{}", prefix, toggled));
    });
}

fn disable_mouse() {
    with_window(|w| {
        w.mouse_x.set(0.0);
        w.mouse_y.set(0.0);
        w.list.set_can_target(false);
    });
}

pub fn get_selected_item() -> Option<crate::protos::generated_proto::query::query_response::Item> {
    with_window(|w| {
        w.selection
            .selected_item()
            .map(Object::downcast::<QueryResponseObject>)
            .and_then(Result::ok)
            .and_then(|obj| obj.response().item.into_option())
    })
}

pub fn get_selected_query_response() -> Option<crate::protos::generated_proto::query::QueryResponse>
{
    with_window(|w| {
        w.selection
            .selected_item()
            .map(Object::downcast::<QueryResponseObject>)
            .and_then(Result::ok)
            .map(|obj| obj.response())
    })
}

pub fn handle_preview() {
    with_window(|w| {
        let Some(preview) = w.builder.object::<Box>("Preview") else {
            return;
        };

        let Some(item) = get_selected_item() else {
            preview.set_visible(false);
            return;
        };

        // Check if preview should be shown (not in ignore list and has preview content)
        let config = crate::config::get_config();
        if config.providers.ignore_preview.contains(&item.provider) {
            preview.set_visible(false);
            return;
        }

        // Only show preview if there's preview content or preview_type specified
        if item.preview.is_empty() && item.preview_type.is_empty() {
            preview.set_visible(false);
            return;
        }

        let builder = {
            let mut preview_builder = w.preview_builder.borrow_mut();
            if preview_builder.is_none() {
                let builder = Builder::new();
                let _ = builder
                    .add_from_string(include_str!("../../resources/themes/default/preview.xml"));
                *preview_builder = Some(builder);
            }
            preview_builder.as_ref().unwrap().clone()
        };

        crate::preview::handle_preview(&item, &preview, &builder);
    });
}

pub fn clear_global_keybind_hints() {
    gtk4::glib::idle_add_once(move || {
        with_window(|w| {
            while let Some(child) = w.global_keybinds.first_child() {
                w.global_keybinds.remove(&child);
            }
        });
    });
}

pub fn set_global_keybind_hints(actions: Vec<String>, provider: String) {
    gtk4::glib::idle_add_once(move || {
        with_window(|w| {
            let k = &w.global_keybinds;

            while let Some(child) = k.first_child() {
                k.remove(&child);
            }

            let providers = PROVIDERS.get().unwrap();

            while let Some(child) = k.first_child() {
                k.remove(&child);
            }

            if let Some(p) = providers.get(&provider) {
                generate_hints(p, &actions, &k);
            }
        });
    });
}

pub fn set_keybind_hint() {
    with_window(|w| {
        let k = &w.item_keybinds;
        let cfg = get_config();

        if is_no_hints()
            || cfg.hide_action_hints
            || (is_dmenu() && cfg.hide_action_hints_dmenu)
            || w.items.n_items() == 0
        {
            w.keybinds.set_visible(false);
        } else {
            w.keybinds.set_visible(true);
        }

        while let Some(child) = k.first_child() {
            k.remove(&child);
        }

        let actions;
        let provider;

        if let Some(item) = get_selected_item() {
            actions = item.actions;
            provider = item.provider;
        } else {
            actions = Vec::new();
            provider = if !get_provider().is_empty() {
                get_provider()
            } else {
                get_prefix_provider()
            };
        };

        let providers = PROVIDERS.get().unwrap();

        if let Some(p) = providers.get(&provider) {
            generate_hints(p, &actions, k);
        } else if provider.starts_with("menus:")
            && let Some(p) = providers.get("menus")
        {
            generate_hints(p, &actions, k);
        } else if providers.get("menus").is_some() {
            while let Some(child) = k.first_child() {
                k.remove(&child);
            }
        }
    });
}

pub fn generate_hints(p: &std::boxed::Box<dyn Provider>, actions: &[String], k: &gtk4::Box) {
    let mut hints = p.get_keybind_hint(actions);

    if !get_prefix_provider().is_empty() {
        hints.retain(|a| a.action != "menus:parent");
    }

    if get_config().debug {
        println!(
            "configured actions [{}]: {}",
            p.get_name(),
            hints
                .iter()
                .map(|h| h.action.clone())
                .collect::<Vec<String>>()
                .join(",")
        );

        println!(
            "available actions [{}]: {}",
            p.get_name(),
            actions.join(",")
        );
    }

    hints.iter().for_each(|h| {
        with_themes(|t| {
            let theme = t
                .get(&get_theme())
                .or_else(|| {
                    set_error(format!(
                        "theme '{}' not found. falling back to default",
                        &get_theme()
                    ));
                    t.get("default")
                })
                .unwrap();
            let b = Builder::new();

            let _ = b.add_from_string(&theme.keybind);

            let container = match b.object::<Box>("Keybind") {
                Some(res) => Some(res),
                None => {
                    set_error("Theme: missing 'Keybind' object".to_string());
                    None
                }
            };

            let bind = match b.object::<Label>("KeybindBind") {
                Some(res) => Some(res),
                None => {
                    set_error("Theme: missing 'KeybindBind' object".to_string());
                    None
                }
            };

            let button = match b.object::<Button>("KeybindButton") {
                Some(res) => Some(res),
                None => {
                    set_error("Theme: missing 'KeybindBind' object".to_string());
                    None
                }
            };

            let h_clone = h.clone();

            if let Some(b) = button {
                b.connect_clicked(move |_| {
                    with_window(|w| {
                        let query = w.input.as_ref().map(Entry::text).unwrap_or_default();

                        if let Some(item) = get_selected_item() {
                            let provider = item.provider.clone();

                            activate(get_selected_query_response(), &provider, &query, &h_clone);

                            let after = h_clone
                                .after
                                .as_ref()
                                .unwrap_or(&AfterAction::Close)
                                .clone();
                            handle_after(&after, &w.app, query.to_string());
                        }
                    });
                });
            };

            let label: Option<Label> = b.object("KeybindLabel");

            check_error();

            if let Some(b) = bind {
                b.set_text(h.bind.as_ref().unwrap())
            }

            if let Some(l) = label {
                if let Some(label) = &h.label {
                    l.set_text(label);
                } else {
                    l.set_text(&h.action);
                }
            }

            if let Some(c) = container {
                k.append(&c);
            }
        });
    });
}

pub fn set_input_text(text: &str) {
    with_window(|w| {
        if let Some(input) = &w.input {
            let sid = w.sid.as_ref().unwrap();
            input.block_signal(sid);
            input.set_text(text);
            input.unblock_signal(sid);
            input.set_position(-1);
            input.emit_by_name::<()>("changed", &[]);
        }
    })
}

pub fn select_page_down() {
    disable_mouse();

    with_window(|w| {
        let selection = &w.selection;
        let current = selection.selected();
        let n_items = selection.n_items();
        let jump = get_config().page_jump_items;

        if n_items == 0 {
            return;
        }

        let next = if current + jump < n_items {
            current + jump
        } else {
            if get_config().selection_wrap && current == n_items - 1 {
                0
            } else {
                n_items - 1
            }
        };
        selection.set_selected(next);
    });
}

pub fn select_page_up() {
    disable_mouse();

    with_window(|w| {
        let selection = &w.selection;
        let current = selection.selected();
        let n_items = selection.n_items();
        let jump = get_config().page_jump_items;

        if n_items == 0 {
            return;
        }

        let prev = if current >= jump {
            current - jump
        } else {
            if get_config().selection_wrap && current == 0 {
                n_items - 1
            } else {
                0
            }
        };
        selection.set_selected(prev);
    });
}

pub fn handle_grid_setting() {
    if is_emergency() {
        return;
    }

    if is_dmenu() {
        with_window(|w| {
            w.list.set_max_columns(1);
            w.list.set_min_columns(1);
            w.list.remove_css_class("grid");
            set_is_grid(false);
        });

        return;
    }

    let p = if !get_provider().is_empty() {
        get_provider()
    } else {
        get_prefix_provider()
    };

    if p.is_empty() {
        with_window(|w| {
            let cols = get_theme_grid_columns();
            w.items.remove_all();
            w.list.set_max_columns(cols);
            w.list.set_min_columns(cols);
            set_is_grid(cols > 1);
        });

        return;
    }

    if let Some(cols) = &get_config().columns
        && let Some(c) = cols.get(&p)
    {
        with_window(|w| {
            w.items.remove_all();
            w.list.set_max_columns(*c);
            w.list.set_min_columns(*c);
            set_is_grid(*c > 1);
        });
    }
}

pub fn handle_changed_items() {
    with_window(|w| {
        let s = &w.selection;

        if let Some(p) = &w.placeholder {
            let provider = if !get_provider().is_empty() {
                get_provider()
            } else {
                get_prefix_provider()
            };

            if let Some(placeholders) = &get_config().placeholders {
                if s.n_items() == 0 {
                    let ph = placeholders
                        .get(&provider)
                        .or(placeholders.get("default"))
                        .unwrap();

                    p.set_text(&ph.list);
                    p.set_visible(s.n_items() == 0);
                } else {
                    p.set_visible(false);
                }
            }
        }

        w.scroll.set_visible(s.n_items() != 0);

        while let Some(child) = w.item_keybinds.first_child() {
            w.item_keybinds.remove(&child);
        }

        if s.n_items() == 0 {
            crate::preview::clear_all_caches();
        }
    });
}
