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
pub const ACTION_QUICK_ACTIVATE: &str = "%QUICK_ACTIVATE%";

#[derive(Debug, Clone)]
pub enum AfterAction {
    KeepOpen,
    Close,
    Nothing,
    Reload,
    ClearReload,
}

#[derive(Debug, Clone)]
pub struct Keybind {
    pub bind: String,
    pub action: Action,
}

#[derive(Debug, Clone)]
pub struct Action {
    pub action: String,
    pub after: AfterAction,
    pub label: &'static str,
    pub required_states: Option<Vec<&'static str>>,
}

static BINDS: LazyLock<RwLock<HashMap<Key, HashMap<gdk::ModifierType, Action>>>> =
    LazyLock::new(RwLock::default);
static PROVIDER_BINDS: LazyLock<
    RwLock<HashMap<String, HashMap<Key, HashMap<gdk::ModifierType, Action>>>>,
> = LazyLock::new(RwLock::default);
static PROVIDER_GLOBAL_BINDS: LazyLock<
    RwLock<HashMap<String, HashMap<Key, HashMap<gdk::ModifierType, Action>>>>,
> = LazyLock::new(RwLock::default);

pub static MODIFIERS: LazyLock<HashMap<&'static str, gdk::ModifierType>> = LazyLock::new(|| {
    let mut map = HashMap::new();
    map.insert("ctrl", gdk::ModifierType::CONTROL_MASK);
    map.insert("alt", gdk::ModifierType::ALT_MASK);
    map.insert("shift", gdk::ModifierType::SHIFT_MASK);
    map.insert("super", gdk::ModifierType::SUPER_MASK);
    map
});

pub fn setup_binds() {
    PROVIDERS.get().unwrap().iter().for_each(|(k, v)| {
        v.get_keybinds().iter().for_each(|bind| {
            parse_bind(bind, k, false).unwrap();
        });

        if let Some(binds) = v.get_global_keybinds() {
            binds.iter().for_each(|bind| {
                parse_bind(bind, k, true).unwrap();
            });
        }
    });

    let config = get_config();

    parse_bind(
        &Keybind {
            bind: config.keybinds.close.clone(),
            action: Action {
                label: "close",
                required_states: None,
                action: ACTION_CLOSE.to_string(),
                after: AfterAction::Close,
            },
        },
        "",
        false,
    )
    .unwrap();

    parse_bind(
        &Keybind {
            bind: config.keybinds.next.clone(),
            action: Action {
                label: "select next",
                required_states: None,
                action: ACTION_SELECT_NEXT.to_string(),
                after: AfterAction::Nothing,
            },
        },
        "",
        false,
    )
    .unwrap();

    parse_bind(
        &Keybind {
            bind: config.keybinds.previous.clone(),
            action: Action {
                label: "select previous",
                required_states: None,
                action: ACTION_SELECT_PREVIOUS.to_string(),
                after: AfterAction::Nothing,
            },
        },
        "",
        false,
    )
    .unwrap();

    parse_bind(
        &Keybind {
            bind: config.keybinds.toggle_exact.clone(),
            action: Action {
                label: "toggle exact search",
                required_states: None,
                action: ACTION_TOGGLE_EXACT.to_string(),
                after: AfterAction::Nothing,
            },
        },
        "",
        false,
    )
    .unwrap();

    parse_bind(
        &Keybind {
            bind: config.keybinds.resume_last_query.clone(),
            action: Action {
                label: "resume last query",
                required_states: None,
                action: ACTION_RESUME_LAST_QUERY.to_string(),
                after: AfterAction::Nothing,
            },
        },
        "",
        false,
    )
    .unwrap();

    if let Some(qa) = &config.keybinds.quick_activate {
        qa.iter().enumerate().for_each(|(k, s)| {
            let action_str = format!("{}:{}", ACTION_QUICK_ACTIVATE, k);

            parse_bind(
                &Keybind {
                    bind: s.clone(),
                    action: Action {
                        label: "quick activate",
                        required_states: None,
                        action: action_str,
                        after: AfterAction::Close,
                    },
                },
                "",
                false,
            )
            .unwrap();
        });
    }
}

fn parse_bind(b: &Keybind, provider: &str, global: bool) -> Result<(), Box<dyn std::error::Error>> {
    let mut fields = b.bind.split_whitespace().peekable();
    if fields.peek().is_none() {
        return Err("incorrect bind".into());
    }

    // helper to register a single key + modifier set for the action
    let register = |key: Key, modifier: gdk::ModifierType| {
        if provider.is_empty() {
            let mut binds = BINDS.write().unwrap();
            binds
                .entry(key)
                .or_insert_with(HashMap::new)
                .insert(modifier, b.action.clone());
            return;
        }

        if !global {
            let mut provider_binds = PROVIDER_BINDS.write().unwrap();
            provider_binds
                .entry(provider.to_string())
                .or_insert_with(HashMap::new)
                .entry(key)
                .or_insert_with(HashMap::new)
                .insert(modifier, b.action.clone());
        } else {
            let mut provider_binds = PROVIDER_GLOBAL_BINDS.write().unwrap();
            provider_binds
                .entry(provider.to_string())
                .or_insert_with(HashMap::new)
                .entry(key)
                .or_insert_with(HashMap::new)
                .insert(modifier, b.action.clone());
        }
    };

    let mut modifiers_list: Vec<gdk::ModifierType> = Vec::new();
    let mut current_key: Option<Key> = None;

    // iterate tokens; tokens are either modifiers (e.g. "ctrl") or key names (e.g. "j" or "Down").
    // When we encounter a new key and a previous key already exists, register the previous key
    // with the modifiers collected so far, then continue to the next.
    for field in fields {
        if let Some(&modifier) = MODIFIERS.get(field) {
            modifiers_list.push(modifier);
            continue;
        }

        // field is not a modifier, so it's a key name
        let parsed = Key::from_name(field.to_string());
        let parsed_key = match parsed {
            Some(k) => k,
            None => {
                eprintln!(
                    "Keybind Error: unable to create key from name: '{}' in '{}'.",
                    field, b.bind
                );
                std::process::exit(1);
            }
        };

        if let Some(prev_key) = current_key.take() {
            // register previous key with modifiers we've accumulated up to this point
            let modifier = modifiers_list
                .iter()
                .fold(gdk::ModifierType::empty(), |acc, &m| acc | m);
            register(prev_key, modifier);
            // reset modifiers for the next key
            modifiers_list.clear();
        }

        current_key = Some(parsed_key);
    }

    // register last collected key + modifiers
    let key = current_key.ok_or("incorrect bind")?;
    let modifier = modifiers_list
        .iter()
        .fold(gdk::ModifierType::empty(), |acc, &m| acc | m);

    register(key, modifier);

    Ok(())
}


pub fn get_bind(key: Key, modifier: gdk::ModifierType) -> Option<Action> {
    BINDS
        .read()
        .ok()?
        .get(&key.to_lower())?
        .get(&modifier)
        .cloned()
}

pub fn get_provider_bind(provider: &str, key: Key, modifier: gdk::ModifierType) -> Option<Action> {
    PROVIDER_BINDS
        .read()
        .ok()?
        .get(provider)?
        .get(&key.to_lower())?
        .get(&modifier)
        .cloned()
}

pub fn get_provider_global_bind(
    provider: &str,
    key: Key,
    modifier: gdk::ModifierType,
) -> Option<Action> {
    PROVIDER_GLOBAL_BINDS
        .read()
        .ok()?
        .get(provider)?
        .get(&key.to_lower())?
        .get(&modifier)
        .cloned()
}
