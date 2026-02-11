use crate::config::get_config;
use crate::providers::PROVIDERS;
use crate::state::get_global_provider_actions;
use gtk4::gdk::{self, Key};
use serde::{Deserialize, Serialize};
use std::collections::HashMap;
use std::sync::{LazyLock, RwLock};

pub const ACTION_CLOSE: &str = "%CLOSE%";
pub const ACTION_SELECT_LEFT: &str = "%SELECT_LEFT%";
pub const ACTION_SELECT_RIGHT: &str = "%SELECT_RIGHT%";
pub const ACTION_SELECT_UP: &str = "%SELECT_UP%";
pub const ACTION_SELECT_DOWN: &str = "%SELECT_DOWN%";
pub const ACTION_SELECT_NEXT: &str = "%NEXT%";
pub const ACTION_SELECT_PREVIOUS: &str = "%PREVIOUS%";
pub const ACTION_TOGGLE_EXACT: &str = "%TOGGLE_EXACT%";
pub const ACTION_RESUME_LAST_QUERY: &str = "%RESUME_LAST_QUERY%";
pub const ACTION_QUICK_ACTIVATE: &str = "%QUICK_ACTIVATE%";
pub const ACTION_SELECT_PAGE_DOWN: &str = "%PAGE_DOWN%";
pub const ACTION_SELECT_PAGE_UP: &str = "%PAGE_UP%";
pub const ACTION_SHOW_ACTIONS: &str = "%SHOW_ACTIONS%";

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
    pub default: Option<bool>,
    pub unset: Option<bool>,

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
static GRID_BINDS: LazyLock<RwLock<HashMap<Key, HashMap<gdk::ModifierType, Action>>>> =
    LazyLock::new(RwLock::default);
static PROVIDER_BINDS: LazyLock<
    RwLock<HashMap<String, HashMap<Key, HashMap<gdk::ModifierType, Vec<Action>>>>>,
