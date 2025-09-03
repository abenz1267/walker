use std::{collections::HashMap, fmt::Debug, process::Command, sync::OnceLock};

use crate::{
    keybinds::Keybind,
    providers::{
        archlinuxpkgs::ArchLinuxPkgs, calc::Calc, clipboard::Clipboard,
        desktopapplications::DesktopApplications, dmenu::Dmenu, files::Files, menus::Menus,
        providerlist::Providerlist, runner::Runner, symbols::Symbols, todo::Todo, unicode::Unicode,
        websearch::Websearch,
    },
};

pub mod archlinuxpkgs;
pub mod calc;
pub mod clipboard;
pub mod desktopapplications;
pub mod dmenu;
pub mod files;
pub mod menus;
pub mod providerlist;
pub mod runner;
pub mod symbols;
pub mod todo;
pub mod unicode;
pub mod websearch;

pub trait Provider: Sync + Send + Debug {
    fn get_keybinds(&self) -> &Vec<Keybind>;
    fn default_action(&self) -> &str;
}

pub static PROVIDERS: OnceLock<HashMap<String, Box<dyn Provider>>> = OnceLock::new();

pub fn setup_providers() {
    let mut providers: HashMap<String, Box<dyn Provider>> = HashMap::new();

    let output = Command::new("elephant")
        .arg("listproviders")
        .output()
        .expect("couldn't run 'elephant'. Make sure it is installed.");

    let stdout = String::from_utf8(output.stdout).unwrap();

    stdout
        .lines()
        .filter_map(|line| line.split_once(';').map(|(_, value)| value.to_string()))
        .for_each(|p| match p.as_str() {
            "calc" => {
                providers.insert("calc".to_string(), Box::new(Calc::new()));
            }
            "clipboard" => {
                providers.insert("clipboard".to_string(), Box::new(Clipboard::new()));
            }
            "desktopapplications" => {
                providers.insert(
                    "desktopapplications".to_string(),
                    Box::new(DesktopApplications::new()),
                );
            }
            "files" => {
                providers.insert("files".to_string(), Box::new(Files::new()));
            }
            "runner" => {
                providers.insert("runner".to_string(), Box::new(Runner::new()));
            }
            "symbols" => {
                providers.insert("symbols".to_string(), Box::new(Symbols::new()));
            }
            "unicode" => {
                providers.insert("unicode".to_string(), Box::new(Unicode::new()));
            }
            "providerlist" => {
                providers.insert("providerlist".to_string(), Box::new(Providerlist::new()));
            }
            "menus" => {
                providers.insert("menus".to_string(), Box::new(Menus::new()));
            }
            "websearch" => {
                providers.insert("websearch".to_string(), Box::new(Websearch::new()));
            }
            "archlinuxpkgs" => {
                providers.insert("archlinuxpkgs".to_string(), Box::new(ArchLinuxPkgs::new()));
            }
            "todo" => {
                providers.insert("todo".to_string(), Box::new(Todo::new()));
            }
            _ => {}
        });

    providers.insert("dmenu".to_string(), Box::new(Dmenu::new()));

    PROVIDERS
        .set(providers)
        .expect("couldn't initialize providers.")
}
