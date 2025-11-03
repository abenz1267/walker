use crate::config::get_config;
use crate::keybinds::{Action, AfterAction};
use crate::protos::generated_proto::activate::ActivateRequest;
use crate::protos::generated_proto::providerstate::{ProviderStateRequest, ProviderStateResponse};
use crate::protos::generated_proto::query::{QueryRequest, QueryResponse};
use crate::protos::generated_proto::subscribe::SubscribeRequest;
use crate::protos::generated_proto::subscribe::SubscribeResponse;
use crate::providers::PROVIDERS;
use crate::state::{
    get_async_after, get_current_prefix, get_current_selection, get_current_set, get_provider,
    is_connected, is_connecting, is_dmenu, is_index, is_service, set_async_after, set_block_scroll,
    set_current_prefix, set_global_provider_actions, set_global_provider_state, set_is_connected,
    set_is_connecting, set_is_visible, set_prefix_provider, set_provider, set_query,
};
use crate::ui::window::{
    check_error, handle_changed_items, set_input_text, set_keybind_hint, with_window,
};
use crate::{QueryResponseObject, send_message};
use gtk4::glib::Object;
use gtk4::{glib, prelude::*};
use nucleo_matcher::pattern::{CaseMatching, Normalization, Pattern};
use nucleo_matcher::{Config, Matcher};
use protobuf::{Message, MessageField};
use std::cmp::Ordering;
use std::collections::HashMap;
use std::io::{BufReader, Read, Write};
use std::os::unix::net::UnixStream;
use std::path::{Path, PathBuf};
use std::sync::Mutex;
use std::time::Duration;
use std::{env, thread};

static CONN: Mutex<Option<UnixStream>> = Mutex::new(None);
static MENUCONN: Mutex<Option<UnixStream>> = Mutex::new(None);
static BLUETOOTHCONN: Mutex<Option<UnixStream>> = Mutex::new(None);

pub fn input_changed(text: &str) {
    set_current_prefix(String::new());

    with_window(|w| {
        let is_empty = if text.is_empty() {
            w.window.remove_css_class("has-input");
            true
        } else {
            w.window.add_css_class("has-input");
            false
        };

        if is_dmenu() {
            if is_empty {
                set_query("");

                let list_store = &w.items;

                list_store
                    .iter()
                    .flatten()
                    .map(Object::downcast::<QueryResponseObject>)
                    .filter_map(Result::ok)
                    .for_each(|i| i.set_dmenu_score(0));
            } else {
                set_query(text);
            }

            sort_items_fuzzy(text);
        } else if is_connected() {
            if !get_provider().is_empty() {
                get_provider_state(get_provider());
            }

            set_query(text);
            query(text);
        }
    });
}

fn sort_items_fuzzy(query: &str) {
    with_window(|w| {
        let list_store = &w.items;

        let mut items: Vec<QueryResponseObject> = list_store
            .iter()
            .flatten()
            .map(Object::downcast::<QueryResponseObject>)
            .filter_map(Result::ok)
            .collect();

        list_store.remove_all();

        if query.is_empty() {
            items.sort_by(|a, b| {
                let score_a = a.response().item.as_ref().map(|i| i.score);
                let score_b = b.response().item.as_ref().map(|i| i.score);
                score_b.cmp(&score_a)
            });
        } else {
            let texts = items
                .iter()
                .map(QueryResponseObject::response)
                .map(|response| response.item)
                .filter_map(MessageField::into_option)
                .map(|item| item.text);

            let mut matcher = Matcher::new(Config::DEFAULT.match_paths());
            let pattern = Pattern::parse(query, CaseMatching::Ignore, Normalization::Smart);
            let matches: Vec<(String, u32)> = pattern.match_list(texts, &mut matcher);
            let score_map: HashMap<String, u32> = HashMap::from_iter(matches);

            items.sort_by(|a, b| {
                let ra = a.response();
                let text_a = ra
                    .item
                    .as_ref()
                    .map(|i| i.text.as_str())
                    .unwrap_or_default();
                let rb = b.response();
                let text_b = rb
                    .item
                    .as_ref()
                    .map(|i| i.text.as_str())
                    .unwrap_or_default();

                match (score_map.get(text_a), score_map.get(text_b)) {
                    (Some(aa), Some(bb)) => {
                        a.set_dmenu_score(*aa);
                        b.set_dmenu_score(*bb);
                        bb.cmp(aa)
                    }
                    (Some(aa), None) => {
                        a.set_dmenu_score(*aa);
                        Ordering::Less
                    }
                    (None, Some(bb)) => {
                        b.set_dmenu_score(*bb);
                        Ordering::Greater
                    }
                    (None, None) => text_a.cmp(text_b),
                }
            });
        }

        list_store.extend_from_slice(&items);
    });
}

