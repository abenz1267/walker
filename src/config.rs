use serde::{Deserialize, Serialize};
use std::process::Command;
use std::sync::OnceLock;

fn default_true() -> bool {
    true
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Config {
    #[serde(default = "default_true")]
    pub close_when_open: bool,

    #[serde(default = "default_true")]
    pub selection_wrap: bool,

    #[serde(flatten)]
    pub providers: Providers,

    #[serde(flatten)]
    pub keybinds: Keybinds,

    #[serde(flatten)]
    pub positions: Position,

    #[serde(default)]
    pub additional_theme_location: Option<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Position {
    #[serde(default = "default_true")]
    pub anchor_top: bool,

    #[serde(default = "default_true")]
    pub anchor_bottom: bool,

    #[serde(default = "default_true")]
    pub anchor_left: bool,

    #[serde(default = "default_true")]
    pub anchor_right: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Keybinds {
    #[serde(default = "default_close")]
    pub close: String,

    #[serde(default = "default_next")]
    pub next: String,

    #[serde(default = "default_previous")]
    pub previous: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Providers {
    #[serde(default = "default_providers")]
    pub default: Vec<String>,

    #[serde(default = "default_empty")]
    pub empty: Vec<String>,

    #[serde(default = "default_prefixes")]
    pub prefixes: Vec<Prefix>,

    #[serde(flatten)]
    pub calc: Calc,

    #[serde(flatten)]
    pub clipboard: Clipboard,

    #[serde(flatten)]
    pub desktop_applications: DesktopApplications,

    #[serde(flatten)]
    pub files: Files,

    #[serde(flatten)]
    pub runner: Runner,

    #[serde(flatten)]
    pub symbols: Symbols,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Calc {
    #[serde(default = "default_enter")]
    pub copy: String,

    #[serde(default = "default_ctrl_s")]
    pub save: String,

    #[serde(default = "default_ctrl_d")]
    pub delete: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DesktopApplications {
    #[serde(default = "default_enter")]
    pub start: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Runner {
    #[serde(default = "default_enter")]
    pub start: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Symbols {
    #[serde(default = "default_enter")]
    pub copy: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Files {
    #[serde(default = "default_enter")]
    pub open: String,

    #[serde(default = "default_ctrl_enter")]
    pub open_dir: String,

    #[serde(default = "default_ctrl_shift_c")]
    pub copy_path: String,

    #[serde(default = "default_ctrl_c")]
    pub copy_file: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Prefix {
    pub prefix: String,
    pub provider: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Clipboard {
    #[serde(default = "default_time_format")]
    pub time_format: String,

    #[serde(default = "default_enter")]
    pub copy: String,

    #[serde(default = "default_ctrl_d")]
    pub delete: String,
}

#[derive(Debug, Clone)]
pub struct Provider {
    pub name: String,
    pub name_pretty: String,
}

static LOADED_CONFIG: OnceLock<Config> = OnceLock::new();
static AVAILABLE_PROVIDERS: OnceLock<Vec<Provider>> = OnceLock::new();

fn default_close() -> String {
    "escape".to_string()
}
fn default_next() -> String {
    "Down".to_string()
}
fn default_previous() -> String {
    "Up".to_string()
}
fn default_providers() -> Vec<String> {
    vec![
        "desktopapplications".to_string(),
        "calc".to_string(),
        "runner".to_string(),
    ]
}
fn default_empty() -> Vec<String> {
    vec!["desktopapplications".to_string()]
}
fn default_prefixes() -> Vec<Prefix> {
    vec![
        Prefix {
            prefix: "/".to_string(),
            provider: "files".to_string(),
        },
        Prefix {
            prefix: ".".to_string(),
            provider: "symbols".to_string(),
        },
        Prefix {
            prefix: "=".to_string(),
            provider: "calc".to_string(),
        },
        Prefix {
            prefix: ":".to_string(),
            provider: "clipboard".to_string(),
        },
    ]
}
fn default_enter() -> String {
    "enter".to_string()
}
fn default_ctrl_s() -> String {
    "ctrl s".to_string()
}
fn default_ctrl_d() -> String {
    "ctrl d".to_string()
}
fn default_ctrl_enter() -> String {
    "ctrl enter".to_string()
}
fn default_ctrl_shift_c() -> String {
    "ctrl shift C".to_string()
}
fn default_ctrl_c() -> String {
    "ctrl c".to_string()
}
fn default_time_format() -> String {
    "dd.MM. - hh:mm".to_string()
}

impl Default for Config {
    fn default() -> Self {
        Config {
            selection_wrap: true,
            additional_theme_location: None,
            positions: Position {
                anchor_top: true,
                anchor_bottom: true,
                anchor_left: true,
                anchor_right: true,
            },
            close_when_open: true,
            keybinds: Keybinds {
                close: "esc".to_string(),
                next: "down".to_string(),
                previous: "up".to_string(),
            },
            providers: Providers {
                default: vec![
                    "desktopapplications".to_string(),
                    "calc".to_string(),
                    "runner".to_string(),
                ],
                empty: vec!["desktopapplications".to_string()],
                prefixes: vec![
                    Prefix {
                        prefix: "/".to_string(),
                        provider: "files".to_string(),
                    },
                    Prefix {
                        prefix: ".".to_string(),
                        provider: "symbols".to_string(),
                    },
                    Prefix {
                        prefix: "=".to_string(),
                        provider: "calc".to_string(),
                    },
                    Prefix {
                        prefix: ":".to_string(),
                        provider: "clipboard".to_string(),
                    },
                ],
                clipboard: Clipboard {
                    time_format: "dd.MM. - hh:mm".to_string(),
                    copy: "enter".to_string(),
                    delete: "ctrl d".to_string(),
                },
                calc: Calc {
                    copy: "enter".to_string(),
                    save: "ctrl s".to_string(),
                    delete: "ctrl d".to_string(),
                },
                desktop_applications: DesktopApplications {
                    start: "enter".to_string(),
                },
                files: Files {
                    open: "enter".to_string(),
                    open_dir: "ctrl enter".to_string(),
                    copy_path: "ctrl shift C".to_string(),
                    copy_file: "ctrl c".to_string(),
                },
                runner: Runner {
                    start: "enter".to_string(),
                },
                symbols: Symbols {
                    copy: "enter".to_string(),
                },
            },
        }
    }
}

pub fn load() -> Result<(), Box<dyn std::error::Error>> {
    load_provider_data()?;

    let config = Config::default();

    LOADED_CONFIG
        .set(config)
        .map_err(|_| "Failed to set loaded config")?;

    Ok(())
}

pub fn get_config() -> Option<&'static Config> {
    LOADED_CONFIG.get()
}

pub fn get_available_providers() -> Option<&'static Vec<Provider>> {
    AVAILABLE_PROVIDERS.get()
}

fn load_provider_data() -> Result<(), Box<dyn std::error::Error>> {
    let output = Command::new("elephant").arg("listproviders").output()?;

    if !output.status.success() {
        return Err("Command failed".into());
    }

    let mut providers = Vec::new();
    let output_str = String::from_utf8(output.stdout)?;

    for line in output_str.lines() {
        let info = line.trim();
        let parts: Vec<&str> = info.split(':').collect();

        if parts.len() >= 2 {
            providers.push(Provider {
                name: parts[1].to_string(),
                name_pretty: parts[0].to_string(),
            });
        }
    }

    AVAILABLE_PROVIDERS
        .set(providers)
        .map_err(|_| "Failed to set available providers")?;

    Ok(())
}
