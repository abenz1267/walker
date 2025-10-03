use crate::config::get_config;
use crate::providers::PROVIDERS;
use gtk4::gdk::{self, Key};
use serde::{Deserialize, Serialize};
use std::collections::HashMap;
use std::sync::{LazyLock, RwLock};

pub const ACTION_CLOSE: &str = "%CLOSE%";
pub const ACTION_SELECT_NEXT: &str = "%NEXT%";
pub const ACTION_SELECT_PREVIOUS: &str = "%PREVIOUS%";
pub const ACTION_TOGGLE_EXACT: &str = "%TOGGLE_EXACT%";
pub const ACTION_RESUME_LAST_QUERY: &str = "%RESUME_LAST_QUERY%";
pub const ACTION_QUICK_ACTIVATE: &str = "%QUICK_ACTIVATE%";

#[derive(Debug, Clone, Serialize, Deserialize)]
pub enum AfterAction {
    KeepOpen,
    Close,
    Nothing,
    Reload,
    ClearReload,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Action {
    pub action: String,
    pub global: Option<bool>,
    pub default: Option<bool>,
    pub bind: String,
    pub after: Option<AfterAction>,
    pub label: Option<String>,
    pub required_states: Option<Vec<String>>,
}

static BINDS: LazyLock<RwLock<HashMap<Key, HashMap<gdk::ModifierType, Action>>>> =
    LazyLock::new(RwLock::default);
static PROVIDER_BINDS: LazyLock<
    RwLock<HashMap<String, HashMap<Key, HashMap<gdk::ModifierType, Vec<Action>>>>>,
> = LazyLock::new(RwLock::default);
static PROVIDER_GLOBAL_BINDS: LazyLock<
    RwLock<HashMap<String, HashMap<Key, HashMap<gdk::ModifierType, Vec<Action>>>>>,
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
        v.get_actions().iter().for_each(|v| {
            parse_bind(v, k).unwrap();
        });
    });

    let config = get_config();

    config.keybinds.close.iter().for_each(|b| {
        parse_bind(
            &Action {
                action: ACTION_CLOSE.to_string(),
                global: None,
                default: Some(true),
                bind: b.clone(),
                label: Some("close".to_string()),
                required_states: None,
                after: None,
            },
            "",
        )
        .unwrap();
    });

    config.keybinds.next.iter().for_each(|b| {
        parse_bind(
            &Action {
                action: ACTION_SELECT_NEXT.to_string(),
                default: None,
                global: Some(true),
                bind: b.clone(),
                label: Some("select next".to_string()),
                required_states: None,
                after: Some(AfterAction::Nothing),
            },
            "",
        )
        .unwrap();
    });

    config.keybinds.previous.iter().for_each(|b| {
        parse_bind(
            &Action {
                action: ACTION_SELECT_PREVIOUS.to_string(),
                default: None,
                global: Some(true),
                bind: b.clone(),
                label: Some("select previous".to_string()),
                required_states: None,
                after: Some(AfterAction::Nothing),
            },
            "",
        )
        .unwrap();
    });

    config.keybinds.toggle_exact.iter().for_each(|b| {
        parse_bind(
            &Action {
                action: ACTION_TOGGLE_EXACT.to_string(),
                default: None,
                global: Some(true),
                bind: b.clone(),
                label: Some("toggle exact search".to_string()),
                required_states: None,
                after: Some(AfterAction::Nothing),
            },
            "",
        )
        .unwrap();
    });

    config.keybinds.resume_last_query.iter().for_each(|b| {
        parse_bind(
            &Action {
                action: ACTION_RESUME_LAST_QUERY.to_string(),
                bind: b.clone(),
                default: None,
                global: Some(true),
                label: Some("resume last query".to_string()),
                required_states: None,
                after: Some(AfterAction::Nothing),
            },
            "",
        )
        .unwrap();
    });

    if let Some(qa) = &config.keybinds.quick_activate {
        qa.iter().enumerate().for_each(|(k, s)| {
            let action_str = format!("{}:{}", ACTION_QUICK_ACTIVATE, k);

            parse_bind(
                &Action {
                    default: None,
                    action: action_str,
                    global: Some(true),
                    bind: s.clone(),
                    label: Some("quick activate".to_string()),
                    required_states: None,
                    after: None,
                },
                "",
            )
            .unwrap();
        });
    }
}

fn parse_bind(b: &Action, provider: &str) -> Result<(), Box<dyn std::error::Error>> {
    let mut fields = b.bind.split_whitespace().peekable();

    if fields.peek().is_none() {
        return Err("incorrect bind".into());
    }

    let mut modifiers_list = Vec::new();
    let mut key: Option<Key> = None;

    for field in fields {
        if let Some(&modifier) = MODIFIERS.get(field) {
            modifiers_list.push(modifier);
            continue;
        }

        key = match Key::from_name(field.to_string()) {
            Some(k) => Some(k),
            None => {
                eprintln!(
                    "Keybind Error: unable to create key from name: '{}' in '{}'.",
                    field, b.bind
                );
                std::process::exit(1);
            }
        };
    }

    let modifier = modifiers_list
        .iter()
        .fold(gdk::ModifierType::empty(), |acc, &m| acc | m);

    let key = key.ok_or("incorrect bind")?;
    if provider.is_empty() {
        let mut binds = BINDS.write().unwrap();
        binds
            .entry(key)
            .or_insert_with(HashMap::new)
            .insert(modifier, b.clone());
        return Ok(());
    }

    if !b.global.unwrap_or(false) {
        let mut provider_binds = PROVIDER_BINDS.write().unwrap();
        provider_binds
            .entry(provider.to_string())
            .or_insert_with(HashMap::new)
            .entry(key)
            .or_insert_with(HashMap::new)
            .entry(modifier)
            .or_insert_with(Vec::new)
            .push(b.clone());
    } else {
        let mut provider_binds = PROVIDER_GLOBAL_BINDS.write().unwrap();
        provider_binds
            .entry(provider.to_string())
            .or_insert_with(HashMap::new)
            .entry(key)
            .or_insert_with(HashMap::new)
            .entry(modifier)
            .or_insert_with(Vec::new)
            .push(b.clone());
    }

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

pub fn get_provider_bind(
    provider: &str,
    key: Key,
    modifier: gdk::ModifierType,
    state: &Vec<String>,
) -> Option<Action> {
    PROVIDER_BINDS
        .read()
        .ok()?
        .get(provider)?
        .get(&key.to_lower())?
        .get(&modifier)?
        .iter()
        .find(|action| match &action.required_states {
            Some(required) => required
                .iter()
                .any(|required_state| state.contains(&required_state.to_string())),
            None => true,
        })
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
        .get(&modifier)?
        .first()
        .cloned()
}
