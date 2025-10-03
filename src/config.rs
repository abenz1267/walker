use config::{Config, ConfigError, File, FileFormat};
use serde::{Deserialize, Serialize};
use std::{collections::HashMap, sync::OnceLock};

use crate::keybinds::{Action, AfterAction};

static LOADED_CONFIG: OnceLock<Walker> = OnceLock::new();
const DEFAULT_CONFIG: &str = include_str!("../resources/config.toml");

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Walker {
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

impl Walker {
    pub fn new() -> Result<Self, ConfigError> {
        let settings = Config::builder()
            .add_source(File::from_str(DEFAULT_CONFIG, FileFormat::Toml))
            .add_source(File::with_name(&get_user_config_path()).required(false))
            .add_source(config::Environment::with_prefix("WALKER"))
            .build()?;

        settings.try_deserialize()
    }
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
