use config::{Config, ConfigError, File, FileFormat};
use serde::{Deserialize, Serialize};
use std::{collections::HashMap, sync::OnceLock};

static LOADED_CONFIG: OnceLock<Elephant> = OnceLock::new();
const DEFAULT_CONFIG: &str = include_str!("../resources/config.toml");

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Elephant {
    pub force_keyboard_focus: bool,
    pub disable_mouse: bool,
    pub close_when_open: bool,
    pub selection_wrap: bool,
    pub global_argument_delimiter: String,
    pub theme: String,
    pub keep_open_modifier: String,
    pub exact_search_prefix: String,
    pub providers: Providers,
    pub keybinds: Keybinds,
    pub shell: Shell,
    pub additional_theme_location: Option<String>,
    pub placeholders: Option<HashMap<String, Placeholder>>,
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

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Keybinds {
    pub close: String,
    pub next: String,
    pub previous: String,
    pub toggle_exact: String,
    pub resume_last_query: String,
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
    pub todo: Todo,
    pub runner: Runner,
    pub symbols: Symbols,
    pub unicode: Unicode,
    pub archlinuxpkgs: ArchLinuxPkgs,
    pub menus: Menus,
    pub websearch: Websearch,
    pub dmenu: Dmenu,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Websearch {
    pub click: String,
    pub search: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Menus {
    pub click: String,
    pub activate: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Calc {
    pub click: String,
    pub copy: String,
    pub save: String,
    pub delete: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Providerlist {
    pub click: String,
    pub activate: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DesktopApplications {
    pub click: String,
    pub start: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Runner {
    pub click: String,
    pub start: String,
    pub start_terminal: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Dmenu {
    pub click: String,
    pub select: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Symbols {
    pub click: String,
    pub copy: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Unicode {
    pub click: String,
    pub copy: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ArchLinuxPkgs {
    pub click: String,
    pub install: String,
    pub remove: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Files {
    pub click: String,
    pub open: String,
    pub open_dir: String,
    pub copy_path: String,
    pub copy_file: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Todo {
    pub click: String,
    pub save: String,
    pub delete: String,
    pub mark_active: String,
    pub mark_done: String,
    pub clear: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Prefix {
    pub prefix: String,
    pub provider: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Clipboard {
    pub click: String,
    pub time_format: String,
    pub copy: String,
    pub delete: String,
    pub edit: String,
    pub toggle_images_only: String,
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
