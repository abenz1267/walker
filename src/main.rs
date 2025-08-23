mod config;
mod data;
mod keybinds;
mod preview;
mod protos;
mod renderers;
mod state;
mod theme;
mod ui;
use gtk4::gio;
use gtk4::gio::prelude::{ApplicationCommandLineExt, DataInputStreamExtManual};
use gtk4::glib::ControlFlow;
use gtk4::glib::object::ObjectExt;
use gtk4::glib::subclass::types::ObjectSubclassIsExt;
use gtk4::prelude::{EditableExt, EntryExt, GtkWindowExt};

use config::get_config;
use state::{init_app_state, with_state};

use std::sync::{Mutex, mpsc};
use std::time::Duration;
use std::{path::Path, thread};

use gtk4::{
    Application,
    gio::{
        ApplicationFlags,
        prelude::{ApplicationExt, ApplicationExtManual},
    },
    glib::{self, OptionFlags, VariantTy},
    prelude::WidgetExt,
};

use crate::data::{init_socket, start_listening};
use crate::keybinds::setup_binds;
use crate::protos::generated_proto::query::{QueryResponse, query_response};
use crate::renderers::setup_item_transformers;
use crate::theme::{setup_css, setup_css_provider, setup_themes};
use crate::ui::window::{handle_preview, quit, setup_window, with_window};

// GObject wrapper for QueryResponse
mod imp {
    use crate::protos::generated_proto::query::QueryResponse;

    use super::*;
    use gtk4::subclass::prelude::*;
    use std::cell::RefCell;

    #[derive(Debug, Default)]
    pub struct QueryResponseObject {
        pub response: RefCell<Option<QueryResponse>>,
    }

    #[glib::object_subclass]
    impl ObjectSubclass for QueryResponseObject {
        const NAME: &'static str = "QueryResponseObject";
        type Type = super::QueryResponseObject;
    }

    impl ObjectImpl for QueryResponseObject {}
}

glib::wrapper! {
    pub struct QueryResponseObject(ObjectSubclass<imp::QueryResponseObject>);
}

impl QueryResponseObject {
    pub fn new(response: crate::protos::generated_proto::query::QueryResponse) -> Self {
        let obj: Self = glib::Object::builder().build();
        obj.imp().response.replace(Some(response));
        obj
    }

    pub fn response(&self) -> crate::protos::generated_proto::query::QueryResponse {
        self.imp().response.borrow().as_ref().unwrap().clone()
    }
}

fn wait_for_file(path: &str) {
    while !Path::new(path).exists() {
        thread::sleep(Duration::from_millis(10));
    }
}

thread_local! {
    static GLOBAL_DMENU_SENDER: Mutex<Option<mpsc::Sender<String>>> = Mutex::new(None);
}

