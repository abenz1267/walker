use crate::config::{self, get_config};
use gtk4::gdk::{self, Key, ModifierType};
use std::collections::HashMap;
use std::sync::{Arc, Mutex, OnceLock};

// Constants
pub const ACTION_CLOSE: &str = "%CLOSE%";
pub const ACTION_SELECT_NEXT: &str = "%NEXT%";
pub const ACTION_SELECT_PREVIOUS: &str = "%PREVIOUS%";
pub const ACTION_TOGGLE_EXACT: &str = "%TOGGLE_EXACT%";

pub const AFTER_CLOSE: &str = "%CLOSE%";
pub const AFTER_NOTHING: &str = "%NOTHING%";
pub const AFTER_RELOAD: &str = "%RELOAD%";
pub const AFTER_CLEAR_RELOAD: &str = "%CLEAR_RELOAD%";

pub const ACTION_CALC_COPY: &str = "copy";
pub const ACTION_CALC_DELETE: &str = "delete";
pub const ACTION_CALC_SAVE: &str = "save";

pub const ACTION_CLIPBOARD_COPY: &str = "copy";
pub const ACTION_CLIPBOARD_DELETE: &str = "remove";

pub const ACTION_DESKTOP_APPLICATIONS_START: &str = "";

pub const ACTION_FILES_COPY: &str = "copyfile";
pub const ACTION_FILES_COPY_PATH: &str = "copypath";
pub const ACTION_FILES_OPEN: &str = "open";
pub const ACTION_FILES_OPEN_DIR: &str = "opendir";

pub const ACTION_RUNNER_START: &str = "run";
pub const ACTION_RUNNER_START_TERMINAL: &str = "runterminal";

pub const ACTION_SYMBOLS_COPY: &str = "copy";

pub const ACTION_PROVIDERLIST_ACTIVATE: &str = "activate";

pub const ACTION_MENUES_ACTIVATE: &str = "activate";

#[derive(Debug, Clone)]
pub struct Action {
    pub action: String,
    pub after: String,
}

static BINDS: OnceLock<Arc<Mutex<HashMap<Key, HashMap<gdk::ModifierType, Action>>>>> =
    OnceLock::new();
static PROVIDER_BINDS: OnceLock<
    Arc<Mutex<HashMap<String, HashMap<Key, HashMap<gdk::ModifierType, Action>>>>>,
> = OnceLock::new();

fn get_binds() -> &'static Arc<Mutex<HashMap<Key, HashMap<gdk::ModifierType, Action>>>> {
    BINDS.get_or_init(|| Arc::new(Mutex::new(HashMap::new())))
}

fn get_provider_binds()
-> &'static Arc<Mutex<HashMap<String, HashMap<Key, HashMap<gdk::ModifierType, Action>>>>> {
    PROVIDER_BINDS.get_or_init(|| Arc::new(Mutex::new(HashMap::new())))
}

pub fn get_modifiers() -> HashMap<&'static str, gdk::ModifierType> {
    let mut map = HashMap::new();
    map.insert("ctrl", gdk::ModifierType::CONTROL_MASK);
    map.insert("lctrl", gdk::ModifierType::CONTROL_MASK);
    map.insert("rctrl", gdk::ModifierType::CONTROL_MASK);
    map.insert("alt", gdk::ModifierType::ALT_MASK);
    map.insert("lalt", gdk::ModifierType::ALT_MASK);
    map.insert("ralt", gdk::ModifierType::ALT_MASK);
    map.insert("lshift", gdk::ModifierType::SHIFT_MASK);
    map.insert("rshift", gdk::ModifierType::SHIFT_MASK);
    map.insert("shift", gdk::ModifierType::SHIFT_MASK);
    map
}

fn get_special_keys() -> HashMap<&'static str, Key> {
    let mut map = HashMap::new();
    map.insert("backspace", gdk::Key::BackSpace);
    map.insert("tab", gdk::Key::Tab);
    map.insert("esc", gdk::Key::Escape);
    map.insert("escape", gdk::Key::Escape);
    map.insert("kpenter", gdk::Key::KP_Enter);
    map.insert("enter", gdk::Key::Return);
    map.insert("down", gdk::Key::Down);
    map.insert("up", gdk::Key::Up);
    map.insert("left", gdk::Key::Left);
    map.insert("right", gdk::Key::Right);
    map
}

