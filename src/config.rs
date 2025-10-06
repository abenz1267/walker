use config::{Config, ConfigError, File, FileFormat};
use serde::{Deserialize, Serialize};
use std::{collections::HashMap, sync::OnceLock};

use crate::keybinds::Action;

static LOADED_CONFIG: OnceLock<Walker> = OnceLock::new();
const DEFAULT_CONFIG: &str = include_str!("../resources/config.toml");

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Walker {
    pub debug: bool,
    pub force_keyboard_focus: bool,
    pub disable_mouse: bool,
    pub click_to_close: bool,
    pub close_when_open: bool,
    pub selection_wrap: bool,
    pub global_argument_delimiter: String,
    pub theme: String,
    pub exact_search_prefix: String,
    pub providers: Providers,
    pub installed_providers: Option<Vec<String>>,
    pub keybinds: Keybinds,
    pub shell: Shell,
    pub additional_theme_location: Option<String>,
    pub placeholders: Option<HashMap<String, Placeholder>>,
}

// Partial config for user overrides
#[derive(Debug, Clone, Serialize, Deserialize, Default)]
#[serde(default)]
struct PartialWalker {
    #[serde(skip_serializing_if = "Option::is_none")]
    pub debug: Option<bool>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub force_keyboard_focus: Option<bool>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub disable_mouse: Option<bool>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub click_to_close: Option<bool>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub close_when_open: Option<bool>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub selection_wrap: Option<bool>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub global_argument_delimiter: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub theme: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub exact_search_prefix: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub providers: Option<PartialProviders>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub installed_providers: Option<Vec<String>>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub keybinds: Option<PartialKeybinds>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub shell: Option<PartialShell>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub additional_theme_location: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub placeholders: Option<HashMap<String, Placeholder>>,
}

#[derive(Debug, Clone, Serialize, Deserialize, Default)]
#[serde(default)]
struct PartialProviders {
    #[serde(skip_serializing_if = "Option::is_none")]
    pub default: Option<Vec<String>>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub empty: Option<Vec<String>>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub prefixes: Option<Vec<Prefix>>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub clipboard: Option<PartialClipboard>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub sets: Option<HashMap<String, ProviderSet>>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub actions: Option<HashMap<String, Vec<Action>>>,
}

#[derive(Debug, Clone, Serialize, Deserialize, Default)]
#[serde(default)]
struct PartialKeybinds {
    #[serde(skip_serializing_if = "Option::is_none")]
    pub close: Option<Vec<String>>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub next: Option<Vec<String>>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub previous: Option<Vec<String>>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub toggle_exact: Option<Vec<String>>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub resume_last_query: Option<Vec<String>>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub quick_activate: Option<Vec<String>>,
}

#[derive(Debug, Clone, Serialize, Deserialize, Default)]
#[serde(default)]
struct PartialShell {
    #[serde(skip_serializing_if = "Option::is_none")]
    pub anchor_top: Option<bool>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub anchor_bottom: Option<bool>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub anchor_left: Option<bool>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub anchor_right: Option<bool>,
}

#[derive(Debug, Clone, Serialize, Deserialize, Default)]
#[serde(default)]
struct PartialClipboard {
    #[serde(skip_serializing_if = "Option::is_none")]
    pub time_format: Option<String>,
}

impl Walker {
    pub fn new() -> Result<Self, ConfigError> {
        let default_config = Config::builder()
            .add_source(File::from_str(DEFAULT_CONFIG, FileFormat::Toml))
            .build()?;

        let mut config: Walker = default_config.try_deserialize()?;

        let user_config_path = get_user_config_path();
        if std::path::Path::new(&user_config_path).exists() {
            let user_config = Config::builder()
                .add_source(File::with_name(&user_config_path))
                .build()?;

            let partial: PartialWalker = user_config.try_deserialize()?;
            config.merge(partial);
        }

        let env_config = Config::builder()
            .add_source(config::Environment::with_prefix("WALKER").separator("_"))
            .build()?;

        if let Ok(partial) = env_config.try_deserialize::<PartialWalker>() {
            config.merge(partial);
        }

        Ok(config)
    }

