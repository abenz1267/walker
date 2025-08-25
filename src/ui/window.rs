use crate::{
    GLOBAL_DMENU_SENDER, QueryResponseObject,
    config::get_config,
    data::{activate, input_changed},
    keybinds::{
        ACTION_CLOSE, ACTION_RESUME_LAST_QUERY, ACTION_SELECT_NEXT, ACTION_SELECT_PREVIOUS,
        ACTION_TOGGLE_EXACT, AFTER_CLEAR_RELOAD, AFTER_CLOSE, AFTER_NOTHING, AFTER_RELOAD,
        get_bind, get_modifiers, get_provider_bind,
    },
    renderers::create_item,
    send_message,
    state::{WindowData, with_state},
    theme::{setup_layer_shell, with_themes},
};
use gtk4::prelude::WidgetExt;
use gtk4::prelude::{EditableExt, EventControllerExt, ListItemExt, SelectionModelExt};
use gtk4::prelude::{EntryExt, GtkWindowExt};
use gtk4::{
    Application, Builder, Entry, EventControllerKey, EventControllerMotion, Label, ScrolledWindow,
    SignalListItemFactory, SingleSelection, Window,
};
use gtk4::{Box, ListScrollFlags};
use gtk4::{
    GridView,
    glib::object::{CastNone, ObjectExt},
};
use gtk4::{gio::ListStore, glib::object::Cast};
use gtk4::{
    gio::prelude::{ApplicationExt, ListModelExt},
    prelude::GtkApplicationExt,
};
use std::{collections::HashMap, process, sync::OnceLock};

thread_local! {
    pub static WINDOWS: OnceLock<HashMap<String, WindowData>> = OnceLock::new();
}

