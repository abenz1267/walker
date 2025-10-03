use crate::{
    GLOBAL_DMENU_SENDER, QueryResponseObject,
    config::get_config,
    data::{activate, clipboard_disable_images_only, input_changed},
    keybinds::{
        ACTION_CLOSE, ACTION_QUICK_ACTIVATE, ACTION_RESUME_LAST_QUERY, ACTION_SELECT_NEXT,
        ACTION_SELECT_PREVIOUS, ACTION_TOGGLE_EXACT, Action, AfterAction, get_bind,
        get_provider_bind, get_provider_global_bind,
    },
    protos::generated_proto::query::QueryResponse,
    providers::PROVIDERS,
    renderers::create_item,
    send_message,
    state::{
        get_current_prefix, get_initial_height, get_initial_max_height, get_initial_max_width,
        get_initial_min_height, get_initial_min_width, get_initial_placeholder, get_initial_width,
        get_last_query, get_provider, get_theme, is_connected, is_dmenu, is_dmenu_exit_after,
        is_dmenu_keep_open, is_service, query, set_current_prefix, set_dmenu_current,
        set_dmenu_exit_after, set_dmenu_keep_open, set_hide_qa, set_initial_height,
        set_initial_max_height, set_initial_max_width, set_initial_min_height,
        set_initial_min_width, set_initial_placeholder, set_initial_width, set_input_only,
        set_is_dmenu, set_is_visible, set_last_query, set_no_search, set_param_close,
        set_parameter_height, set_parameter_max_height, set_parameter_max_width,
        set_parameter_min_height, set_parameter_min_width, set_parameter_width, set_placeholder,
        set_provider, set_query, set_theme,
    },
    theme::{setup_layer_shell, with_themes},
};
use gtk4::{
    Application, Builder, CustomFilter, Entry, EventControllerKey, EventControllerMotion,
    FilterListModel, GestureClick, Label, PropagationPhase, ScrolledWindow, SignalListItemFactory,
    SingleSelection, Window,
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
    process,
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
    pub elephant_hint: Label,
    pub keybinds: Option<Label>,
    pub scroll: ScrolledWindow,
    pub search_container: Option<gtk4::Box>,
    pub preview_container: Option<gtk4::Box>,
    pub content_container: gtk4::Box,
    pub box_wrapper: gtk4::Box,
}

pub fn with_window<F, R>(f: F) -> R
where
    F: FnOnce(&WindowData) -> R,
{
    WINDOWS.with(|windows| {
        let windows_map = windows.get().unwrap();
        let theme = get_theme();

        windows_map.get(&theme).map(f).unwrap_or_else(|| {
            println!("theme not found: {theme}");
            process::exit(130);
        })
    })
}