pub fn init_socket() -> Result<(), Box<dyn std::error::Error>> {
    if is_connecting() {
        return Ok(());
    }

    set_is_connecting(true);
    println!("connecting to elephant...");

    let mut socket_path = env::var("XDG_RUNTIME_DIR")
        .map(PathBuf::from)
        .unwrap_or_else(|_| env::temp_dir());

    socket_path.push("elephant");
    socket_path.push("elephant.sock");

    println!("waiting for elephant to start...");
    wait_for_file(&socket_path.to_string_lossy());

    let conn = loop {
        match UnixStream::connect(&socket_path) {
            Ok(conn) => break conn,
            Err(e) => {
                println!("Failed to connect: {e}. Retrying in 1 second...");
                thread::sleep(Duration::from_secs(1));
            }
        }
    };
    *CONN.lock().unwrap() = Some(conn);

    let menuconn = loop {
        match UnixStream::connect(&socket_path) {
            Ok(conn) => break conn,
            Err(e) => {
                println!("Failed to connect to menu: {e}. Retrying in 1 second...");
                thread::sleep(Duration::from_secs(1));
            }
        }
    };

    *MENUCONN.lock().unwrap() = Some(menuconn);

    if PROVIDERS.get().unwrap().get("bluetooth").is_some() {
        let bluetoothconn = loop {
            match UnixStream::connect(&socket_path) {
                Ok(conn) => break conn,
                Err(e) => {
                    println!("Failed to connect to menu: {e}. Retrying in 1 second...");
                    thread::sleep(Duration::from_secs(1));
                }
            }
        };

        *BLUETOOTHCONN.lock().unwrap() = Some(bluetoothconn);
        subscribe_bluetooth().unwrap();
    }

    subscribe_menu().unwrap();
    start_listening();

    glib::idle_add_once(|| {
        with_window(|w| {
            w.elephant_hint.set_visible(false);
            w.scroll.set_visible(true);

            set_is_connected(true);

            if let Some(input) = &w.input {
                input.emit_by_name::<()>("changed", &[]);
            }
        });
    });

    set_is_connecting(false);

    println!("connected.");

    Ok(())
}

fn start_listening() {
    thread::spawn(|| {
        if let Err(e) = listen_loop() {
            eprintln!("Listen loop error: {e}");
            handle_disconnect();
        }
    });

    thread::spawn(|| {
        if let Err(e) = listen_menus_loop() {
            eprintln!("Listen menu_loop error: {e}");
            handle_disconnect();
        }
    });

    if PROVIDERS.get().unwrap().get("bluetooth").is_some() {
        thread::spawn(|| {
            if let Err(e) = listen_bluetooth_loop() {
                eprintln!("Listen bluetooth_loop error: {e}");
                handle_disconnect();
            }
        });
    }
}

fn listen_bluetooth_loop() -> Result<(), Box<dyn std::error::Error>> {
    let mut conn_guard = BLUETOOTHCONN.lock().unwrap();
    let conn = conn_guard.as_mut().ok_or("Connection not initialized")?;

    let mut conn_clone = conn.try_clone()?;
    drop(conn_guard);

    let mut reader = BufReader::new(&mut conn_clone);

    loop {
        let mut header = [0u8; 5];
        reader.read_exact(&mut header)?;

        match header[0] {
            0 => {
                let length = u32::from_be_bytes(header[1..].try_into().unwrap());

                let mut payload = vec![0u8; length as usize];
                reader.read_exact(&mut payload)?;

                let mut resp = SubscribeResponse::new();
                resp.merge_from_bytes(&payload)?;

                glib::idle_add_once(move || {
                    with_window(|w| {
                        if let Some(p) = &w.placeholder {
                            match resp.value.as_str() {
                                "bluetooth:remove" => p.set_text("Removing..."),
                                "bluetooth:connect" => p.set_text("Connecting..."),
                                "bluetooth:disconnect" => p.set_text("Disconnecting..."),
                                "bluetooth:trust" => p.set_text("Trusting..."),
                                "bluetooth:untrust" => p.set_text("Un-Trusting..."),
                                "bluetooth:pair" => p.set_text("Pairing..."),
                                "bluetooth:find" => p.set_text("Scanning..."),
                                _ => (),
                            }

                            p.set_visible(true);
                            w.scroll.set_visible(false);
                        }
                    });
                });
            }
            _ => continue,
        }
    }
}

