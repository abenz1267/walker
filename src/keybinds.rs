use crate::config::get_config;
use crate::providers::PROVIDERS;
use gtk4::gdk::{self, Key};
use std::collections::HashMap;
use std::sync::{LazyLock, RwLock};

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

static BINDS: LazyLock<RwLock<HashMap<Key, HashMap<gdk::ModifierType, Action>>>> =
    LazyLock::new(RwLock::default);
static PROVIDER_BINDS: LazyLock<
    RwLock<HashMap<String, HashMap<Key, HashMap<gdk::ModifierType, Action>>>>,
> = LazyLock::new(RwLock::default);

pub static MODIFIERS: LazyLock<HashMap<&'static str, gdk::ModifierType>> = LazyLock::new(|| {
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
});

pub static SPECIAL_KEYS: LazyLock<HashMap<&'static str, gdk::Key>> = LazyLock::new(|| {
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
});

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
    let fields = bind.split_whitespace();

    let Some(field) = fields.filter(|field| field.len() > 1).find_map(|field| {
        let exists_mod = MODIFIERS.contains_key(field);
        let exists_special = SPECIAL_KEYS.contains_key(field);

        (!exists_mod && !exists_special).then_some(field)
    }) else {
        return true;
    };

    eprintln!("Invalid keybind: {bind} - key: {field}");
    false
}

fn parse_bind(b: &Keybind, provider: &str) -> Result<(), Box<dyn std::error::Error>> {
    if !validate_bind(&b.bind) {
        return Err("incorrect bind".into());
    }

    let mut fields = b.bind.split_whitespace().peekable();

    if fields.peek().is_none() {
        return Err("incorrect bind".into());
    }

    let mut modifiers_list = Vec::new();
    let mut key: Option<Key> = None;

    for field in fields {
        if field.len() > 1 {
            if let Some(&modifier) = MODIFIERS.get(field) {
                modifiers_list.push(modifier);
            }

            if let Some(&special_key) = SPECIAL_KEYS.get(field) {
                key = Some(special_key);
            }
            continue;
        }

        key = Some(
            Key::from_name(
                field
                    .chars()
                    .next()
                    .ok_or("unable to get the next char")?
                    .to_string(),
            )
            .ok_or("unable to create key from name")?,
        );
    }

    let modifier = modifiers_list
        .iter()
        .fold(gdk::ModifierType::empty(), |acc, &m| acc | m);

    let action_struct = Action {
        action: b.action.to_string(),
        after: b.after.clone(),
    };

    let key = key.ok_or("incorrect bind")?;
    if provider.is_empty() {
        let mut binds = BINDS.write().unwrap();
        binds
            .entry(key)
            .or_insert_with(HashMap::new)
            .insert(modifier, action_struct);
        return Ok(());
    }

    let mut provider_binds = PROVIDER_BINDS.write().unwrap();
    provider_binds
        .entry(provider.to_string())
        .or_insert_with(HashMap::new)
        .entry(key)
        .or_insert_with(HashMap::new)
        .insert(modifier, action_struct);

    Ok(())
}

pub fn get_bind(key: Key, modifier: gdk::ModifierType) -> Option<Action> {
    BINDS.read().ok()?.get(&key)?.get(&modifier).cloned()
}

pub fn get_provider_bind(provider: &str, key: Key, modifier: gdk::ModifierType) -> Option<Action> {
    let cfg = get_config();
    let mut modifier = modifier;

    if let Some(keep_open) = MODIFIERS.get(cfg.keep_open_modifier.as_str()) {
        if *keep_open == modifier {
            modifier = gdk::ModifierType::empty();
        }
    }

    PROVIDER_BINDS
        .read()
        .ok()?
        .get(provider)?
        .get(&key)?
        .get(&modifier)
        .cloned()
}
