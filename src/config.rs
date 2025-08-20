use config::{Config, ConfigError, File, FileFormat};
use serde::{Deserialize, Serialize};
use std::{collections::HashMap, sync::OnceLock};

static LOADED_CONFIG: OnceLock<Elephant> = OnceLock::new();
const DEFAULT_CONFIG: &str = include_str!("../resources/config.toml");
pub const DEFAULT_STYLE: &str = include_str!("../resources/themes/default/style.css");

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Elephant {
    pub close_when_open: bool,
    pub selection_wrap: bool,
    pub global_argument_delimiter: String,
    pub theme: String,
    pub keep_open_modifier: String,
    pub exact_search_prefix: String,
    pub providers: Providers,
    pub keybinds: Keybinds,
    pub additional_theme_location: Option<String>,
    pub placeholders: Option<HashMap<String, Placeholder>>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Placeholder {
    pub input: String,
    pub list: String,
}

impl Elephant {
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

fn get_user_theme_path() -> String {
    dirs::config_dir()
        .map(|mut path| {
            path.push("walker");
            path.push("themes");
            path.to_string_lossy().to_string()
        })
        .unwrap_or_else(|| "~/.config/walker/themes/".to_string())
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Keybinds {
    pub close: String,
    pub next: String,
    pub previous: String,
    pub toggle_exact: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Providers {
    pub default: Vec<String>,
    pub empty: Vec<String>,
    pub prefixes: Vec<Prefix>,
    pub calc: Calc,
    pub providerlist: Providerlist,
    pub clipboard: Clipboard,
    pub desktopapplications: DesktopApplications,
    pub files: Files,
    pub runner: Runner,
    pub symbols: Symbols,
    pub menus: Menus,
    pub websearch: Websearch,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Websearch {
    pub search: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Menus {
    pub activate: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Calc {
    pub copy: String,
    pub save: String,
    pub delete: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Providerlist {
    pub activate: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DesktopApplications {
    pub start: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Runner {
    pub start: String,
    pub start_terminal: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Symbols {
    pub copy: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Files {
    pub open: String,
    pub open_dir: String,
    pub copy_path: String,
    pub copy_file: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Prefix {
    pub prefix: String,
    pub provider: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Clipboard {
    pub time_format: String,
    pub copy: String,
    pub delete: String,
}

pub fn load() -> Result<(), Box<dyn std::error::Error>> {
    LOADED_CONFIG
        .set(Elephant::new()?)
        .map_err(|_| "Failed to set loaded config")?;

    Ok(())
}

pub fn get_config() -> &'static Elephant {
    LOADED_CONFIG.get().expect("config not initialized")
}