pub fn with_window<F, R>(f: F) -> R
where
    F: FnOnce(&WindowData) -> R,
{
    with_state(|s| {
        WINDOWS.with(|windows| {
            let windows_map = windows.get().unwrap();
            let theme = s.get_theme();

            windows_map.get(&theme).map(f).unwrap_or_else(|| {
                println!("theme not found: {}", theme);
                process::exit(130);
            })
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
            let items = ListStore::new::<QueryResponseObject>();
            let placeholder: Option<Label> = builder.object("Placeholder");
            let keybinds: Option<Label> = builder.object("Keybinds");
            let selection = SingleSelection::new(Some(items.clone()));
            let search_container: Option<Box> = builder.object("SearchContainer");

            let ui = WindowData {
                search_container,
                builder: builder.clone(),
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

            setup_window_behavior(&ui, app);

            if let Some(input) = &ui.input {
                setup_input_handling(input);
            }

            setup_keyboard_handling(&ui);
            setup_list_behavior(&ui);
            setup_mouse_handling(&ui);

            ui.window.set_application(Some(app));
            ui.window.set_css_classes(&vec![]);

            setup_layer_shell(&ui.window);

            windows.insert(key.to_string(), ui);
        }
    });

    WINDOWS.with(|s| {
        s.set(windows).expect("failed initializing windows");
    });
}

fn setup_window_behavior(ui: &WindowData, app: &Application) {
    if let Some(p) = &ui.placeholder {
        p.set_visible(false);
    }

    ui.selection.set_autoselect(true);
    ui.selection.connect_items_changed(move |s, _, _, _| {
        with_window(|w| {
            if s.n_items() == 0 {
                if let Some(p) = &w.placeholder {
                    p.set_visible(true);
                }

                w.scroll.set_visible(false);

                if let Some(k) = &w.keybinds {
                    clear_keybind_hint(k);
                }

                // Clear preview caches when no items are visible
                crate::preview::clear_all_caches();
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
            let query = if let Some(input) = &w.input {
                input.text().to_string()
            } else {
                String::new()
            };

            if let Some(i) = get_selected_query_response() {
                let action = match i.item.provider.as_str() {
                    "desktopapplications" => &get_config().providers.desktopapplications.click,
                    "calc" => &get_config().providers.calc.click,
                    "clipboard" => &get_config().providers.clipboard.click,
                    "providerlist" => &get_config().providers.providerlist.click,
                    "symbols" => &get_config().providers.symbols.click,
                    "websearch" => &get_config().providers.websearch.click,
                    "menus" => &get_config().providers.menus.click,
                    "dmenu" => &get_config().providers.dmenu.click,
                    "runner" => &get_config().providers.runner.click,
                    "files" => &get_config().providers.files.click,
                    _ => "",
                };

                activate(i, &query, action);
                quit(&app_copy, false);
            };
        });
    });
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
        if let Some(action) = get_bind(k, m) {
            match action.action.as_str() {
                ACTION_CLOSE => quit(&app, true),
                ACTION_SELECT_NEXT => select_next(),
                ACTION_SELECT_PREVIOUS => select_previous(),
                ACTION_TOGGLE_EXACT => toggle_exact(),
                ACTION_RESUME_LAST_QUERY => resume_last_query(),
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
                let query = if let Some(input) = &w.input {
                    input.text().to_string()
                } else {
                    String::new()
                };

                activate(response, &query, &action.action);

                let mut after = if item_clone.identifier.starts_with("keepopen:") {
                    AFTER_CLEAR_RELOAD
                } else {
                    action.after.as_str()
                };

                with_state(|s| {
                    if s.is_dmenu_keep_open() && !s.is_dmenu_exit_after() {
                        after = AFTER_NOTHING;
                    }
                });

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
                            quit(&app, false);
                        }
                        return true;
                    }
                    AFTER_CLEAR_RELOAD => {
                        with_window(|w| {
                            if let Some(input) = &w.input {
                                if input.text().is_empty() {
                                    input.emit_by_name::<()>("changed", &[]);
                                } else {
                                    input.set_text("");
                                }
                            }
                        });
                    }
                    AFTER_RELOAD => crate::data::input_changed(&query),
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

        with_state(|s| {
            with_themes(|t| {
                if let Some(theme) = t.get(&s.get_theme()) {
                    if let Some(i) = response.item.as_ref() {
                        create_item(&item, &i, theme);
                    }
                }
            });
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
}

pub fn quit(app: &Application, cancelled: bool) {
    GLOBAL_DMENU_SENDER.with(|sender| {
        if sender.lock().unwrap().is_some() {
            send_message("CNCLD".to_string()).unwrap();
        }
    });

    if app
        .flags()
        .contains(gtk4::gio::ApplicationFlags::IS_SERVICE)
    {
        app.active_window().unwrap().set_visible(false);

        with_window(|w| {
            if let Some(preview) = w.builder.object::<Box>("Preview") {
                while let Some(child) = preview.first_child() {
                    child.unparent();
                }
            }

            w.preview_builder.borrow_mut().take();
        });

        // Clear all preview caches
        crate::preview::clear_all_caches();

        with_state(|s| {
            s.set_provider("");
            s.set_parameter_height(0);
            s.set_parameter_width(0);
            s.set_no_search(false);
            s.set_placeholder("");
            s.is_visible.set(false);
            s.set_dmenu_current(0);
            s.set_is_dmenu(false);

            if s.is_dmenu_exit_after() {
                s.set_dmenu_exit_after(false);
                s.set_dmenu_keep_open(false);
            }

            with_window(|w| {
                if let Some(input) = &w.input {
                    s.set_last_query(&input.text());
                    if !s.get_initial_placeholder().is_empty() {
                        input.set_placeholder_text(Some(&s.get_initial_placeholder()));
                        s.set_initial_placeholder("");
                    }
                }
            });
        });

        gtk4::glib::idle_add_once(|| {
            with_window(|w| {
                if let Some(search_container) = &w.search_container {
                    search_container.set_visible(true);
                }

                if let Some(input) = &w.input {
                    input.set_text("");
                    input.emit_by_name::<()>("changed", &[]);
                }

                with_state(|s| {
                    if s.get_initial_height() != 0 {
                        w.scroll.set_min_content_height(s.get_initial_height());
                        w.scroll.set_max_content_height(s.get_initial_height());
                    }

                    if s.get_initial_width() != 0 {
                        w.scroll.set_min_content_width(s.get_initial_width());
                        w.scroll.set_max_content_width(s.get_initial_width());
                    }

                    s.set_theme(&get_config().theme);
                });
            });
        });
    } else {
        if cancelled {
            process::exit(130);
        }

        app.quit();
    }
}

pub fn select_next() {
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

pub fn select_previous() {
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

fn resume_last_query() {
    with_window(|w| {
        with_state(|s| {
            if !s.get_last_query().is_empty() {
                if let Some(input) = &w.input {
                    input.set_text(&s.get_last_query());
                    input.set_position(-1);
                }
            }
        });
    });
}

pub fn toggle_exact() {
    with_window(|w| {
        if let Some(i) = &w.input {
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
    let result = with_window(|w| {
        w.selection
            .selected_item()
            .and_then(|item| item.downcast::<QueryResponseObject>().ok())
            .and_then(|obj| obj.response().item.as_ref().cloned())
    });

    return result;
}

pub fn get_selected_query_response() -> Option<crate::protos::generated_proto::query::QueryResponse>
{
    let result = with_window(|w| {
        w.selection
            .selected_item()
            .and_then(|item| item.downcast::<QueryResponseObject>().ok())
            .and_then(|obj| Some(obj.response()))
    });

    return result;
}

pub fn handle_preview() {
    with_window(|w| {
        if let Some(preview) = w.builder.object::<Box>("Preview") {
            if let Some(item) = get_selected_item() {
                let mut provider = item.provider.clone();

                if provider.starts_with("menus:") {
                    provider = "menus".to_string();
                }

                if crate::preview::has_previewer(&provider) {
                    let builder = {
                        let mut preview_builder = w.preview_builder.borrow_mut();
                        if preview_builder.is_none() {
                            let builder = Builder::new();
                            let _ = builder.add_from_string(include_str!(
                                "../../resources/themes/default/preview.xml"
                            ));
                            *preview_builder = Some(builder);
                        }
                        preview_builder.as_ref().unwrap().clone()
                    };

                    crate::preview::handle_preview(&provider, &item, &preview, &builder);
                } else {
                    preview.set_visible(false);
                }
            } else {
                preview.set_visible(false);
            }
        }
    });
}

pub fn set_keybind_hint() {
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