fn listen_menus_loop() -> Result<(), Box<dyn std::error::Error>> {
    let mut conn_guard = MENUCONN.lock().unwrap();
    let conn = conn_guard.as_mut().ok_or("Connection not initialized")?;

    let mut conn_clone = conn.try_clone()?;
    drop(conn_guard);

    let mut reader = BufReader::new(&mut conn_clone);

    loop {
        let mut header = [0u8; 5];
        reader.read_exact(&mut header)?;

        match header[0] {
            0 => {
                let length = u32::from_be_bytes(header[1..].try_into().unwrap());

                let mut payload = vec![0u8; length as usize];
                reader.read_exact(&mut payload)?;

                let mut resp = SubscribeResponse::new();
                resp.merge_from_bytes(&payload)?;

                glib::idle_add_once(move || {
                    set_global_provider_actions(None);
                    set_provider(resp.value);

                    with_window(|w| {
                        set_input_text("");
                        w.window.present();
                    });

                    set_is_visible(true);
                });
            }
            _ => continue,
        }
    }
}

fn listen_loop() -> Result<(), Box<dyn std::error::Error>> {
    let mut conn_guard = CONN.lock().unwrap();
    let conn = conn_guard.as_mut().ok_or("Connection not initialized")?;

    let mut conn_clone = conn.try_clone()?;
    drop(conn_guard);

    let mut reader = BufReader::new(&mut conn_clone);

    loop {
        let mut header = [0u8; 5];
        reader.read_exact(&mut header)?;

        match header[0] {
            255 => {
                glib::idle_add_once(|| {
                    check_error();

                    handle_changed_items();

                    set_keybind_hint();
                    crate::ui::window::handle_preview();

                    match get_async_after() {
                        Some(AfterAction::AsyncReloadKeepSelection) => {
                            with_window(|w| {
                                set_block_scroll(true);
                                w.selection.set_selected(get_current_selection());
                                set_async_after(None);
                                set_block_scroll(false);
                            });
                        }
                        _ => {}
                    }
                });
            }
            254 => {
                glib::idle_add_once(clear_items);
            }
            230 => {}
            3 => {
                let length = u32::from_be_bytes(header[1..].try_into().unwrap());

                let mut payload = vec![0u8; length as usize];
                reader.read_exact(&mut payload)?;

                let mut resp = ProviderStateResponse::new();
                resp.merge_from_bytes(&payload)?;

                set_global_provider_state(resp);
            }
            2 => {
                glib::idle_add_once(move || match get_async_after() {
                    Some(AfterAction::AsyncReload) => {
                        with_window(|w| {
                            if let Some(input) = &w.input {
                                set_input_text(&input.text());
                            }
                        });

                        set_async_after(None);
                    }
                    Some(AfterAction::AsyncReloadKeepSelection) => {
                        with_window(|w| {
                            if let Some(input) = &w.input {
                                set_input_text(&input.text());
                            }
                        });
                    }
                    Some(AfterAction::AsyncClearReload) => {
                        with_window(|w| {
                            if let Some(input) = &w.input {
                                if input.text().is_empty() {
                                    input.emit_by_name::<()>("changed", &[]);
                                } else {
                                    set_input_text(&get_current_prefix());
                                }
                            }
                        });

                        set_async_after(None);
                    }
                    _ => (),
                });
            }
            _ => {
                let length = u32::from_be_bytes(header[1..].try_into().unwrap());

                let mut payload = vec![0u8; length as usize];
                reader.read_exact(&mut payload)?;

                let mut resp = QueryResponse::new();
                resp.merge_from_bytes(&payload)?;

                glib::idle_add_once(move || handle_response(resp, header[0]));
            }
        };
    }
}

fn handle_response(resp: QueryResponse, header_type: u8) {
    let function = match header_type {
        1 => update_existing_item,
        _ => add_new_item,
    };

    function(resp)
}

fn clear_items() {
    with_window(|w| w.items.remove_all());
    crate::preview::clear_all_caches();
}

