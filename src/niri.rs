use niri_ipc::{Event, Request, Response, socket::Socket};
use std::{
    collections::{HashMap, HashSet},
    io, thread,
    time::Duration,
};

#[derive(Default)]
struct WorkspaceTracker {
    workspace_ids: Option<HashSet<u64>>,
    focused_workspace_id: Option<u64>,
    windows: Option<HashMap<u64, u64>>,
    overview_open: Option<bool>,
    pending_launch: bool,
    initial_workspace_checked: bool,
}

impl WorkspaceTracker {
    fn replace_workspaces(&mut self, workspaces: impl IntoIterator<Item = (u64, bool)>) {
        let workspaces = workspaces.into_iter().collect::<Vec<_>>();
        self.focused_workspace_id = workspaces
            .iter()
            .find_map(|(id, focused)| focused.then_some(*id));
        self.workspace_ids = Some(workspaces.into_iter().map(|(id, _)| id).collect());
    }

    fn replace_windows(&mut self, windows: impl IntoIterator<Item = (u64, Option<u64>)>) {
        self.windows = Some(
            windows
                .into_iter()
                .filter_map(|(window_id, workspace_id)| {
                    workspace_id.map(|workspace_id| (window_id, workspace_id))
                })
                .collect(),
        );
    }

    fn update_window(&mut self, window_id: u64, workspace_id: Option<u64>) {
        let Some(windows) = self.windows.as_mut() else {
            return;
        };

        if let Some(workspace_id) = workspace_id {
            windows.insert(window_id, workspace_id);
        } else {
            windows.remove(&window_id);
        }
    }

    fn remove_window(&mut self, window_id: u64) {
        if let Some(windows) = self.windows.as_mut() {
            windows.remove(&window_id);
        }
    }

    fn focused_workspace_is_empty(&self, workspace_id: u64, focused: bool) -> bool {
        focused
            && self
                .workspace_ids
                .as_ref()
                .is_some_and(|ids| ids.contains(&workspace_id))
            && self.windows.as_ref().is_some_and(|windows| {
                windows
                    .values()
                    .all(|window_workspace_id| *window_workspace_id != workspace_id)
            })
    }

    fn request_launch(&mut self) -> bool {
        if self.overview_open == Some(false) {
            true
        } else {
            self.pending_launch = true;
            false
        }
    }

    fn set_overview_open(&mut self, is_open: bool) -> bool {
        self.overview_open = Some(is_open);

        if is_open || !std::mem::take(&mut self.pending_launch) {
            return false;
        }

        self.focused_workspace_id
            .is_some_and(|id| self.focused_workspace_is_empty(id, true))
    }

    fn check_initial_workspace(&mut self, launch_on_startup: bool) -> bool {
        if self.initial_workspace_checked
            || self.workspace_ids.is_none()
            || self.windows.is_none()
            || self.overview_open.is_none()
        {
            return false;
        }

        self.initial_workspace_checked = true;

        let should_launch = launch_on_startup
            && self
                .focused_workspace_id
                .is_some_and(|id| self.focused_workspace_is_empty(id, true));

        should_launch && self.request_launch()
    }

    fn handle_event(
        &mut self,
        event: Event,
        launch_on_empty_workspace: bool,
        launch_on_startup: bool,
    ) -> bool {
        let should_launch = match event {
            Event::WorkspacesChanged { workspaces } => {
                self.replace_workspaces(
                    workspaces
                        .into_iter()
                        .map(|workspace| (workspace.id, workspace.is_focused)),
                );
                false
            }
            Event::WindowsChanged { windows } => {
                self.replace_windows(
                    windows
                        .into_iter()
                        .map(|window| (window.id, window.workspace_id)),
                );
                false
            }
            Event::WindowOpenedOrChanged { window } => {
                self.update_window(window.id, window.workspace_id);
                false
            }
            Event::WindowClosed { id } => {
                self.remove_window(id);
                false
            }
            Event::WorkspaceActivated { id, focused } => {
                if focused {
                    self.focused_workspace_id = Some(id);
                }

                let should_launch =
                    launch_on_empty_workspace && self.focused_workspace_is_empty(id, focused);

                should_launch && self.request_launch()
            }
            Event::OverviewOpenedOrClosed { is_open } => self.set_overview_open(is_open),
            _ => false,
        };

        should_launch || self.check_initial_workspace(launch_on_startup)
    }
}

pub fn watch_empty_workspace_focuses(
    launch_on_empty_workspace: bool,
    mut launch_on_startup: bool,
    mut notify: impl FnMut() + Send + 'static,
) {
    let mut last_error = None;

    loop {
        if let Err(error) = watch_event_stream(
            launch_on_empty_workspace,
            &mut launch_on_startup,
            &mut notify,
        ) {
            let error = error.to_string();

            if last_error.as_deref() != Some(error.as_str()) {
                eprintln!("niri event stream: {error}");
                last_error = Some(error);
            }
        }

        thread::sleep(Duration::from_secs(1));
    }
}

fn watch_event_stream(
    launch_on_empty_workspace: bool,
    launch_on_startup: &mut bool,
    notify: &mut impl FnMut(),
) -> io::Result<()> {
    let mut socket = Socket::connect()?;
    let response = socket.send(Request::EventStream)?;

    match response {
        Ok(Response::Handled) => {}
        Ok(response) => {
            return Err(io::Error::new(
                io::ErrorKind::InvalidData,
                format!("unexpected response: {response:?}"),
            ));
        }
        Err(error) => return Err(io::Error::other(error)),
    }

    let mut tracker = WorkspaceTracker::default();
    let mut read_event = socket.read_events();

    loop {
        let should_notify =
            tracker.handle_event(read_event()?, launch_on_empty_workspace, *launch_on_startup);

        if tracker.initial_workspace_checked {
            *launch_on_startup = false;
        }

        if should_notify {
            notify();
        }
    }
}

