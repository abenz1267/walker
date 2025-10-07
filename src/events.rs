use std::process::{Command, Stdio};

use crate::config::get_config;
use crate::keybinds::Action;
use crate::protos::generated_proto::query::QueryResponse;
use crate::state::{get_prefix_provider, get_provider};

/// Commands run in background (fire-and-forget) with no output captured.
fn run_event(event: &str, command: Option<&String>, mut envs: Vec<(&str, String)>) {
    let Some(cmd) = command.filter(|cmd| !cmd.trim().is_empty()) else {
        return;
    };

    let mut command = Command::new("sh");
    command
        .arg("-c")
        .arg(cmd)
        .stdin(Stdio::null())
        .stdout(Stdio::null())
        .stderr(Stdio::null())
        .env("WALKER_EVENT", event);

    for (key, value) in envs.drain(..) {
        command.env(key, value);
    }

    if let Err(err) = command.spawn() {
        eprintln!("Failed to run event '{event}': {err}");
    }
}

/// Emit event when Walker is launched.
pub fn emit_launch() {
    let config = get_config();
    run_event("launch", config.events.launch.as_ref(), Vec::new());
}

pub fn emit_selection(selection: Option<QueryResponse>, query: &str) {
    let config = get_config();
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

    run_event("selection", config.events.selection.as_ref(), envs);
}

pub fn emit_query_change(query: &str) {
    let config = get_config();
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

    run_event("query_change", config.events.query_change.as_ref(), envs);
}

// Emit event when an item is activated.
pub fn emit_activate(
    selection: Option<&QueryResponse>,
    provider: &str,
    query: &str,
    action: &Action,
) {
    let config = get_config();
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

    run_event("activate", config.events.activate.as_ref(), envs);
}

/// Emit event when Walker exit.
pub fn emit_exit(cancelled: bool) {
    let config = get_config();
    let envs = vec![("WALKER_EXIT_CANCELLED", cancelled.to_string())];
    run_event("exit", config.events.exit.as_ref(), envs);
}