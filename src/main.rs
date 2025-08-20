mod config;
mod data;
mod keybinds;
mod preview;
mod protos;
mod renderers;
mod state;
mod theme;
mod ui;

use gtk4::gio::prelude::ApplicationCommandLineExt;
use gtk4::glib::object::ObjectExt;
use gtk4::glib::subclass::types::ObjectSubclassIsExt;
use gtk4::prelude::{EntryExt, GtkWindowExt};

use config::get_config;
use state::{init_app_state, with_state};

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
use crate::renderers::setup_item_transformers;
use crate::theme::{setup_css, setup_css_provider, setup_themes, start_theme_watcher};
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
        "height",
        b'h'.into(),
        OptionFlags::NONE,
        glib::OptionArg::Int64,
        "forced height",
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

    app.connect_command_line(|app, cmd| {
        let options = cmd.options_dict();

        if options.contains("version") {
            cmd.print_literal("1.0.0-beta\n");
            return 0;
        }

        if options.contains("provider") {
            with_state(|s| {
                if let Some(val) = options.lookup_value("provider", Some(VariantTy::STRING)) {
                    s.set_provider(val.str().unwrap());
                }
            });
        }

        if options.contains("theme") {
            with_state(|s| {
                if let Some(val) = options.lookup_value("theme", Some(VariantTy::STRING)) {
                    s.set_theme(val.str().unwrap());
                }
            });
        } else {
            with_state(|s| {
                s.set_theme("default");
            });
        }

        if options.contains("height") {
            with_state(|s| {
                if let Some(val) = options.lookup_value("height", Some(VariantTy::INT64)) {
                    s.set_parameter_height(val.get::<i64>().unwrap() as i32);
                }
            });
        }

        if options.contains("width") {
            with_state(|s| {
                if let Some(val) = options.lookup_value("width", Some(VariantTy::INT64)) {
                    s.set_parameter_width(val.get::<i64>().unwrap() as i32);
                }
            });
        }

        if options.contains("nosearch") {
            with_state(|s| {
                s.set_no_search(true);
            });
        }

        app.activate();
        return 0;
    });

    app.connect_activate(move |app| {
        with_state(|s| {
            let cfg = get_config();
            if cfg.close_when_open && s.is_visible() {
                quit(app);
            } else {
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
                            w.input.set_placeholder_text(Some(&placeholder.input));
                            w.placeholder
                                .as_ref()
                                .map(|p| p.set_text(&placeholder.list));
                        }
                    }

                    if s.get_parameter_height() != 0 {
                        w.scroll.set_min_content_height(s.get_parameter_height());
                        w.scroll.set_max_content_height(s.get_parameter_height());
                    }

                    if s.get_parameter_width() != 0 {
                        w.scroll.set_min_content_width(s.get_parameter_width());
                        w.scroll.set_max_content_width(s.get_parameter_width());
                    }

                    if s.is_no_search() {
                        w.search_container.set_visible(false);
                    }

                    setup_css(s.get_theme());

                    w.input.emit_by_name::<()>("changed", &[]);
                    w.input.grab_focus();

                    w.window.present();
                });

                s.set_is_visible(true);
            }
        });
    });

    app.connect_startup(move |app| {
        *hold_guard.borrow_mut() = Some(app.hold());

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

    println!("Waiting for elephant to start...");
    wait_for_file("/tmp/elephant.sock");
    println!("Elephant started!");

    config::load().unwrap();
    preview::load_previewers();
    setup_binds().unwrap();

    init_socket().unwrap();
    start_listening();

    setup_css_provider();
    setup_themes();
    setup_item_transformers();
    setup_window(app);

    with_state(|s| {
        // start_theme_watcher(s.get_theme());

        with_window(|w| {
            s.set_initial_width(w.scroll.max_content_width());
            s.set_initial_height(w.scroll.max_content_height());
        });
    });
}
