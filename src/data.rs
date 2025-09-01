use crate::config::get_config;
use crate::protos::generated_proto::activate::ActivateRequest;
use crate::protos::generated_proto::query::{QueryRequest, QueryResponse};
use crate::protos::generated_proto::subscribe::SubscribeRequest;
use crate::protos::generated_proto::subscribe::SubscribeResponse;
use crate::state::with_state;
use crate::theme::with_installed_providers;
use crate::ui::window::{set_keybind_hint, with_window};
use crate::{QueryResponseObject, handle_preview, send_message};
use gtk4::{glib, prelude::*};
use nucleo_matcher::pattern::{CaseMatching, Normalization, Pattern};
use nucleo_matcher::{Config, Matcher};
use protobuf::Message;
use std::io::{BufReader, Read, Write};
use std::os::unix::net::UnixStream;
use std::path::{Path, PathBuf};
use std::process::Command;
use std::sync::Mutex;
use std::time::Duration;
use std::{env, thread};

static CONN: Mutex<Option<UnixStream>> = Mutex::new(None);
static MENUCONN: Mutex<Option<UnixStream>> = Mutex::new(None);

pub fn input_changed(text: &str) {
    with_state(|s| {
        if s.is_dmenu() {
            sort_items_fuzzy(&text);
        } else {
            query(text);
        }
    });
}

