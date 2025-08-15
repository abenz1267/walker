use gtk4::{glib, glib::SourceId};
use std::cell::RefCell;
use std::rc::Rc;
use std::time::Duration;

#[derive(Debug)]
pub struct LatestOnlyThrottler {
    state: Rc<RefCell<ThrottlerState>>,
    timeout_id: Rc<RefCell<Option<SourceId>>>,
    interval: Duration,
}

#[derive(Debug)]
struct ThrottlerState {
    latest_call: Option<String>,
    pending: bool,
}

impl LatestOnlyThrottler {
    pub fn new(interval: Duration) -> Self {
        let state = Rc::new(RefCell::new(ThrottlerState {
            latest_call: None,
            pending: false,
        }));

        let timeout_id = Rc::new(RefCell::new(None));

        Self {
            state,
            timeout_id,
            interval,
        }
    }

    pub fn execute<F>(&self, file: &str, callback: F)
    where
        F: Fn(&str) + 'static,
    {
        {
            let mut state = self.state.borrow_mut();
            state.latest_call = Some(file.to_string());

            if state.pending {
                return;
            }
            state.pending = true;
        }

        if let Some(id) = self.timeout_id.borrow_mut().take() {
            id.remove();
        }

        let state_clone = Rc::clone(&self.state);
        let timeout_id_clone = Rc::clone(&self.timeout_id);
        let interval = self.interval;

        let timeout_id = glib::timeout_add_local(interval, move || {
            let latest_call = {
                let mut state = state_clone.borrow_mut();
                state.pending = false;
                state.latest_call.take()
            };

            if let Some(file_path) = latest_call {
                callback(&file_path);
            }

            *timeout_id_clone.borrow_mut() = None;

            glib::ControlFlow::Break
        });

        *self.timeout_id.borrow_mut() = Some(timeout_id);
    }
}

impl Drop for LatestOnlyThrottler {
    fn drop(&mut self) {
        if let Some(id) = self.timeout_id.borrow_mut().take() {
            id.remove();
        }
    }
}
