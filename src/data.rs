use crate::config::get_config;
use crate::protos::generated_proto::activate::ActivateRequest;
use crate::protos::generated_proto::query::{QueryRequest, QueryResponse};
use crate::protos::generated_proto::subscribe::SubscribeRequest;
use crate::protos::generated_proto::subscribe::SubscribeResponse;
use crate::providers::PROVIDERS;
use crate::state::{
    get_provider, is_connected, is_dmenu, set_current_prefix, set_is_connected, set_is_visible,
    set_provider,
};
use crate::ui::window::{set_keybind_hint, with_window};
use crate::{QueryResponseObject, handle_preview, send_message};
use gtk4::glib::Object;
use gtk4::{glib, prelude::*};
use nucleo_matcher::pattern::{CaseMatching, Normalization, Pattern};
use nucleo_matcher::{Config, Matcher};
use protobuf::{Message, MessageField};
use std::io::{BufReader, Read, Write};
use std::os::unix::net::UnixStream;
use std::path::{Path, PathBuf};
use std::sync::Mutex;
use std::time::Duration;
use std::{env, thread};

static CONN: Mutex<Option<UnixStream>> = Mutex::new(None);
static MENUCONN: Mutex<Option<UnixStream>> = Mutex::new(None);

pub fn input_changed(text: &str) {
    if is_dmenu() {
        sort_items_fuzzy(&text);
        return;
    }

    if is_connected() {
        query(text);
    }
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

            let score_map: std::collections::HashMap<&str, u32> = matches
                .iter()
                .map(|(text, score)| (text.as_str(), *score))
                .collect();

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

                let score_a = score_map.get(text_a);
                let score_b = score_map.get(text_b);

                match (score_a, score_b) {
                    (Some(a), Some(b)) => b.cmp(a),              // Higher scores first
                    (Some(_), None) => std::cmp::Ordering::Less, // Matched items first
                    (None, Some(_)) => std::cmp::Ordering::Greater,
                    (None, None) => text_a.cmp(text_b), // Alphabetical for non-matches
                }
            });
        }

        list_store.extend_from_slice(&items);
    });
}

pub fn init_socket() -> Result<(), Box<dyn std::error::Error>> {
    let mut socket_path = env::var("XDG_RUNTIME_DIR")
        .map(PathBuf::from)
        .unwrap_or_else(|_| env::temp_dir());

    socket_path.push("elephant");
    socket_path.push("elephant.sock");

    println!("waiting for elephant to start...");
    wait_for_file(&socket_path.to_string_lossy().to_string());
    println!("connecting to elephant...");

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

    Ok(())
}

fn start_listening() {
    thread::spawn(|| {
        if let Err(e) = listen_loop() {
            eprintln!("Listen loop error: {e}");
        }
    });

    thread::spawn(|| {
        if let Err(e) = listen_menus_loop() {
            eprintln!("Listen menu_loop error: {e}");
        }
    });
}

