use crate::config::get_config;
use crate::providers::PROVIDERS;
use gtk4::gdk::{self, Key};
use std::collections::HashMap;
use std::sync::{Arc, Mutex, OnceLock};

pub const ACTION_CLOSE: &str = "%CLOSE%";
pub const ACTION_SELECT_NEXT: &str = "%NEXT%";
pub const ACTION_SELECT_PREVIOUS: &str = "%PREVIOUS%";
pub const ACTION_TOGGLE_EXACT: &str = "%TOGGLE_EXACT%";
pub const ACTION_RESUME_LAST_QUERY: &str = "%RESUME_LAST_QUERY%";

#[derive(Debug, Clone)]
pub enum AfterAction {
    Close,
    Nothing,
    Reload,
    ClearReload,
    ClearReloadKeepPrefix,
}

#[derive(Debug, Clone)]
pub struct Keybind {
    pub bind: String,
    pub action: String,
    pub after: AfterAction,
}

#[derive(Debug, Clone)]
pub struct Action {
    pub action: String,
    pub after: AfterAction,
}

static BINDS: OnceLock<Arc<Mutex<HashMap<Key, HashMap<gdk::ModifierType, Action>>>>> =
    OnceLock::new();
static PROVIDER_BINDS: OnceLock<
    Arc<Mutex<HashMap<String, HashMap<Key, HashMap<gdk::ModifierType, Action>>>>>,
> = OnceLock::new();

fn get_binds() -> &'static Arc<Mutex<HashMap<Key, HashMap<gdk::ModifierType, Action>>>> {
    BINDS.get_or_init(|| Arc::new(Mutex::new(HashMap::new())))
}

fn get_provider_binds()
-> &'static Arc<Mutex<HashMap<String, HashMap<Key, HashMap<gdk::ModifierType, Action>>>>> {
    PROVIDER_BINDS.get_or_init(|| Arc::new(Mutex::new(HashMap::new())))
}

pub fn get_modifiers() -> HashMap<&'static str, gdk::ModifierType> {
    let mut map = HashMap::new();
    map.insert("ctrl", gdk::ModifierType::CONTROL_MASK);
    map.insert("lctrl", gdk::ModifierType::CONTROL_MASK);
    map.insert("rctrl", gdk::ModifierType::CONTROL_MASK);
    map.insert("alt", gdk::ModifierType::ALT_MASK);
    map.insert("lalt", gdk::ModifierType::ALT_MASK);
    map.insert("ralt", gdk::ModifierType::ALT_MASK);
    map.insert("lshift", gdk::ModifierType::SHIFT_MASK);
    map.insert("rshift", gdk::ModifierType::SHIFT_MASK);
    map.insert("shift", gdk::ModifierType::SHIFT_MASK);
    map
}

fn get_special_keys() -> HashMap<&'static str, Key> {
    let mut map = HashMap::new();
    map.insert("backspace", gdk::Key::BackSpace);
    map.insert("tab", gdk::Key::Tab);
    map.insert("esc", gdk::Key::Escape);
    map.insert("escape", gdk::Key::Escape);
    map.insert("kpenter", gdk::Key::KP_Enter);
    map.insert("enter", gdk::Key::Return);
    map.insert("down", gdk::Key::Down);
    map.insert("up", gdk::Key::Up);
    map.insert("left", gdk::Key::Left);
    map.insert("right", gdk::Key::Right);
    map
}