fn main() -> glib::ExitCode {
    let app = Application::builder()
        .application_id("dev.benz.walker")
        .flags(ApplicationFlags::HANDLES_COMMAND_LINE)
        .build();

    let hold_guard = std::cell::RefCell::new(None);

    app.connect_handle_local_options(|_app, _dict| return -1);

    app.add_main_option(
        "version",
        b'v'.into(),
        OptionFlags::NONE,
        glib::OptionArg::None,
        "show version",
        None,
    );

    app.add_main_option(
        "nosearch",
        b'n'.into(),
        OptionFlags::NONE,
        glib::OptionArg::None,
        "hide search input",
        None,
    );

    app.add_main_option(
        "provider",
        b'p'.into(),
        OptionFlags::NONE,
        glib::OptionArg::String,
        "exclusive provider to query",
        None,
    );

    app.add_main_option(
        "placeholder",
        b'i'.into(),
        OptionFlags::NONE,
        glib::OptionArg::String,
        "input placeholder",
        None,
    );

    app.add_main_option(
        "height",
        b'h'.into(),
        OptionFlags::NONE,
        glib::OptionArg::Int64,
        "forced height",
        None,
    );

    app.add_main_option(
        "current",
        b'c'.into(),
        OptionFlags::NONE,
        glib::OptionArg::Int64,
        "mark current value. dmenu only.",
        None,
    );

    app.add_main_option(
        "width",
        b'w'.into(),
        OptionFlags::NONE,
        glib::OptionArg::Int64,
        "forced width",
        None,
    );

    app.add_main_option(
        "theme",
        b't'.into(),
        OptionFlags::NONE,
        glib::OptionArg::String,
        "theme to use",
        None,
    );

    app.add_main_option(
        "dmenu",
        b'd'.into(),
        OptionFlags::NONE,
        glib::OptionArg::None,
        "dmenu",
        None,
    );

    app.add_main_option(
        "close",
        b'q'.into(),
        OptionFlags::NONE,
        glib::OptionArg::None,
        "closes walker when open",
        None,
    );

    app.add_main_option(
        "keepopen",
        b'k'.into(),
        OptionFlags::NONE,
        glib::OptionArg::None,
        "keep walker open after selection. only when using service. dmenu only.",
        None,
    );

    app.add_main_option(
        "exit",
        b'e'.into(),
        OptionFlags::NONE,
        glib::OptionArg::None,
        "exit after this dmenu call. only when using service. dmenu only",
        None,
    );

    app.connect_command_line(|app, cmd| {
        let options = cmd.options_dict();

        if options.contains("version") {
            cmd.print_literal("1.0.0-beta-7\n");
            return 0;
        }

        with_state(|s| {
            if options.contains("provider") {
                if let Some(val) = options.lookup_value("provider", Some(VariantTy::STRING)) {
                    s.set_provider(val.str().unwrap());
                }
            }

            s.set_param_close(options.contains("close"));

            if options.contains("theme") {
                if let Some(val) = options.lookup_value("theme", Some(VariantTy::STRING)) {
                    s.set_theme(val.str().unwrap());
                }
            }

            if options.contains("height") {
                if let Some(val) = options.lookup_value("height", Some(VariantTy::INT64)) {
                    s.set_parameter_height(val.get::<i64>().unwrap() as i32);
                }
            }

            if options.contains("width") {
                if let Some(val) = options.lookup_value("width", Some(VariantTy::INT64)) {
                    s.set_parameter_width(val.get::<i64>().unwrap() as i32);
                }
            }

            s.set_no_search(options.contains("nosearch"));

            if options.contains("dmenu") {
                if options.contains("placeholder") {
                    if let Some(val) = options.lookup_value("placeholder", Some(VariantTy::STRING))
                    {
                        s.set_placeholder(val.str().unwrap());
                    }
                }

                if options.contains("keepopen")
                    && app.flags().contains(ApplicationFlags::IS_SERVICE)
                {
                    s.set_dmenu_keep_open(true);
                }

                if options.contains("current") {
                    if let Some(val) = options.lookup_value("current", Some(VariantTy::INT64)) {
                        s.set_dmenu_current(val.get::<i64>().unwrap());
                    }
                }

                s.set_dmenu_exit_after(options.contains("exit"));

                let mut exists = false;

                GLOBAL_DMENU_SENDER.with(|sender| {
                    if sender.lock().unwrap().is_some() {
                        send_message("CNCLD".to_string()).unwrap();
                        exists = true;
                    }
                });

                if !exists {
                    let stdin = cmd.stdin();

                    let data_stream = gio::DataInputStream::new(&stdin.unwrap());

                    let mut i = 0;

                    with_window(|w| {
                        if let Some(input) = &w.input {
                            input.set_text("");
                        }

                        let items = &w.items;
                        items.remove_all();

                        loop {
                            match data_stream.read_line(gio::Cancellable::NONE) {
                                Ok(line_slice) => {
                                    if line_slice.is_empty() {
                                        break;
                                    }

                                    if let Ok(line_str) = std::str::from_utf8(&line_slice) {
                                        let trimmed = line_str.trim();
                                        if !trimmed.is_empty() {
                                            let mut item = query_response::Item::new();
                                            item.text = trimmed.to_string();
                                            item.provider = "dmenu".to_string();
                                            item.score = 1000000 - i;

                                            let mut response = QueryResponse::new();
                                            response.item = protobuf::MessageField::some(item);

                                            items.append(&QueryResponseObject::new(response));
                                            i += 1;
                                        }
                                    }
                                }
                                Err(e) => {
                                    eprintln!("Error reading: {}", e);
                                    break;
                                }
                            }
                        }
                    });

                    s.set_is_dmenu(true);

                    if s.is_service() {
                        let (sender, receiver) = mpsc::channel::<String>();

                        GLOBAL_DMENU_SENDER.with(|s| {
                            *s.lock().unwrap() = Some(sender);
                        });

                        let cmd_clone = cmd.clone();

                        glib::idle_add_local(move || match receiver.try_recv() {
                            Ok(message) => {
                                match message.as_str() {
                                    "CNCLD" => {
                                        cmd_clone.set_exit_status(130);
                                    }
                                    msg => cmd_clone.print_literal(msg),
                                };

                                GLOBAL_DMENU_SENDER.with(|s| {
                                    *s.lock().unwrap() = None;
                                });

                                ControlFlow::Break
                            }
                            Err(mpsc::TryRecvError::Empty) => ControlFlow::Continue,
                            Err(mpsc::TryRecvError::Disconnected) => {
                                cmd_clone.set_exit_status(130);
                                ControlFlow::Break
                            }
                        });
                    }
                }
            } else {
                s.set_dmenu_keep_open(false);
            }
        });

        app.activate();
        return 0;
    });

    app.connect_activate(move |app| {
        with_state(|s| {
            let cfg = get_config();

            if (cfg.close_when_open && s.is_visible() && !s.is_dmenu_keep_open())
                || s.is_param_close()
            {
                quit(app, false);
            } else if !s.is_dmenu_keep_open() || !s.is_visible() {
                let provider = s.get_provider();
                let p;

                if provider.is_empty() {
                    p = "default".to_string();
                } else {
                    p = provider.clone();
                }

                drop(provider);

                with_window(|w| {
                    if let Some(placeholders) = &cfg.placeholders {
                        if let Some(placeholder) = placeholders.get(&p) {
                            if let Some(input) = &w.input {
                                input.set_placeholder_text(Some(&placeholder.input));
                            }

                            w.placeholder
                                .as_ref()
                                .map(|p| p.set_text(&placeholder.list));
                        }
                    }

                    if !s.get_placeholder().is_empty() {
                        if let Some(input) = &w.input {
                            if let Some(p) = input.placeholder_text() {
                                s.set_initial_placeholder(&p);
                            }

                            input.set_placeholder_text(Some(&s.get_placeholder()));
                        }
                    }

                    if s.get_parameter_height() != 0 {
                        s.set_initial_height(w.scroll.max_content_height());
                        w.scroll.set_max_content_height(s.get_parameter_height());
                        w.scroll.set_min_content_height(s.get_parameter_height());
                    } else {
                        s.set_initial_height(0);
                    }

                    if s.get_parameter_width() != 0 {
                        s.set_initial_width(w.scroll.max_content_width());
                        w.scroll.set_max_content_width(s.get_parameter_width());
                        w.scroll.set_min_content_width(s.get_parameter_width());
                    } else {
                        s.set_initial_width(0);
                    }

                    if s.is_no_search() {
                        if let Some(search_container) = &w.search_container {
                            search_container.set_visible(false);
                        }
                    }

                    setup_css(s.get_theme());

                    if let Some(input) = &w.input {
                        input.emit_by_name::<()>("changed", &[]);
                        input.grab_focus();
                    }

                    w.window.present();
                });

                s.set_is_visible(true);
            }
        });
    });

    app.connect_startup(move |app| {
        *hold_guard.borrow_mut() = Some(app.hold());

        // if !app.flags().contains(ApplicationFlags::IS_SERVICE) {
        //     println!("make sure 'walker --gapplication-service' is running!");
        //     process::exit(1);
        // }

        init_app_state();
        init_ui(app);
    });

    app.run()
}

fn init_ui(app: &Application) {
    if app.flags().contains(ApplicationFlags::IS_SERVICE) {
        with_state(|s| {
            s.set_is_service(true);
        });
    }

    with_state(|s| {
        if !s.is_dmenu() {
            println!("Waiting for elephant to start...");
        }

        wait_for_file("/tmp/elephant.sock");

        if !s.is_dmenu() {
            println!("Elephant started!");
        }

        config::load().unwrap();

        let theme = if get_config().theme.is_empty() {
            "default"
        } else {
            &get_config().theme
        };

        s.set_theme(&theme);

        preview::load_previewers();
        setup_binds().unwrap();

        init_socket().unwrap();
        start_listening();

        setup_css_provider();
        setup_themes();
        setup_item_transformers();
        setup_window(app);

        // start_theme_watcher(s.get_theme());
    });
}

fn send_message(message: String) -> Result<(), String> {
    GLOBAL_DMENU_SENDER.with(|sender| {
        let sender_guard = sender.lock().unwrap();
        if let Some(tx) = sender_guard.as_ref() {
            tx.send(message)
                .map_err(|_| "Failed to send message".to_string())
        } else {
            Err("No sender available".to_string())
        }
    })
}
