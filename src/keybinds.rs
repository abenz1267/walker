use crate::config::get_config;
use crate::providers::PROVIDERS;
use gtk4::gdk::{self, Key};
use gtk4::glib::bitflags::Flags;
use serde::{Deserialize, Serialize};
use std::collections::HashMap;
use std::sync::{LazyLock, RwLock};

pub const ACTION_CLOSE: &str = "%CLOSE%";
pub const ACTION_SELECT_NEXT: &str = "%NEXT%";
pub const ACTION_SELECT_PREVIOUS: &str = "%PREVIOUS%";
pub const ACTION_TOGGLE_EXACT: &str = "%TOGGLE_EXACT%";
pub const ACTION_RESUME_LAST_QUERY: &str = "%RESUME_LAST_QUERY%";
pub const ACTION_QUICK_ACTIVATE: &str = "%QUICK_ACTIVATE%";

#[derive(Debug, Clone, Serialize, Deserialize, Default)]
pub enum AfterAction {
    KeepOpen,
    #[default]
    Close,
    Nothing,
    Reload,
    ClearReload,
    AsyncClearReload,
    AsyncReload,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Action {
    pub action: String,
    pub global: Option<bool>,
    pub default: Option<bool>,

    #[serde(default = "default_bind")]
    pub bind: Option<String>,

    #[serde(default = "default_after")]
    pub after: Option<AfterAction>,

    pub label: Option<String>,
}

fn default_bind() -> Option<String> {
    Some("Return".to_string())
}

fn default_after() -> Option<AfterAction> {
    Some(AfterAction::Close)
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
                bind: Some(b.clone()),
                label: Some("close".to_string()),
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
                bind: Some(b.clone()),
                label: Some("select next".to_string()),
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
                bind: Some(b.clone()),
                label: Some("select previous".to_string()),
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
                bind: Some(b.clone()),
                label: Some("toggle exact search".to_string()),
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
                bind: Some(b.clone()),
                default: None,
                global: Some(true),
                label: Some("resume last query".to_string()),
                after: Some(AfterAction::Nothing),
            },
            "",
        )
        .unwrap();
    });

    if let Some(qa) = &config.keybinds.quick_activate {
        qa.iter().enumerate().for_each(|(k, s)| {
            let action_str = format!("{ACTION_QUICK_ACTIVATE}:{k}");

            parse_bind(
                &Action {
                    default: None,
                    action: action_str,
                    global: Some(true),
                    bind: Some(s.clone()),
                    label: Some("quick activate".to_string()),
                    after: None,
                },
                "",
            )
            .unwrap();
        });
    }
}

fn parse_bind(b: &Action, provider: &str) -> Result<(), Box<dyn std::error::Error>> {
    let mut fields = b.bind.as_ref().unwrap().split_whitespace().peekable();

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
                    field,
                    b.bind.as_ref().unwrap()
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
        binds.entry(key).or_default().insert(modifier, b.clone());
        return Ok(());
    }

    if !b.global.unwrap_or(false) {
        let mut provider_binds = PROVIDER_BINDS.write().unwrap();
        provider_binds
            .entry(provider.to_string())
            .or_default()
            .entry(key)
            .or_default()
            .entry(modifier)
            .or_default()
            .push(b.clone());
    } else {
        let mut provider_binds = PROVIDER_GLOBAL_BINDS.write().unwrap();
        provider_binds
            .entry(provider.to_string())
            .or_default()
            .entry(key)
            .or_default()
            .entry(modifier)
            .or_default()
            .push(b.clone());
    }

    Ok(())
}

pub fn get_bind(key: Key, modifier: gdk::ModifierType) -> Option<Action> {
    if get_config().debug {
        if modifier != gdk::ModifierType::empty() {
            let mut modifiers = Vec::new();

            modifier.iter().for_each(|mt| {
                let m = if let Some((key, _)) = MODIFIERS.iter().find(|&(_, &v)| v == mt) {
                    key
                } else {
                    "modifier not supported"
                };

                modifiers.push(m);
            });

            println!("bind: {} {}", modifiers.join(" "), key.name().unwrap());
        } else {
            println!("bind: {}", key.name().unwrap());
        }
    }

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
    actions: &[String],
) -> Option<Action> {
    let action = PROVIDER_BINDS
        .read()
        .ok()?
        .get(provider)?
        .get(&key.to_lower())?
        .get(&modifier)?
        .iter()
        .find(|action| actions.contains(&action.action))
        .cloned();

    if actions.len() == 1 && action.is_none() {
        return Some(Action {
            action: actions.first().unwrap().to_string(),
            global: None,
            default: Some(true),
            bind: Some("Return".to_string()),
            after: None,
            label: None,
        });
    }

    action
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