pub fn setup_window(app: &Application) {
    let mut windows: HashMap<String, WindowData> = HashMap::new();

    with_themes(|t| {
        for (key, val) in t {
            let builder = Builder::new();
            let _ = builder.add_from_string(&val.layout);

            let window: Window = builder
                .object("Window")
                .expect("Couldn't get 'Window' from UI file");
            let input: Option<Entry> = builder.object("Input");
            let scroll: ScrolledWindow = builder
                .object("Scroll")
                .expect("can't get scroll from layout");
            let list: GridView = builder.object("List").expect("can't get list from layout");
            let placeholder: Option<Label> = builder.object("Placeholder");
            let elephant_hint: Label = builder
                .object("ElephantHint")
                .expect("can't get ElephantHint");
            let keybinds: Option<Label> = builder.object("Keybinds");

            let filter = CustomFilter::new({
                move |entry| {
                    let item = entry.downcast_ref::<QueryResponseObject>().unwrap();

                    let q = query();

                    if is_dmenu() && !q.is_empty() {
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
            let box_wrapper: gtk4::Box =
                builder.object("BoxWrapper").expect("BoxWrapper not found");
            let content_container: gtk4::Box = builder
                .object("ContentContainer")
                .expect("ContentContainer not found");

            let ui = WindowData {
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
            };

            if let Some(p) = &ui.preview_container {
                p.set_visible(false);
            }

            ui.elephant_hint.set_visible(false);

            setup_window_behavior(&ui, app);

            if let Some(input) = &ui.input {
                setup_input_handling(input);
            }

            setup_keyboard_handling(&ui);
            setup_list_behavior(&ui);
            setup_mouse_handling(&ui);

            ui.window.set_application(Some(app));
            ui.window.set_css_classes(&[]);

            setup_layer_shell(&ui.window);

            windows.insert(key.to_string(), ui);
        }
    });

    WINDOWS.with(|s| s.set(windows).expect("failed initializing windows"));
}

fn setup_window_behavior(ui: &WindowData, app: &Application) {
    if let Some(p) = &ui.placeholder {
        p.set_visible(false);
    }

    ui.selection.set_autoselect(true);
    ui.selection.connect_items_changed(move |s, _, _, _| {
        with_window(|w| {
            if let Some(p) = &w.placeholder {
                p.set_visible(s.n_items() == 0);
            }

            w.scroll.set_visible(s.n_items() != 0);

            if let Some(k) = &w.keybinds {
                k.set_text("");
            }

            if s.n_items() == 0 {
                crate::preview::clear_all_caches();
            }
        });
    });

    if let Some(preview) = ui.builder.object::<Box>("PreviewBox") {
        preview.set_visible(false);
    }

    ui.selection.connect_selection_changed(move |_, _, _| {
        with_window(|w| {
            crate::handle_preview();
            w.list
                .scroll_to(w.selection.selected(), ListScrollFlags::NONE, None);

            set_keybind_hint();
        });
    });

    let app_copy = app.clone();

    ui.list.connect_activate(move |_, _| {
        with_window(|w| {
            let query = w.input.as_ref().map(Entry::text).unwrap_or_default();

            let Some(i) = get_selected_query_response() else {
                return;
            };

            let providers = PROVIDERS.get().unwrap();
            let provider = i.item.provider.clone();

            let action = providers.get(&provider).and_then(|p| {
                p.get_actions()
                    .iter()
                    .find(|v| v.default.unwrap_or(false))
                    .map(|k| k.action.clone())
            });

            activate(Some(i), &provider, &query, action.unwrap().as_str());
            quit(&app_copy, false);
        });
    });

    let config = get_config();

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

fn setup_input_handling(input: &Entry) {
    input.connect_changed(move |input| {
        disable_mouse();

        let text = input.text().to_string();

        if !text.contains(&get_config().global_argument_delimiter) {
            input_changed(&text);
        }
    });
}

fn setup_keyboard_handling(ui: &WindowData) {
    let controller = EventControllerKey::new();
    controller.set_propagation_phase(gtk4::PropagationPhase::Capture);

    let app = ui.app.clone();

    controller.connect_key_pressed(move |_, k, _, m| {
        let handled = with_window(|w| {
            if !is_connected() && !is_dmenu() {
                if let Some(action) = get_bind(k, m)
                    && action.action == ACTION_CLOSE
                {
                    quit(&app, true);
                }

                return true;
            }

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

            if let Some(action) = get_bind(k, m) {
                match action.action.as_str() {
                    ACTION_CLOSE => quit(&app, true),
                    ACTION_SELECT_NEXT => select_next(),
                    ACTION_SELECT_PREVIOUS => select_previous(),
                    ACTION_TOGGLE_EXACT => toggle_exact(),
                    ACTION_RESUME_LAST_QUERY => resume_last_query(),
                    action if action.starts_with(ACTION_QUICK_ACTIVATE) => {
                        if let Some((_, after)) = action.split_once(":") {
                            let i: u32 = after.parse().unwrap();
                            // TODO: keep open for quick activate
                            quick_activate(&app, i, false)
                        }
                    }
                    _ => (),
                }

                return true;
            }

            let mut keybind_action: Option<Action> = None;
            let mut provider = get_provider();

            if provider.is_empty()
                && let Some(prefix) = get_config().providers.prefixes.iter().find(|prefix| {
                    if let Some(input) = &w.input {
                        return input.text().starts_with(&prefix.prefix)
                            && PROVIDERS.get().unwrap().contains_key(&prefix.provider);
                    }

                    false
                })
            {
                provider = prefix.provider.clone();
            }

            let mut after: Option<AfterAction> = None;

            if !provider.is_empty()
                && let Some(action) = get_provider_global_bind(&provider, k, m)
            {
                keybind_action = Some(action.clone());
                after = Some(action.after.unwrap_or(AfterAction::Close));
            }

            let mut response: Option<QueryResponse> = None;

            if keybind_action.is_none() {
                let items = &w.selection;
                if items.n_items() == 0 {
                    return false;
                }

                let Some(r) = selection
                    .selected_item()
                    .and_downcast::<QueryResponseObject>()
                else {
                    return false;
                };

                let r = r.response();
                response = Some(r.clone());
                let Some(item) = r.item.as_ref() else {
                    return false;
                };

                let item_clone = item.clone();
                provider = item.provider.clone();

                if let Some(action) = get_provider_bind(&item.provider, k, m, &item.actions) {
                    after = if item_clone.identifier.starts_with("keepopen:") {
                        Some(AfterAction::ClearReload)
                    } else {
                        Some(action.after.as_ref().unwrap_or(&AfterAction::Close).clone())
                    };

                    keybind_action = Some(action);
                }

                let is_dmenu_next = item_clone.identifier.contains("dmenu:");

                if (is_dmenu_keep_open() && !is_dmenu_exit_after()) || is_dmenu_next {
                    after = Some(AfterAction::Nothing)
                }

                if is_dmenu_next {
                    set_is_dmenu(true);
                }
            }

            if keybind_action.is_none() {
                return false;
            }

            let query = w.input.as_ref().map(Entry::text).unwrap_or_default();

            if let Some(a) = keybind_action {
                activate(response, provider.as_str(), &query, &a.action);
            } else {
                return false;
            }

            if let Some(a) = after {
                match a {
                    AfterAction::Close => {
                        quit(&app, false);

                        return true;
                    }
                    AfterAction::KeepOpen => {
                        select_next();

                        return true;
                    }
                    AfterAction::ClearReload => {
                        if let Some(input) = &w.input {
                            if input.text().is_empty() {
                                input.emit_by_name::<()>("changed", &[]);
                            } else {
                                input.set_text(&get_current_prefix());
                                input.set_position(-1);
                            }
                        }
                    }
                    AfterAction::Reload => crate::data::input_changed(&query),
                    _ => {}
                }
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
            if let Some(theme) = t.get(&get_theme())
                && let Some(i) = response.item.as_ref()
            {
                create_item(item, i, theme);
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

    ui.list.set_single_click_activate(true);

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

pub fn quit(app: &Application, cancelled: bool) {
    if PROVIDERS.get().unwrap().contains_key("clipboard") {
        clipboard_disable_images_only();
    }

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
    set_parameter_height(None);
    set_parameter_width(None);
    set_parameter_min_height(None);
    set_parameter_min_width(None);
    set_parameter_max_height(None);
    set_parameter_max_width(None);
    set_no_search(false);
    set_placeholder(String::new());
    set_is_visible(false);
    set_dmenu_current(0);
    set_is_dmenu(false);
    set_input_only(false);
    set_param_close(false);
    set_hide_qa(false);
    set_query("");

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

            if let Some(input) = &w.input {
                input.set_text("");
                input.emit_by_name::<()>("changed", &[]);
            }

            w.content_container.set_visible(true);

            if let Some(keybinds) = &w.keybinds {
                keybinds.set_visible(true);
            }

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

fn quick_activate(app: &Application, i: u32, keep_open: bool) {
    with_window(|w| {
        if let Some(item) = &w.selection.item(i) {
            let item = item.clone().downcast::<QueryResponseObject>().unwrap();
            let item = &item;

            let resp = item.response();

            let query = w
                .input
                .as_ref()
                .map_or(String::new(), |i| i.text().to_string());

            let providers = PROVIDERS.get().unwrap();
            let action = providers.get(&resp.item.provider).and_then(|p| {
                p.get_actions()
                    .iter()
                    .find(|v| v.default.unwrap_or(false))
                    .map(|k| k.action.clone())
            });

            activate(
                Some(resp.clone()),
                resp.item.provider.as_str(),
                &query,
                &action.unwrap(),
            );

            if resp.item.provider == "providerlist" || resp.item.identifier.contains("menus:") {
                if let Some(input) = &w.input {
                    if input.text().is_empty() {
                        input.emit_by_name::<()>("changed", &[]);
                    } else {
                        input.set_text("");
                    }
                }
                return;
            }

            if !keep_open {
                quit(app, false);
            }
        }
    });
}

fn resume_last_query() {
    with_window(|w| {
        if !get_last_query().is_empty()
            && let Some(input) = &w.input
        {
            input.set_text(&get_last_query());
            input.set_position(-1);
        }
    });
}

pub fn toggle_exact() {
    with_window(|w| {
        let Some(i) = &w.input else {
            return;
        };

        let cfg = get_config();
        if i.text().starts_with(&cfg.exact_search_prefix)
            && let Some(t) = i.text().strip_prefix(&cfg.exact_search_prefix)
        {
            i.set_text(t);
            i.set_position(-1);
        } else if i.text().strip_prefix(&cfg.exact_search_prefix).is_some() {
            i.set_text(&format!("{}{}", cfg.exact_search_prefix, i.text()));
            i.set_position(-1);
        }
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

        let mut provider = item.provider.clone();

        if provider.starts_with("menus:") {
            provider = "menus".to_string();
        }

        if !crate::preview::has_previewer(&provider) {
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

        crate::preview::handle_preview(&provider, &item, &preview, &builder);
    });
}

pub fn set_keybind_hint() {
    with_window(|w| {
        let Some(k) = &w.keybinds else {
            return;
        };

        let Some(item) = get_selected_item() else {
            k.set_text("");
            return;
        };

        let providers = PROVIDERS.get().unwrap();

        if let Some(p) = providers.get(&item.provider) {
            k.set_text(&p.get_keybind_hint(&item.actions));
        } else if item.provider.starts_with("menus:")
            && let Some(p) = providers.get("menus")
        {
            k.set_text(&p.get_keybind_hint(&item.actions));
        } else if providers.get("menus").is_some() {
            k.set_text("");
        }
    });
}