pub fn setup_binds() {
    PROVIDERS.get().unwrap().iter().for_each(|(k, v)| {
        v.get_keybinds().iter().for_each(|bind| {
            parse_bind(bind, k).unwrap();
        });
    });

    let config = get_config();

    parse_bind(
        &Keybind {
            bind: config.keybinds.close.clone(),
            action: ACTION_CLOSE.to_string(),
            after: AfterAction::Close,
        },
        "",
    )
    .unwrap();

    parse_bind(
        &Keybind {
            bind: config.keybinds.next.clone(),
            action: ACTION_SELECT_NEXT.to_string(),
            after: AfterAction::Nothing,
        },
        "",
    )
    .unwrap();

    parse_bind(
        &Keybind {
            bind: config.keybinds.previous.clone(),
            action: ACTION_SELECT_PREVIOUS.to_string(),
            after: AfterAction::Nothing,
        },
        "",
    )
    .unwrap();

    parse_bind(
        &Keybind {
            bind: config.keybinds.toggle_exact.clone(),
            action: ACTION_TOGGLE_EXACT.to_string(),
            after: AfterAction::Nothing,
        },
        "",
    )
    .unwrap();

    parse_bind(
        &Keybind {
            bind: config.keybinds.resume_last_query.clone(),
            action: ACTION_RESUME_LAST_QUERY.to_string(),
            after: AfterAction::Nothing,
        },
        "",
    )
    .unwrap();
}

fn validate_bind(bind: &str) -> bool {
    let fields: Vec<&str> = bind.split_whitespace().collect();
    let modifiers = get_modifiers();
    let special_keys = get_special_keys();

    let mut ok = true;

    for field in fields {
        if field.len() > 1 {
            let exists_mod = modifiers.contains_key(field);
            let exists_special = special_keys.contains_key(field);

            if !exists_mod && !exists_special {
                eprintln!("Invalid keybind: {} - key: {}", bind, field);
                ok = false;
            }
        }
    }

    ok
}

fn parse_bind(b: &Keybind, provider: &str) -> Result<(), Box<dyn std::error::Error>> {
    if !validate_bind(&b.bind) {
        return Err("incorrect bind".into());
    }

    let fields: Vec<&str> = b.bind.split_whitespace().collect();

    if fields.len() == 0 {
        return Err("incorrect bind".into());
    }

    let modifiers_map = get_modifiers();
    let special_keys = get_special_keys();

    let mut modifiers_list = Vec::new();
    let mut key: Option<Key> = None;

    for field in fields {
        if field.len() > 1 {
            if let Some(&modifier) = modifiers_map.get(field) {
                modifiers_list.push(modifier);
            }

            if let Some(&special_key) = special_keys.get(field) {
                key = Some(special_key);
            }
        } else {
            key = Some(Key::from_name(field.chars().next().unwrap().to_string()).unwrap());
        }
    }

    let modifier = modifiers_list
        .iter()
        .fold(gdk::ModifierType::empty(), |acc, &m| acc | m);

    let action_struct = Action {
        action: b.action.to_string(),
        after: b.after.clone(),
    };

    if key.is_some() {
        if provider.is_empty() {
            let mut binds = get_binds().lock().unwrap();
            binds
                .entry(key.unwrap())
                .or_insert_with(HashMap::new)
                .insert(modifier, action_struct);
        } else {
            let mut provider_binds = get_provider_binds().lock().unwrap();
            provider_binds
                .entry(provider.to_string())
                .or_insert_with(HashMap::new)
                .entry(key.unwrap())
                .or_insert_with(HashMap::new)
                .insert(modifier, action_struct);
        }
    } else {
        return Err("incorrect bind".into());
    }

    Ok(())
}

pub fn get_bind(key: Key, modifier: gdk::ModifierType) -> Option<Action> {
    get_binds()
        .lock()
        .unwrap()
        .get(&key)?
        .get(&modifier)
        .cloned()
}

pub fn get_provider_bind(provider: &str, key: Key, modifier: gdk::ModifierType) -> Option<Action> {
    let cfg = get_config();
    let modifiers = get_modifiers();
    let mut modifier = modifier;

    if let Some(keep_open) = modifiers.get(cfg.keep_open_modifier.as_str()) {
        if *keep_open == modifier {
            modifier = gdk::ModifierType::empty();
        }
    }

    get_provider_binds()
        .lock()
        .unwrap()
        .get(provider)?
        .get(&key)?
        .get(&modifier)
        .cloned()
}