    fn merge(&mut self, partial: PartialWalker) {
        if let Some(v) = partial.debug {
            self.debug = v;
        }
        if let Some(v) = partial.force_keyboard_focus {
            self.force_keyboard_focus = v;
        }
        if let Some(v) = partial.disable_mouse {
            self.disable_mouse = v;
        }
        if let Some(v) = partial.click_to_close {
            self.click_to_close = v;
        }
        if let Some(v) = partial.close_when_open {
            self.close_when_open = v;
        }
        if let Some(v) = partial.selection_wrap {
            self.selection_wrap = v;
        }
        if let Some(v) = partial.global_argument_delimiter {
            self.global_argument_delimiter = v;
        }
        if let Some(v) = partial.theme {
            self.theme = v;
        }
        if let Some(v) = partial.exact_search_prefix {
            self.exact_search_prefix = v;
        }
        if let Some(v) = partial.installed_providers {
            self.installed_providers = Some(v);
        }
        if let Some(v) = partial.additional_theme_location {
            self.additional_theme_location = Some(v);
        }
        if let Some(v) = partial.placeholders {
            self.placeholders = Some(v);
        }

        if let Some(p) = partial.providers {
            self.providers.merge(p);
        }
        if let Some(k) = partial.keybinds {
            self.keybinds.merge(k);
        }
        if let Some(s) = partial.shell {
            self.shell.merge(s);
        }
    }
}

impl Providers {
    fn merge(&mut self, partial: PartialProviders) {
        if let Some(v) = partial.default {
            self.default = v;
        }
        if let Some(v) = partial.empty {
            self.empty = v;
        }
        if let Some(v) = partial.prefixes {
            self.prefixes = v;
        }
        if let Some(v) = partial.sets {
            self.sets = v;
        }
        if let Some(v) = partial.actions {
            v.iter().for_each(|(key, value)| {
                if !self.actions.contains_key(key) {
                    self.actions.insert(key.clone(), value.clone());
                } else {
                    let mut defaults: HashMap<String, Action> = self
                        .actions
                        .get(key)
                        .unwrap()
                        .iter()
                        .map(|action| (action.action.clone(), action.clone()))
                        .collect();

                    let user: HashMap<String, Action> = value
                        .iter()
                        .map(|action| (action.action.clone(), action.clone()))
                        .collect();

                    defaults.extend(user);

                    self.actions
                        .insert(key.clone(), defaults.into_values().collect());
                }
            });
        }
        if let Some(c) = partial.clipboard {
            self.clipboard.merge(c);
        }
    }
}

impl Keybinds {
    fn merge(&mut self, partial: PartialKeybinds) {
        if let Some(v) = partial.close {
            self.close = v;
        }
        if let Some(v) = partial.next {
            self.next = v;
        }
        if let Some(v) = partial.previous {
            self.previous = v;
        }
        if let Some(v) = partial.toggle_exact {
            self.toggle_exact = v;
        }
        if let Some(v) = partial.resume_last_query {
            self.resume_last_query = v;
        }
        if let Some(v) = partial.quick_activate {
            self.quick_activate = Some(v);
        }
    }
}

impl Shell {
    fn merge(&mut self, partial: PartialShell) {
        if let Some(v) = partial.anchor_top {
            self.anchor_top = v;
        }
        if let Some(v) = partial.anchor_bottom {
            self.anchor_bottom = v;
        }
        if let Some(v) = partial.anchor_left {
            self.anchor_left = v;
        }
        if let Some(v) = partial.anchor_right {
            self.anchor_right = v;
        }
    }
}

impl Clipboard {
    fn merge(&mut self, partial: PartialClipboard) {
        if let Some(v) = partial.time_format {
            self.time_format = v;
        }
    }
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CustomKeybind {}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Shell {
    pub anchor_top: bool,
    pub anchor_bottom: bool,
    pub anchor_left: bool,
    pub anchor_right: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Placeholder {
    pub input: String,
    pub list: String,
}

fn get_user_config_path() -> String {
    dirs::config_dir()
        .map(|mut path| {
            path.push("walker");
            path.push("config.toml");
            path.to_string_lossy().to_string()
        })
        .unwrap_or_else(|| "~/.config/walker/config.toml".to_string())
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Keybinds {
    pub close: Vec<String>,
    pub next: Vec<String>,
    pub previous: Vec<String>,
    pub toggle_exact: Vec<String>,
    pub resume_last_query: Vec<String>,
    pub quick_activate: Option<Vec<String>>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Providers {
    pub default: Vec<String>,
    pub empty: Vec<String>,
    pub prefixes: Vec<Prefix>,
    pub clipboard: Clipboard,
    pub actions: HashMap<String, Vec<Action>>,
    pub sets: HashMap<String, ProviderSet>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ProviderSet {
    pub default: Vec<String>,
    pub empty: Vec<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Prefix {
    pub prefix: String,
    pub provider: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Clipboard {
    pub time_format: String,
}

pub fn load() -> Result<(), Box<dyn std::error::Error>> {
    LOADED_CONFIG
        .set(Walker::new()?)
        .map_err(|_| "Failed to set loaded config".into())
}

pub fn get_config() -> &'static Walker {
    LOADED_CONFIG.get().expect("config not initialized")
}