fn update_existing_item(resp: QueryResponse) {
    with_window(|w| {
        let items = &w.items;
        let n_items = items.n_items();
        for i in 0..n_items {
            let Some(obj) = items.item(i).and_downcast::<crate::QueryResponseObject>() else {
                continue;
            };

            let existing = obj.response();
            let (Some(existing_item), Some(resp_item)) =
                (existing.item.as_ref(), resp.item.as_ref())
            else {
                continue;
            };

            if existing_item.identifier != resp_item.identifier {
                continue;
            }

            if resp_item.text == "%DELETE%" {
                items.remove(i);
            } else {
                items.splice(i, 1, &[crate::QueryResponseObject::new(resp)]);
            }
            break;
        }

        set_keybind_hint();
    });
}

fn add_new_item(resp: QueryResponse) {
    with_window(|w| {
        let items = &w.items;
        let n_items = items.n_items();

        if let Some(n_items) = n_items.checked_sub(1)
            && let Some(last_obj) = items
                .item(n_items)
                .and_downcast::<crate::QueryResponseObject>()
        {
            let last_resp = last_obj.response();

            if resp.qid > last_resp.qid {
                items.remove_all();
            }
        }

        items.append(&crate::QueryResponseObject::new(resp));
    });
}

fn query(text: &str) {
    set_prefix_provider(String::new());

    let mut query_text = text.to_string();
    let mut exact = false;
    let cfg = get_config();
    let mut provider = get_provider();
    let providers = PROVIDERS.get().unwrap();

    if get_provider().is_empty()
        && let Some(prefix) = cfg.providers.prefixes.iter().find(|prefix| {
            text.starts_with(&prefix.prefix) && providers.contains_key(&prefix.provider)
        })
    {
        provider = prefix.provider.clone();
        query_text = text
            .strip_prefix(&prefix.prefix)
            .unwrap_or(text)
            .to_string();
        set_current_prefix(prefix.prefix.clone());
        set_prefix_provider(provider.clone());
    }

    let delimiter = &cfg.global_argument_delimiter;

    if let Some((before, _)) = query_text.split_once(delimiter) {
        query_text = before.to_string();
    }

    if let Some(stripped) = query_text.strip_prefix(&cfg.exact_search_prefix) {
        exact = true;
        query_text = stripped.to_string();
    }

    let mut req = QueryRequest::new();
    req.query = query_text;
    req.exactsearch = exact;

    if !provider.is_empty() {
        req.providers.push(provider.clone());
    }

    if req.providers.is_empty() {
        if get_current_set().is_empty() {
            if text.is_empty() {
                req.providers = cfg.providers.empty.clone();
            } else {
                req.providers = cfg.providers.default.clone();
            }
        } else {
            let set = cfg
                .providers
                .sets
                .get(&get_current_set())
                .expect("can't find specified set");
            if text.is_empty() {
                req.providers = set.empty.clone();
            } else {
                req.providers = set.default.clone();
            }
        }
    }

    if req.providers.len() == 1 {
        if let Some(mr) = cfg
            .providers
            .max_results_provider
            .get(req.providers.first().unwrap())
        {
            req.maxresults = *mr;
        } else {
            req.maxresults = cfg.providers.max_results;
        }
    } else {
        req.maxresults = cfg.providers.max_results;
    }

    let mut buffer = vec![0, 0];
    let length = req.compute_size() as u32;
    buffer.extend_from_slice(&length.to_be_bytes());
    req.write_to_vec(&mut buffer).unwrap();

    if let Some(conn) = CONN.lock().unwrap().as_mut() {
        match conn.write_all(&buffer) {
            Err(e) => {
                eprintln!("send query socket error: {e}");
                handle_disconnect();
            }
            _ => (),
        }
    }
}

fn handle_disconnect() {
    set_is_connected(false);

    thread::spawn(|| {
        glib::idle_add_once(|| {
            with_window(|w| {
                w.elephant_hint.set_visible(true);
                w.scroll.set_visible(false);
            });
        });

        while let Err(err) = init_socket() {
            println!("{err}");
        }
    });
}

pub fn clipboard_disable_images_only() {
    let mut req = ActivateRequest::new();
    req.action = "show_combined".to_string();
    req.provider = "clipboard".to_string();

    let mut buffer = vec![1, 0];
    let length = req.compute_size() as u32;
    buffer.extend_from_slice(&length.to_be_bytes());
    req.write_to_vec(&mut buffer).unwrap();

    let mut conn_guard = CONN.lock().unwrap();

    if let Some(conn) = conn_guard.as_mut() {
        match conn.write_all(&buffer) {
            Err(e) => {
                eprintln!("send clipboard disable images only socket error: {e}");
                handle_disconnect();
            }
            _ => (),
        }
    }
}