fn listen_menus_loop() -> Result<(), Box<dyn std::error::Error>> {
    let mut conn_guard = MENUCONN.lock().unwrap();
    let mut conn = conn_guard
        .as_mut()
        .ok_or("Connection not initialized")?
        .try_clone()?;

    let mut reader = BufReader::new(&mut conn);

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
                    set_provider(resp.value);

                    with_window(|w| {
                        if let Some(input) = &w.input {
                            input.set_text("");
                            input.emit_by_name::<()>("changed", &[]);
                        }

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
    let mut conn = conn_guard
        .as_mut()
        .ok_or("Connection not initialized")?
        .try_clone()?;

    let mut reader = BufReader::new(&mut conn);

    loop {
        let mut header = [0u8; 5];
        reader.read_exact(&mut header)?;

        match header[0] {
            255 => glib::idle_add_once(|| {
                set_keybind_hint();
                handle_preview();
            }),
            254 => glib::idle_add_once(clear_items),
            _ => {
                let length = u32::from_be_bytes(header[1..].try_into().unwrap());

                let mut payload = vec![0u8; length as usize];
                reader.read_exact(&mut payload)?;

                let mut resp = QueryResponse::new();
                resp.merge_from_bytes(&payload)?;

                glib::idle_add_once(move || handle_response(resp, header[0]))
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

            if resp.qid > last_resp.qid || (resp.qid == last_resp.qid && resp.iid > last_resp.iid) {
                items.remove_all();
            }
        }

        items.append(&crate::QueryResponseObject::new(resp));
    });
}

fn query(text: &str) {
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
    // TODO: per provider config
    req.maxresults = 50;
    req.exactsearch = exact;

    if !provider.is_empty() {
        req.providers.push(provider.clone());
    } else if text.is_empty() {
        req.providers = cfg.providers.empty.clone();
    } else {
        req.providers = cfg.providers.default.clone();
    }

    let mut buffer = vec![0];
    let length = req.compute_size() as u32;
    buffer.extend_from_slice(&length.to_be_bytes());
    req.write_to_vec(&mut buffer).unwrap();

    let mut conn_guard = CONN.lock().unwrap();
    if let Some(conn) = conn_guard.as_mut()
        && conn.write_all(&buffer).is_err()
    {
        drop(conn_guard);
        handle_disconnect();
    }
}

fn handle_disconnect() {
    thread::spawn(|| {
        glib::idle_add_once(|| {
            with_window(|w| {
                w.elephant_hint.set_visible(true);
                w.scroll.set_visible(false);
            });
        });

        println!("re-connecting...");
        while let Err(err) = init_socket() {
            println!("{err}");
        }
        println!("reconnected");
    });
}

pub fn activate(item: QueryResponse, query: &str, action: &str) {
    match item.item.provider.as_str() {
        "dmenu" => {
            send_message(item.item.text.clone()).unwrap();
            return;
        }
        "providerlist" => {
            set_provider(item.item.identifier.to_string());
            set_current_prefix(String::new());
            return;
        }
        _ => (),
    }

    let cfg = get_config();
    let mut arguments = query;

    if let Some(prefix) = cfg.providers.prefixes.iter().find(|prefix| {
        item.item.provider == prefix.provider && arguments.starts_with(&prefix.prefix)
    }) {
        arguments = arguments
            .strip_prefix(&prefix.prefix)
            .expect("couldn't trim prefix");
    }

    if let Some(stripped) = arguments.strip_prefix(&cfg.exact_search_prefix) {
        arguments = stripped;
    }

    let mut req = ActivateRequest::new();
    req.qid = item.qid;
    req.provider = item.item.provider.clone();
    req.identifier = item.item.identifier.clone();
    req.action = action.to_string();
    req.arguments = arguments.to_string();

    let mut buffer = vec![1];
    let length = req.compute_size() as u32;
    buffer.extend_from_slice(&length.to_be_bytes());
    req.write_to_vec(&mut buffer).unwrap();

    let mut conn_guard = CONN.lock().unwrap();
    if let Some(conn) = conn_guard.as_mut()
        && conn.write_all(&buffer).is_err()
    {
        drop(conn_guard);
        handle_disconnect();
    }
}

fn subscribe_menu() -> Result<(), Box<dyn std::error::Error>> {
    let mut req = SubscribeRequest::new();
    req.provider = "menus".to_string();

    let mut buffer = vec![2];
    let length = req.compute_size() as u32;
    buffer.extend_from_slice(&length.to_be_bytes());
    req.write_to_vec(&mut buffer).unwrap();

    let mut conn_guard = MENUCONN.lock().unwrap();
    let conn = conn_guard.as_mut().ok_or("Connection not available")?;

    conn.write_all(&buffer)?;

    Ok(())
}

fn wait_for_file(path: &str) {
    while !Path::new(path).exists() {
        thread::sleep(Duration::from_millis(10));
    }
}
