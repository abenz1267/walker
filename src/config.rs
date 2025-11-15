use config::{Config, ConfigError, File, FileFormat};
use serde::{Deserialize, Serialize};
use std::{collections::HashMap, sync::OnceLock};

use crate::{keybinds::Action, state::set_error};

static LOADED_CONFIG: OnceLock<Walker> = OnceLock::new();
const DEFAULT_CONFIG: &str = include_str!("../resources/config.toml");

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct EmergencyEntry {
    pub text: String,
    pub command: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Walker {
    pub debug: bool,
    pub force_keyboard_focus: bool,
    pub disable_mouse: bool,
    pub click_to_close: bool,
    pub close_when_open: bool,
    pub hide_quick_activation: bool,
    pub selection_wrap: bool,
    pub resume_last_query: bool,
    pub global_argument_delimiter: String,
    pub theme: String,
    pub exact_search_prefix: String,
    pub providers: Providers,
    pub installed_providers: Option<Vec<String>>,
    pub emergencies: Option<Vec<EmergencyEntry>>,
    pub keybinds: Keybinds,
    pub shell: Shell,
    pub additional_theme_location: Option<String>,
    pub placeholders: Option<HashMap<String, Placeholder>>,
    pub columns: Option<HashMap<String, u32>>,
    pub page_jump_items: u32,
}

// Partial config for user overrides
#[derive(Debug, Clone, Serialize, Deserialize, Default)]
#[serde(default)]
struct PartialWalker {
    #[serde(skip_serializing_if = "Option::is_none")]
    pub debug: Option<bool>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub resume_last_query: Option<bool>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub emergencies: Option<Vec<EmergencyEntry>>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub force_keyboard_focus: Option<bool>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub disable_mouse: Option<bool>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub hide_quick_activation: Option<bool>,
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
    #[serde(skip_serializing_if = "Option::is_none")]
    pub columns: Option<HashMap<String, u32>>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub page_jump_items: Option<u32>,
}

#[derive(Debug, Clone, Serialize, Deserialize, Default)]
#[serde(default)]
struct PartialProviders {
    #[serde(skip_serializing_if = "Option::is_none")]
    pub default: Option<Vec<String>>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub max_results: Option<i32>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub empty: Option<Vec<String>>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub ignore_preview: Option<Vec<String>>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub prefixes: Option<Vec<Prefix>>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub clipboard: Option<PartialClipboard>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub sets: Option<HashMap<String, ProviderSet>>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub actions: Option<HashMap<String, Vec<Action>>>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub max_results_provider: Option<HashMap<String, i32>>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub argument_delimiter: Option<HashMap<String, String>>,
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
    #[serde(skip_serializing_if = "Option::is_none")]
    pub page_down: Option<Vec<String>>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub page_up: Option<Vec<String>>,
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

        if let Some(user_config_path) =
            xdg::BaseDirectories::with_prefix("walker").find_config_file("config.toml")
        {
            let user_config = Config::builder()
                .add_source(File::from(user_config_path))
                .build()?;

            match user_config.try_deserialize() {
                Ok(res) => config.merge(res),
                Err(error) => {
                    set_error(format!("Config: {error}"));
                    println!("{error}");
                }
            }
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
        if let Some(v) = partial.resume_last_query {
            self.resume_last_query = v;
        }
        if let Some(v) = partial.emergencies {
            self.emergencies = Some(v);
        }
        if let Some(v) = partial.force_keyboard_focus {
            self.force_keyboard_focus = v;
        }
        if let Some(v) = partial.disable_mouse {
            self.disable_mouse = v;
        }
        if let Some(v) = partial.hide_quick_activation {
            self.hide_quick_activation = v;
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
        if let Some(v) = partial.columns {
            self.columns = Some(v);
        }
        if let Some(v) = partial.page_jump_items {
            self.page_jump_items = v;
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
        if let Some(v) = partial.ignore_preview {
            self.ignore_preview = v;
        }
        if let Some(v) = partial.prefixes {
            self.prefixes = v;
        }
        if let Some(v) = partial.sets {
            self.sets = v;
        }
        if let Some(v) = partial.max_results {
            self.max_results = v;
        }

        if let Some(v) = partial.max_results_provider {
            v.iter().for_each(|(key, value)| {
                self.max_results_provider.insert(key.clone(), *value);
            });
        }

        if let Some(v) = partial.argument_delimiter {
            v.iter().for_each(|(key, value)| {
                self.argument_delimiter
                    .insert(key.clone(), value.to_string());
            });
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

                    defaults.retain(|_, v| !v.unset.unwrap_or_default());

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
        if let Some(v) = partial.page_down {
            self.page_down = v;
        }
        if let Some(v) = partial.page_up {
            self.page_up = v;
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

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Keybinds {
    pub close: Vec<String>,
    pub next: Vec<String>,
    pub previous: Vec<String>,
    pub toggle_exact: Vec<String>,
    pub resume_last_query: Vec<String>,
    pub quick_activate: Option<Vec<String>>,
    pub page_down: Vec<String>,
    pub page_up: Vec<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Providers {
    pub default: Vec<String>,
    pub empty: Vec<String>,
    pub ignore_preview: Vec<String>,
    pub max_results: i32,
    pub max_results_provider: HashMap<String, i32>,
    pub argument_delimiter: HashMap<String, String>,
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