> = LazyLock::new(RwLock::default);
static PROVIDER_BINDS_ACTIONS: LazyLock<RwLock<HashMap<String, Vec<String>>>> =
    LazyLock::new(RwLock::default);

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

            let mut pba = PROVIDER_BINDS_ACTIONS.write().unwrap();

            if let Some(actions) = pba.get_mut(k) {
                actions.push(v.action.clone());
            } else {
                pba.insert(k.clone(), vec![v.action.clone()]);
            }
        });
    });

    let config = get_config();

    config
        .providers
        .actions
        .get("fallback")
        .unwrap_or(&Vec::new())
        .iter()
        .for_each(|v| {
            parse_bind(v, "fallback").unwrap();
        });

    config.keybinds.close.iter().for_each(|b| {
        parse_bind(
            &Action {
                action: ACTION_CLOSE.to_string(),
                unset: None,
                default: Some(true),
                bind: Some(b.clone()),
                label: Some("close".to_string()),
                after: None,
            },
            "",
        )
        .unwrap();
    });

    config.keybinds.show_actions.iter().for_each(|b| {
        parse_bind(
            &Action {
                action: ACTION_SHOW_ACTIONS.to_string(),
                unset: None,
                default: Some(true),
                bind: Some(b.clone()),
                label: Some("actions".to_string()),
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
                unset: None,
                default: None,
                bind: Some(b.clone()),
                label: Some("select next".to_string()),
                after: Some(AfterAction::Nothing),
            },
            "",
        )
        .unwrap();
    });

    config.keybinds.left.iter().for_each(|b| {
        parse_bind(
            &Action {
                action: ACTION_SELECT_LEFT.to_string(),
                unset: None,
                default: None,
                bind: Some(b.clone()),
                label: Some("select left".to_string()),
                after: Some(AfterAction::Nothing),
            },
            "",
        )
        .unwrap();
    });

    config.keybinds.right.iter().for_each(|b| {
        parse_bind(
            &Action {
                action: ACTION_SELECT_RIGHT.to_string(),
                unset: None,
                default: None,
                bind: Some(b.clone()),
                label: Some("select right".to_string()),
                after: Some(AfterAction::Nothing),
            },
            "",
        )
        .unwrap();
    });

    config.keybinds.up.iter().for_each(|b| {
        parse_bind(
            &Action {
                action: ACTION_SELECT_UP.to_string(),
                unset: None,
                default: None,
                bind: Some(b.clone()),
                label: Some("select up".to_string()),
                after: Some(AfterAction::Nothing),
            },
            "",
        )
        .unwrap();
    });

    config.keybinds.down.iter().for_each(|b| {
        parse_bind(
            &Action {
                action: ACTION_SELECT_DOWN.to_string(),
                unset: None,
                default: None,
                bind: Some(b.clone()),
                label: Some("select down".to_string()),
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
                unset: None,
                default: None,
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
                unset: None,
                default: None,
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
                unset: None,
                bind: Some(b.clone()),
                default: None,
                label: Some("resume last query".to_string()),
                after: Some(AfterAction::Nothing),
            },
            "",
        )
        .unwrap();
    });

    config.keybinds.page_down.iter().for_each(|b| {
        parse_bind(
            &Action {
                action: ACTION_SELECT_PAGE_DOWN.to_string(),
                default: None,
                unset: None,
                bind: Some(b.clone()),
                label: Some("select page down".to_string()),
                after: Some(AfterAction::Nothing),
            },
            "",
        )
        .unwrap();
    });

    config.keybinds.page_up.iter().for_each(|b| {
        parse_bind(
            &Action {
                unset: None,
                action: ACTION_SELECT_PAGE_UP.to_string(),
                default: None,
                bind: Some(b.clone()),
                label: Some("select page up".to_string()),
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
                    unset: None,
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
    let mut b = b.clone();

    if let Some((first, _)) = b.action.split_once(":")
        && b.action.ends_with(":keep")
    {
        b.action = first.to_string();
    }

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
        let mut grid_binds = GRID_BINDS.write().unwrap();

        match b.action.as_str() {
            ACTION_SELECT_PREVIOUS | ACTION_SELECT_NEXT => {
                binds.entry(key).or_default().insert(modifier, b.clone());
            }
            ACTION_SELECT_UP | ACTION_SELECT_DOWN | ACTION_SELECT_LEFT | ACTION_SELECT_RIGHT => {
                grid_binds
                    .entry(key)
                    .or_default()
                    .insert(modifier, b.clone());
            }
            _ => {
                binds.entry(key).or_default().insert(modifier, b.clone());
                grid_binds
                    .entry(key)
                    .or_default()
                    .insert(modifier, b.clone());
            }
        };

        return Ok(());
    }

    let mut provider_binds = PROVIDER_BINDS.write().unwrap();

    provider_binds
        .entry(provider.to_string())
        .or_default()
        .entry(key)
        .or_default()
        .entry(modifier)
        .or_default()
        .push(b.clone());

    Ok(())
}

pub fn get_show_actions_action() -> Action {
    BINDS
        .read()
        .unwrap()
        .values()
        .flat_map(|inner| inner.values())
        .find(|a| a.action == ACTION_SHOW_ACTIONS)
        .unwrap()
        .clone()
}

pub fn get_bind(key: Key, modifier: gdk::ModifierType, is_grid: bool) -> Option<Action> {
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

    if is_grid {
        GRID_BINDS
            .read()
            .ok()?
            .get(&key.to_lower())?
            .get(&modifier)
            .cloned()
    } else {
        BINDS
            .read()
            .ok()?
            .get(&key.to_lower())?
            .get(&modifier)
            .cloned()
    }
}

pub fn get_fallback_action(action: &str) -> Option<Action> {
    PROVIDER_BINDS
        .read()
        .unwrap()
        .get("fallback")?
        .values()
        .flat_map(|modifier_map| modifier_map.values())
        .flat_map(|action_vec| action_vec.iter())
        .find(|a| a.action == action)
        .cloned()
}

pub fn get_provider_bind(
    provider: &str,
    key: Key,
    modifier: gdk::ModifierType,
    actions: &[String],
) -> Option<Action> {
    let mut action = None;

    // remove hardcoded global binds for elephant
    let actions: Vec<_> = actions.iter().filter(|a| **a != "menus:parent").collect();

    if let Ok(binds) = PROVIDER_BINDS.read() {
        action = binds
            .get(provider)
            .and_then(|keys| keys.get(&key.to_lower()))
            .and_then(|modifiers| modifiers.get(&modifier))
            .and_then(|actions_list| {
                actions_list
                    .iter()
                    .find(|action| actions.contains(&&action.action))
                    .cloned()
            });

        if action.is_none() {
            let provider_actions = PROVIDER_BINDS_ACTIONS
                .read()
                .ok()
                .and_then(|pba| pba.get(provider).cloned())
                .unwrap_or_default();

            let filtered_actions: Vec<_> = actions
                .iter()
                .filter(|a| !provider_actions.contains(a))
                .collect();

            action = binds
                .get("fallback")
                .and_then(|keys| keys.get(&key.to_lower()))
                .and_then(|modifiers| modifiers.get(&modifier))
                .and_then(|actions_list| {
                    actions_list
                        .iter()
                        .find(|action| filtered_actions.contains(&&&action.action))
                        .cloned()
                });
        }
    }

    if actions.len() == 1 && action.is_none() && key == gdk::Key::Return {
        return Some(Action {
            unset: None,
            action: actions.first().unwrap().to_string(),
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
    let global_actions = get_global_provider_actions()?;

    if let Ok(binds) = PROVIDER_BINDS.read() {
        let mut action = binds
            .get(provider)
            .and_then(|keys| keys.get(&key.to_lower()))
            .and_then(|modifiers| modifiers.get(&modifier))
            .and_then(|actions_list| {
                actions_list
                    .iter()
                    .find(|action| global_actions.contains(&action.action))
                    .cloned()
            });

        if action.is_none() {
            action = binds
                .get("fallback")
                .and_then(|keys| keys.get(&key.to_lower()))
                .and_then(|modifiers| modifiers.get(&modifier))
                .and_then(|actions_list| {
                    actions_list
                        .iter()
                        .find(|action| global_actions.contains(&action.action))
                        .cloned()
                });
        }

        action
    } else {
        None
    }
}
