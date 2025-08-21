use crate::config::get_config;
use crate::handle_preview;
use crate::protos::generated_proto::activate::ActivateRequest;
use crate::protos::generated_proto::query::{QueryRequest, QueryResponse};
use crate::protos::generated_proto::subscribe::SubscribeRequest;
use crate::protos::generated_proto::subscribe::SubscribeResponse;
use crate::state::with_state;
use crate::ui::window::{set_keybind_hint, with_window};
use gtk4::{glib, prelude::*};
use protobuf::Message;
use std::io::{BufReader, Read, Write};
use std::os::unix::net::UnixStream;
use std::sync::Mutex;
use std::thread;

static CONN: Mutex<Option<UnixStream>> = Mutex::new(None);
static MENUCONN: Mutex<Option<UnixStream>> = Mutex::new(None);

pub fn input_changed(text: String) {
    query(&text);
}

pub fn init_socket() -> Result<(), Box<dyn std::error::Error>> {
    let socket_path = std::env::temp_dir().join("elephant.sock");

    let conn = UnixStream::connect(&socket_path)?;
    *CONN.lock().unwrap() = Some(conn);

    let menuconn = UnixStream::connect(&socket_path)?;
    *MENUCONN.lock().unwrap() = Some(menuconn);

    subscribe_menu().unwrap();

    Ok(())
}

pub fn start_listening() {
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
            Err(e) if e.kind() == std::io::ErrorKind::UnexpectedEof => continue,
            Err(e) => return Err(e.into()),
        }

        match header[0] {
            0 => {
                let length = u32::from_be_bytes([header[1], header[2], header[3], header[4]]);

                let mut payload = vec![0u8; length as usize];
                reader.read_exact(&mut payload)?;

                let mut resp = SubscribeResponse::new();
                resp.merge_from_bytes(&payload)?;

                with_state(|s| {
                    s.set_provider(&resp.value);
                });

                glib::idle_add_once(|| {
                    with_window(|w| {
                        w.input.set_text("");
                        w.input.emit_by_name::<()>("changed", &[]);
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
            Err(e) if e.kind() == std::io::ErrorKind::UnexpectedEof => continue,
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
    // Clear preview caches when clearing items (new query starting)
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

        if s.get_provider().is_empty() {
            for prefix in &cfg.providers.prefixes {
                if text.starts_with(&prefix.prefix) {
                    provider = prefix.provider.clone();
                    query_text = text
                        .strip_prefix(&prefix.prefix)
                        .unwrap_or(text)
                        .to_string();
                    break;
                }
            }
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
            conn.write_all(&buffer).unwrap();
        }
    });
}

pub fn activate(item: QueryResponse, query: &str, action: &str) {
    // handle switcher
    if item.item.provider == "providerlist" {
        with_state(|s| {
            s.set_provider(&item.item.identifier);
        });
        return;
    }

    let mut req = ActivateRequest::new();
    req.qid = item.qid;
    req.provider = item.item.provider.clone();
    req.identifier = item.item.identifier.clone();
    req.action = action.to_string();
    req.arguments = query.to_string();

    let payload = req.write_to_bytes().unwrap();

    let mut buffer = Vec::new();

    buffer.push(1);

    let length = payload.len() as u32;
    buffer.extend_from_slice(&length.to_be_bytes());

    buffer.extend_from_slice(&payload);

    {
        let mut conn_guard = CONN.lock().unwrap();
        if let Some(conn) = conn_guard.as_mut() {
            conn.write_all(&buffer).unwrap();
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