#[cfg(test)]
mod tests {
    use super::WorkspaceTracker;
    use niri_ipc::Event;

    fn initialized_tracker() -> WorkspaceTracker {
        let mut tracker = WorkspaceTracker::default();
        tracker.replace_workspaces([(1, false), (2, true)]);
        tracker.replace_windows([]);
        tracker.overview_open = Some(false);
        tracker.initial_workspace_checked = true;
        tracker
    }

    #[test]
    fn empty_focused_workspace_triggers() {
        let tracker = initialized_tracker();

        assert!(tracker.focused_workspace_is_empty(2, true));
    }

    #[test]
    fn populated_focused_workspace_does_not_trigger() {
        let mut tracker = initialized_tracker();
        tracker.update_window(10, Some(2));

        assert!(!tracker.focused_workspace_is_empty(2, true));
    }

    #[test]
    fn inactive_empty_workspace_does_not_trigger() {
        let tracker = initialized_tracker();

        assert!(!tracker.focused_workspace_is_empty(2, false));
    }

    #[test]
    fn snapshots_must_be_initialized_before_triggering() {
        let mut tracker = WorkspaceTracker::default();

        assert!(!tracker.focused_workspace_is_empty(1, true));

        tracker.replace_workspaces([(1, true)]);
        assert!(!tracker.focused_workspace_is_empty(1, true));
    }

    #[test]
    fn unknown_workspace_does_not_trigger() {
        let tracker = initialized_tracker();

        assert!(!tracker.focused_workspace_is_empty(3, true));
    }

    #[test]
    fn closing_last_window_makes_workspace_empty() {
        let mut tracker = initialized_tracker();
        tracker.update_window(10, Some(2));
        tracker.remove_window(10);

        assert!(tracker.focused_workspace_is_empty(2, true));
    }

    #[test]
    fn moving_window_updates_both_workspaces() {
        let mut tracker = initialized_tracker();
        tracker.update_window(10, Some(1));
        tracker.update_window(10, Some(2));

        assert!(tracker.focused_workspace_is_empty(1, true));
        assert!(!tracker.focused_workspace_is_empty(2, true));
    }

    #[test]
    fn configured_empty_initial_workspace_triggers_once() {
        let mut tracker = WorkspaceTracker::default();
        tracker.replace_workspaces([(1, true)]);
        tracker.replace_windows([]);
        tracker.overview_open = Some(false);

        assert!(tracker.check_initial_workspace(true));
        assert!(!tracker.check_initial_workspace(true));
    }

    #[test]
    fn initial_workspace_does_not_trigger_when_disabled() {
        let mut tracker = WorkspaceTracker::default();
        tracker.replace_workspaces([(1, true)]);
        tracker.replace_windows([]);
        tracker.overview_open = Some(false);

        assert!(!tracker.check_initial_workspace(false));
    }

    #[test]
    fn populated_initial_workspace_does_not_trigger() {
        let mut tracker = WorkspaceTracker::default();
        tracker.replace_workspaces([(1, true)]);
        tracker.replace_windows([(10, Some(1))]);
        tracker.overview_open = Some(false);

        assert!(!tracker.check_initial_workspace(true));
    }

    #[test]
    fn empty_workspace_launch_waits_for_overview_to_close() {
        let mut tracker = initialized_tracker();
        tracker.overview_open = Some(true);

        assert!(!tracker.handle_event(
            Event::WorkspaceActivated {
                id: 2,
                focused: true,
            },
            true,
            false,
        ));
        assert!(tracker.pending_launch);
        assert!(tracker.handle_event(
            Event::OverviewOpenedOrClosed { is_open: false },
            true,
            false,
        ));
    }

    #[test]
    fn deferred_launch_is_cancelled_if_workspace_is_no_longer_empty() {
        let mut tracker = initialized_tracker();
        tracker.overview_open = Some(true);

        assert!(!tracker.handle_event(
            Event::WorkspaceActivated {
                id: 2,
                focused: true,
            },
            true,
            false,
        ));
        tracker.update_window(10, Some(2));

        assert!(!tracker.handle_event(
            Event::OverviewOpenedOrClosed { is_open: false },
            true,
            false,
        ));
        assert!(!tracker.pending_launch);
    }

    #[test]
    fn startup_launch_waits_for_initial_overview_state() {
        let mut tracker = WorkspaceTracker::default();
        tracker.replace_workspaces([(1, true)]);
        tracker.replace_windows([]);

        assert!(!tracker.check_initial_workspace(true));
        assert!(!tracker.initial_workspace_checked);

        assert!(tracker.handle_event(
            Event::OverviewOpenedOrClosed { is_open: false },
            false,
            true,
        ));
    }

    #[test]
    fn startup_launch_is_deferred_if_overview_is_initially_open() {
        let mut tracker = WorkspaceTracker::default();
        tracker.replace_workspaces([(1, true)]);
        tracker.replace_windows([]);

        assert!(!tracker.handle_event(
            Event::OverviewOpenedOrClosed { is_open: true },
            false,
            true,
        ));
        assert!(tracker.pending_launch);
        assert!(tracker.handle_event(
            Event::OverviewOpenedOrClosed { is_open: false },
            false,
            false,
        ));
    }
}