pub fn setup_binds() -> Result<(), Box<dyn std::error::Error>> {
    let config = config::get_config().ok_or("Config not loaded")?;

    parse_bind(&config.keybinds.close, ACTION_CLOSE, AFTER_CLOSE, "")?;
    parse_bind(&config.keybinds.next, ACTION_SELECT_NEXT, AFTER_NOTHING, "")?;
    parse_bind(
        &config.keybinds.previous,
        ACTION_SELECT_PREVIOUS,
        AFTER_NOTHING,
        "",
    )?;
    parse_bind(
        &config.keybinds.toggle_exact,
        ACTION_TOGGLE_EXACT,
        AFTER_NOTHING,
        "",
    )?;

    parse_bind(
        &config.providers.clipboard.copy,
        ACTION_CLIPBOARD_COPY,
        AFTER_CLOSE,
        "clipboard",
    )?;
    parse_bind(
        &config.providers.clipboard.delete,
        ACTION_CLIPBOARD_DELETE,
        AFTER_RELOAD,
        "clipboard",
    )?;

    parse_bind(
        &config.providers.calc.copy,
        ACTION_CALC_COPY,
        AFTER_CLOSE,
        "calc",
    )?;
    parse_bind(
        &config.providers.calc.save,
        ACTION_CALC_SAVE,
        AFTER_RELOAD,
        "calc",
    )?;
    parse_bind(
        &config.providers.calc.delete,
        ACTION_CALC_DELETE,
        AFTER_RELOAD,
        "calc",
    )?;

    parse_bind(
        &config.providers.desktop_applications.start,
        ACTION_DESKTOP_APPLICATIONS_START,
        AFTER_CLOSE,
        "desktopapplications",
    )?;

    parse_bind(
        &config.providers.files.copy_file,
        ACTION_FILES_COPY,
        AFTER_CLOSE,
        "files",
    )?;
    parse_bind(
        &config.providers.files.copy_path,
        ACTION_FILES_COPY_PATH,
        AFTER_CLOSE,
        "files",
    )?;
    parse_bind(
        &config.providers.files.open,
        ACTION_FILES_OPEN,
        AFTER_CLOSE,
        "files",
    )?;
    parse_bind(
        &config.providers.files.open_dir,
        ACTION_FILES_OPEN_DIR,
        AFTER_CLOSE,
        "files",
    )?;

    parse_bind(
        &config.providers.runner.start,
        ACTION_RUNNER_START,
        AFTER_CLOSE,
        "runner",
    )?;

    parse_bind(
        &config.providers.runner.start_terminal,
        ACTION_RUNNER_START_TERMINAL,
        AFTER_CLOSE,
        "runner",
    )?;

    parse_bind(
        &config.providers.symbols.copy,
        ACTION_SYMBOLS_COPY,
        AFTER_CLOSE,
        "symbols",
    )?;

    parse_bind(
        &config.providers.providerlist.activate,
        ACTION_PROVIDERLIST_ACTIVATE,
        AFTER_CLEAR_RELOAD,
        "providerlist",
    )?;

    parse_bind(
        &config.providers.menues.activate,
        ACTION_MENUES_ACTIVATE,
        AFTER_CLOSE, // not really?
        "menues",
    )?;

    Ok(())
}

fn validate_bind(bind: &str) -> bool {
    let fields: Vec<&str> = bind.split_whitespace().collect();
    let modifiers = get_modifiers();
    let special_keys = get_special_keys();

    let mut ok = true;

    for field in fields {
        if field.len() > 1 {
            let exists_mod = modifiers.contains_key(field);
            let exists_special = special_keys.contains_key(field);

            if !exists_mod && !exists_special {
                eprintln!("Invalid keybind: {} - key: {}", bind, field);
                ok = false;
            }
        }
    }

    ok
}

fn parse_bind(
    bind: &str,
    action: &str,
    after: &str,
    provider: &str,
) -> Result<(), Box<dyn std::error::Error>> {
    if !validate_bind(bind) {
        return Err("incorrect bind".into());
    }

    let fields: Vec<&str> = bind.split_whitespace().collect();

    if fields.len() == 0 {
        return Err("incorrect bind".into());
    }

    let modifiers_map = get_modifiers();
    let special_keys = get_special_keys();

    let mut modifiers_list = Vec::new();
    let mut key: Option<Key> = None;

    for field in fields {
        if field.len() > 1 {
            if let Some(&modifier) = modifiers_map.get(field) {
                modifiers_list.push(modifier);
            }

            if let Some(&special_key) = special_keys.get(field) {
                key = Some(special_key);
            }
        } else {
            key = Some(Key::from_name(field.chars().next().unwrap().to_string()).unwrap());
        }
    }

    let modifier = modifiers_list
        .iter()
        .fold(gdk::ModifierType::empty(), |acc, &m| acc | m);

    let action_struct = Action {
        action: action.to_string(),
        after: after.to_string(),
    };

    if key.is_some() {
        if provider.is_empty() {
            let mut binds = get_binds().lock().unwrap();
            binds
                .entry(key.unwrap())
                .or_insert_with(HashMap::new)
                .insert(modifier, action_struct);
        } else {
            let mut provider_binds = get_provider_binds().lock().unwrap();
            provider_binds
                .entry(provider.to_string())
                .or_insert_with(HashMap::new)
                .entry(key.unwrap())
                .or_insert_with(HashMap::new)
                .insert(modifier, action_struct);
        }
    } else {
        return Err("incorrect bind".into());
    }

    Ok(())
}

pub fn get_bind(key: Key, modifier: gdk::ModifierType) -> Option<Action> {
    get_binds()
        .lock()
        .unwrap()
        .get(&key)?
        .get(&modifier)
        .cloned()
}

pub fn get_provider_bind(provider: &str, key: Key, modifier: gdk::ModifierType) -> Option<Action> {
    if let Some(cfg) = get_config() {
        let modifiers = get_modifiers();
        let mut modifier = modifier;

        if let Some(keep_open) = modifiers.get(cfg.keep_open_modifier.as_str()) {
            if *keep_open == modifier {
                modifier = gdk::ModifierType::empty();
            }
        }

        get_provider_binds()
            .lock()
            .unwrap()
            .get(provider)?
            .get(&key)?
            .get(&modifier)
            .cloned()
    } else {
        None
    }
}