pub fn get_provider_state(provider: String) {
    let mut req = ProviderStateRequest::new();
    req.provider = provider;

    let mut buffer = vec![4, 0];
    let length = req.compute_size() as u32;
    buffer.extend_from_slice(&length.to_be_bytes());
    req.write_to_vec(&mut buffer).unwrap();

    let mut conn_guard = CONN.lock().unwrap();

    if let Some(conn) = conn_guard.as_mut() {
        match conn.write_all(&buffer) {
            Err(e) => {
                eprintln!("send providerstate request socket error: {e}");
                handle_disconnect();
            }
            _ => (),
        }
    }
}

pub fn activate(item_option: Option<QueryResponse>, provider: &str, query: &str, action: &Action) {
    let cfg = get_config();

    let mut query = query;
    if let Some(stripped) = query.strip_prefix(&cfg.exact_search_prefix) {
        query = stripped;
    }

    let mut req = ActivateRequest::new();
    req.action = action.action.to_string();
    req.provider = provider.to_string();

    if let Some(item) = item_option {
        match provider {
            "dmenu" => {
                let mut res = item.item.text.clone();

                if is_index() {
                    res = format!("{}", 1000000 - item.item.score);
                }

                if is_service() {
                    send_message(res);
                } else {
                    println!("{}", res);
                }

                return;
            }
            "providerlist" => {
                set_provider(item.item.identifier.to_string());
                set_current_prefix(String::new());
                return;
            }
            _ => {
                req.query = query.to_string();

                let delimiter =
                    if let Some(d) = cfg.providers.argument_delimiter.get(&item.item.provider) {
                        d
                    } else {
                        &cfg.global_argument_delimiter
                    };

                match query.split_once(delimiter) {
                    Some(res) => {
                        req.query = res.0.to_string();
                        req.arguments = res.1.to_string();
                    }
                    None => {
                        req.query = query.to_string();
                    }
                }
                req.provider = item.item.provider.clone();
                req.identifier = item.item.identifier.clone();
            }
        }
    } else if provider.starts_with("menus:") {
        req.identifier = provider.to_string();
    }

    if let Some(prefix) = cfg
        .providers
        .prefixes
        .iter()
        .find(|prefix| provider == prefix.provider && query.starts_with(&prefix.prefix))
    {
        if let Some(after) = req.query.to_string().strip_prefix(&prefix.prefix) {
            req.query = after.to_string();
        }
    }

    let mut buffer = vec![1, 0];
    let length = req.compute_size() as u32;
    buffer.extend_from_slice(&length.to_be_bytes());
    req.write_to_vec(&mut buffer).unwrap();

    let mut conn_guard = CONN.lock().unwrap();

    if let Some(conn) = conn_guard.as_mut() {
        match conn.write_all(&buffer) {
            Err(e) => {
                eprintln!("send activate socket error: {e}");
                handle_disconnect();
            }
            _ => (),
        }
    }
}

fn subscribe_menu() -> Result<(), Box<dyn std::error::Error>> {
    let mut req = SubscribeRequest::new();
    req.provider = "menus".to_string();

    let mut buffer = vec![2, 0];
    let length = req.compute_size() as u32;
    buffer.extend_from_slice(&length.to_be_bytes());
    req.write_to_vec(&mut buffer).unwrap();

    {
        let mut conn_guard = MENUCONN.lock().unwrap();

        if let Some(conn) = conn_guard.as_mut() {
            conn.write_all(&buffer)?;
        } else {
            return Err("Connection not available".into());
        }
    }

    Ok(())
}

fn subscribe_bluetooth() -> Result<(), Box<dyn std::error::Error>> {
    let mut req = SubscribeRequest::new();
    req.provider = "bluetooth".to_string();

    let mut buffer = vec![2, 0];
    let length = req.compute_size() as u32;
    buffer.extend_from_slice(&length.to_be_bytes());
    req.write_to_vec(&mut buffer).unwrap();

    {
        let mut conn_guard = BLUETOOTHCONN.lock().unwrap();

        if let Some(conn) = conn_guard.as_mut() {
            conn.write_all(&buffer)?;
        } else {
            return Err("Connection not available".into());
        }
    }

    Ok(())
}

fn wait_for_file(path: &str) {
    while !Path::new(path).exists() {
        thread::sleep(Duration::from_millis(10));
    }
}
