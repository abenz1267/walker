use std::process::{Command, Stdio};
use crate::config::get_config;
use crate::keybinds::Action;
use crate::protos::generated_proto::query::QueryResponse;
use crate::state::{get_prefix_provider, get_provider};

/// Event types that can trigger user-defined commands
#[derive(Debug, Clone, Copy)]
pub enum Event {
    Launch,
    Selection,
    Activate,
    QueryChange,
    Exit,
}

impl Event {
    /// Get the command string from config for this event type
    fn get_command<'a>(&self, config: &'a crate::config::Walker) -> Option<&'a String> {
        match self {
            Event::Launch => config.events.launch.as_ref(),
            Event::Selection => config.events.selection.as_ref(),
            Event::Activate => config.events.activate.as_ref(),
            Event::QueryChange => config.events.query_change.as_ref(),
            Event::Exit => config.events.exit.as_ref(),
        }
    }

    /// Get the event name as a string (for WALKER_EVENT env var)
    fn as_str(&self) -> &'static str {
        match self {
            Event::Launch => "launch",
            Event::Selection => "selection",
            Event::Activate => "activate",
            Event::QueryChange => "query_change",
            Event::Exit => "exit",
        }
    }
}

/// Run event command with environment variables
/// 
/// Environment variables are necessary because they provide context
/// to user-defined scripts about what triggered the event:
/// - WALKER_EVENT: which event occurred
/// - WALKER_QUERY: current search query
/// - WALKER_PROVIDER: active provider
/// - WALKER_SELECTION_TEXT: selected item text
/// - WALKER_ACTION: action being performed
/// - WALKER_EXIT_CANCELLED: whether exit was cancelled
fn run_event(event: Event, mut envs: Vec<(&str, String)>) {
    let config = get_config();
    
    let Some(cmd) = event.get_command(config).filter(|cmd| !cmd.trim().is_empty()) else {
        return;
    };

    let mut command = Command::new("sh");
    command
        .arg("-c")
        .arg(cmd)
        .stdin(Stdio::null())  // Needed: prevents child from reading stdin
        .stdout(Stdio::null())
        .stderr(Stdio::null());

    command.env("WALKER_EVENT", event.as_str());

    for (key, value) in envs.drain(..) {
        command.env(key, value);
    }

    if let Err(err) = command.spawn() {
        eprintln!("Failed to run event '{}': {}", event.as_str(), err);
    }
}

pub fn emit_launch() {
    run_event(Event::Launch, Vec::new());
}

pub fn emit_selection(selection: Option<QueryResponse>, query: &str) {
    let mut envs = vec![("WALKER_QUERY", query.to_string())];

    if let Some(selection) = selection {
        if let Some(item) = selection.item.as_ref() {
            if !item.text.is_empty() {
                envs.push(("WALKER_SELECTION_TEXT", item.text.clone()));
            }
            if !item.provider.is_empty() {
                envs.push(("WALKER_PROVIDER", item.provider.clone()));
            }
        }
    }

    run_event(Event::Selection, envs);
}

pub fn emit_query_change(query: &str) {
    let mut envs = vec![("WALKER_QUERY", query.to_string())];

    let provider = {
        let active = get_provider();
        if !active.is_empty() {
            active
        } else {
            get_prefix_provider()
        }
    };

    if !provider.is_empty() {
        envs.push(("WALKER_PROVIDER", provider));
    }

    run_event(Event::QueryChange, envs);
}

pub fn emit_activate(
    selection: Option<&QueryResponse>,
    provider: &str,
    query: &str,
    action: &Action,
) {
    let mut envs = vec![
        ("WALKER_QUERY", query.to_string()),
        ("WALKER_ACTION", action.action.clone()),
    ];

    let mut provider_name = provider.to_string();

    if let Some(selection) = selection {
        if let Some(item) = selection.item.as_ref() {
            if !item.text.is_empty() {
                envs.push(("WALKER_SELECTION_TEXT", item.text.clone()));
            }
            if !item.provider.is_empty() {
                provider_name = item.provider.clone();
            }
        }
    }

    if !provider_name.is_empty() {
        envs.push(("WALKER_PROVIDER", provider_name));
    }

    run_event(Event::Activate, envs);
}

pub fn emit_exit(cancelled: bool) {
    let envs = vec![("WALKER_EXIT_CANCELLED", cancelled.to_string())];
    run_event(Event::Exit, envs);
}