fn sort_items_fuzzy(query: &str) {
    with_window(|w| {
        let list_store = &w.items;

        let mut items: Vec<QueryResponseObject> = Vec::new();
        for i in 0..list_store.n_items() {
            if let Some(item) = list_store.item(i) {
                if let Ok(obj) = item.downcast::<QueryResponseObject>() {
                    items.push(obj);
                }
            }
        }

        if query.is_empty() {
            items.sort_by(|a, b| {
                let score_a = a.response().item.as_ref().map(|i| i.score).unwrap_or(0);
                let score_b = b.response().item.as_ref().map(|i| i.score).unwrap_or(0);
                score_b.cmp(&score_a)
            });
        } else {
            let texts: Vec<String> = items
                .iter()
                .map(|item| {
                    item.response()
                        .item
                        .as_ref()
                        .map(|i| i.text.clone())
                        .unwrap_or_default()
                })
                .collect();

            let mut matcher = Matcher::new(Config::DEFAULT.match_paths());
            let pattern = Pattern::parse(query, CaseMatching::Ignore, Normalization::Smart);
            let matches: Vec<(String, u32)> = pattern.match_list(texts, &mut matcher);

            let score_map: std::collections::HashMap<&str, u32> = matches
                .iter()
                .map(|(text, score)| (text.as_str(), *score))
                .collect();

            items.sort_by(|a, b| {
                let ra = a.response();
                let text_a = ra.item.as_ref().map(|i| i.text.as_str()).unwrap_or("");
                let rb = b.response();
                let text_b = rb.item.as_ref().map(|i| i.text.as_str()).unwrap_or("");

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

        list_store.remove_all();
        for item in items {
            list_store.append(&item);
        }
    });
}

pub fn init_socket() -> Result<(), Box<dyn std::error::Error>> {
    let mut socket_path = if let Ok(value) = env::var("XDG_RUNTIME_DIR") {
        PathBuf::from(value)
    } else {
        std::env::temp_dir()
    };

    socket_path.push("elephant");
    socket_path.push("elephant.sock");

    println!("waiting for elephant to start...");
    wait_for_file(&socket_path.to_string_lossy().to_string());
    println!("connecting to elephant...");

    let conn = loop {
        match UnixStream::connect(&socket_path) {
            Ok(conn) => break conn,
            Err(e) => {
                println!("Failed to connect: {}. Retrying in 1 second...", e);
                thread::sleep(Duration::from_secs(1));
            }
        }
    };
    *CONN.lock().unwrap() = Some(conn);

    let menuconn = loop {
        match UnixStream::connect(&socket_path) {
            Ok(conn) => break conn,
            Err(e) => {
                println!("Failed to connect to menu: {}. Retrying in 1 second...", e);
                thread::sleep(Duration::from_secs(1));
            }
        }
    };
    *MENUCONN.lock().unwrap() = Some(menuconn);

    subscribe_menu().unwrap();
    start_listening();

    Ok(())
}

fn start_listening() {
    thread::spawn(|| {
        if let Err(e) = listen_loop() {
            eprintln!("Listen loop error: {}", e);
        }
    });

    thread::spawn(|| {
        if let Err(e) = listen_menus_loop() {
            eprintln!("Listen menu_loop error: {}", e);
        }
    });
}

fn listen_menus_loop() -> Result<(), Box<dyn std::error::Error>> {
    let mut conn_guard = MENUCONN.lock().unwrap();
    let conn = conn_guard.as_mut().ok_or("Connection not initialized")?;

    let mut conn_clone = conn.try_clone()?;
    drop(conn_guard);

    let mut reader = BufReader::new(&mut conn_clone);

    loop {
        let mut header = [0u8; 5];
        match reader.read_exact(&mut header) {
            Ok(_) => {}
            Err(e) => return Err(e.into()),
        }

        match header[0] {
            0 => {
                let length = u32::from_be_bytes([header[1], header[2], header[3], header[4]]);

                let mut payload = vec![0u8; length as usize];
                reader.read_exact(&mut payload)?;

                let mut resp = SubscribeResponse::new();
                resp.merge_from_bytes(&payload)?;

                glib::idle_add_once(move || {
                    with_state(|s| {
                        s.set_provider(&resp.value);
                    });

                    with_window(|w| {
                        if let Some(input) = &w.input {
                            input.set_text("");
                            input.emit_by_name::<()>("changed", &[]);
                        }

                        w.window.present();
                    });

                    with_state(|s| {
                        s.is_visible.set(true);
                    });
                });
            }
            _ => {
                continue;
            }
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
        match reader.read_exact(&mut header) {
            Ok(_) => {}
            Err(e) => return Err(e.into()),
        }

        match header[0] {
            255 => {
                glib::idle_add_once(|| {
                    set_keybind_hint();
                    handle_preview();
                });
                continue;
            }
            254 => {
                glib::idle_add_once(|| {
                    clear_items();
                });
                continue;
            }
            _ => {
                let length = u32::from_be_bytes([header[1], header[2], header[3], header[4]]);

                let mut payload = vec![0u8; length as usize];
                reader.read_exact(&mut payload)?;

                let mut resp = QueryResponse::new();
                resp.merge_from_bytes(&payload)?;

                let header_type = header[0];

                glib::idle_add_once(move || {
                    handle_response(resp, header_type);
                });
            }
        }
    }
}

fn handle_response(resp: QueryResponse, header_type: u8) {
    if header_type == 1 {
        update_existing_item(resp);
    } else {
        add_new_item(resp);
    }
}

fn clear_items() {
    with_window(|w| {
        w.items.remove_all();
    });
    crate::preview::clear_all_caches();
}

fn update_existing_item(resp: QueryResponse) {
    with_window(|w| {
        let items = &w.items;
        let n_items = items.n_items();
        for i in 0..n_items {
            if let Some(obj) = items.item(i).and_downcast::<crate::QueryResponseObject>() {
                let existing = obj.response();
                if let (Some(existing_item), Some(resp_item)) =
                    (existing.item.as_ref(), resp.item.as_ref())
                {
                    if existing_item.identifier == resp_item.identifier {
                        if resp_item.text == "%DELETE%" {
                            items.remove(i);
                        } else {
                            items.splice(i, 1, &[crate::QueryResponseObject::new(resp)]);
                        }
                        break;
                    }
                }
            }
        }

        set_keybind_hint();
    });
}

fn add_new_item(resp: QueryResponse) {
    with_window(|w| {
        let items = &w.items;
        let n_items = items.n_items();

        if n_items > 0 {
            if let Some(last_obj) = items
                .item(n_items - 1)
                .and_downcast::<crate::QueryResponseObject>()
            {
                let last_resp = last_obj.response();

                if resp.qid > last_resp.qid
                    || (resp.qid == last_resp.qid && resp.iid > last_resp.iid)
                {
                    items.remove_all();
                }
            }
        }

        items.append(&crate::QueryResponseObject::new(resp));
    });
}

fn query(text: &str) {
    with_state(|s| {
        let mut query_text = text.to_string();
        let mut exact = false;
        let cfg = get_config();
        let mut provider = s.get_provider();

        with_installed_providers(|p| {
            if s.get_provider().is_empty() {
                for prefix in &cfg.providers.prefixes {
                    if text.starts_with(&prefix.prefix) && p.contains(&prefix.provider) {
                        provider = prefix.provider.clone();
                        query_text = text
                            .strip_prefix(&prefix.prefix)
                            .unwrap_or(text)
                            .to_string();
                        s.set_current_prefix(&prefix.prefix);
                        break;
                    }
                }
            }
        });

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
        } else {
            if text.is_empty() {
                req.providers = cfg.providers.empty.clone();
            } else {
                req.providers = cfg.providers.default.clone();
            }
        }

        let payload = req.write_to_bytes().unwrap();

        let mut buffer = Vec::new();
        buffer.push(0);

        let length = payload.len() as u32;
        buffer.extend_from_slice(&length.to_be_bytes());
        buffer.extend_from_slice(&payload);

        let mut conn_guard = CONN.lock().unwrap();
        if let Some(conn) = conn_guard.as_mut() {
            match conn.write_all(&buffer) {
                Err(_) => {
                    drop(conn_guard);
                    handle_disconnect();
                }
                _ => (),
            }
        }
    });
}

fn handle_disconnect() {
    Command::new("notify-send")
        .arg("Walker")
        .arg("reconnecting to elephant...")
        .spawn()
        .expect("failed to execute process");

    loop {
        println!("re-connect...");

        match init_socket() {
            Ok(_) => {
                println!("reconnected");
                break;
            }
            Err(err) => {
                println!("{}", err);
            }
        }
    }
}

pub fn activate(item: QueryResponse, query: &str, action: &str) {
    // handle dmenu
    if item.item.provider == "dmenu" {
        with_state(|s| {
            if s.is_service() {
                send_message(item.item.text.clone()).unwrap();
            } else {
                println!("{}", item.item.text.clone());
            }
        });
        return;
    }

    // handle switcher
    if item.item.provider == "providerlist" {
        with_state(|s| {
            s.set_provider(&item.item.identifier);
            s.set_current_prefix("");
        });
        return;
    }

    let cfg = get_config();

    let mut arguments = query;

    for prefix in &cfg.providers.prefixes {
        if item.item.provider == prefix.provider && arguments.starts_with(&prefix.prefix) {
            arguments = arguments
                .strip_prefix(&prefix.prefix)
                .expect("couldn't trim prefix");
            break;
        }
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

    let payload = req.write_to_bytes().unwrap();

    let mut buffer = Vec::new();

    buffer.push(1);

    let length = payload.len() as u32;
    buffer.extend_from_slice(&length.to_be_bytes());

    buffer.extend_from_slice(&payload);

    {
        let mut conn_guard = CONN.lock().unwrap();
        if let Some(conn) = conn_guard.as_mut() {
            match conn.write_all(&buffer) {
                Err(_) => {
                    drop(conn_guard);
                    handle_disconnect();
                }
                _ => (),
            }
        }
    }
}

fn subscribe_menu() -> Result<(), Box<dyn std::error::Error>> {
    let mut req = SubscribeRequest::new();
    req.provider = "menus".to_string();

    let payload = req.write_to_bytes()?;

    let mut buffer = Vec::new();

    buffer.push(2);

    let length = payload.len() as u32;
    buffer.extend_from_slice(&length.to_be_bytes());

    buffer.extend_from_slice(&payload);

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

fn wait_for_file(path: &str) {
    while !Path::new(path).exists() {
        thread::sleep(Duration::from_millis(10));
    }
}
