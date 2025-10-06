mod config;
mod data;
mod keybinds;
mod preview;
mod protos;
mod providers;
mod renderers;
mod state;
mod theme;
mod ui;
use gtk4::gio::prelude::{ApplicationCommandLineExt, DataInputStreamExtManual};
use gtk4::gio::{self, ApplicationCommandLine, ApplicationHoldGuard};
use gtk4::glib::Priority;
use gtk4::glib::object::ObjectExt;
use gtk4::prelude::{EditableExt, EntryExt};

use config::get_config;
use state::init_app_state;
use which::which;

use std::cell::OnceCell;
use std::os::fd::AsRawFd;
use std::os::unix::net::UnixListener;
use std::path::PathBuf;
use std::process;
use std::rc::Rc;
use std::sync::RwLock;
use std::thread;
use std::{env, fs};
use tokio::sync::oneshot::{self, Sender};

use gtk4::{
    Application,
    gio::{
        ApplicationFlags,
        prelude::{ApplicationExt, ApplicationExtManual},
    },
    glib::{self, OptionFlags, VariantTy},
    prelude::WidgetExt,
};

use crate::data::init_socket;
use crate::keybinds::setup_binds;
use crate::protos::QueryResponseObject;
use crate::protos::generated_proto::query::{QueryResponse, query_response};
use crate::providers::setup_providers;
use crate::state::{
    get_parameter_height, get_parameter_max_height, get_parameter_max_width,
    get_parameter_min_height, get_parameter_min_width, get_parameter_width, get_placeholder,
    get_provider, get_theme, has_elephant, has_theme, is_connected, is_dmenu, is_dmenu_keep_open,
    is_input_only, is_no_hints, is_no_search, is_param_close, is_service, is_visible,
    set_dmenu_current, set_dmenu_exit_after, set_dmenu_keep_open, set_has_elephant, set_hide_qa,
    set_initial_height, set_initial_max_height, set_initial_max_width, set_initial_min_height,
    set_initial_min_width, set_initial_placeholder, set_initial_width, set_input_only,
    set_is_dmenu, set_is_service, set_is_visible, set_no_hints, set_no_search, set_param_close,
    set_parameter_height, set_parameter_max_height, set_parameter_max_width,
    set_parameter_min_height, set_parameter_min_width, set_parameter_width, set_placeholder,
    set_provider, set_theme,
};
use crate::theme::{setup_css, setup_css_provider, setup_themes};
use crate::ui::window::{
    handle_preview, quit, set_input_text, set_keybind_hint, setup_window, with_window,
};

static GLOBAL_DMENU_SENDER: RwLock<Option<Sender<String>>> = RwLock::new(None);

thread_local! {
    static HOLD_GUARD: OnceCell<ApplicationHoldGuard> = OnceCell::new();
}

fn main() -> glib::ExitCode {
    let app = Application::builder()
        .application_id("dev.benz.walker")
        .flags(ApplicationFlags::HANDLES_COMMAND_LINE)
        .build();

    app.connect_handle_local_options(|_, _| return -1);

    add_flags(&app);

    app.connect_command_line(handle_command_line);
    app.connect_activate(activate);
    app.connect_startup(startup);

    app.run()
}

fn init_ui(app: &Application, dmenu: bool) {
    if app.flags().contains(ApplicationFlags::IS_SERVICE) {
        set_is_service(true);
    }

    config::load().unwrap();

    let mut theme = get_config().theme.as_str();

    if theme.is_empty() {
        theme = "default";
    }

    set_theme(theme.to_string());

    let mut elephant = false;

    if !dmenu || is_service() {
        elephant = which("elephant").is_ok();
        set_has_elephant(elephant);
    }

    setup_providers(elephant);

    setup_css_provider();

    setup_binds();

    setup_themes(elephant && !dmenu, get_theme(), is_service());

    setup_window(app);

    // start_theme_watcher(s.get_theme());
}

fn send_message(message: String) {
    let mut sender_guard = GLOBAL_DMENU_SENDER.write().unwrap();

    if let Some(sender) = sender_guard.take() {
        if let Err(_) = sender.send(message) {
            println!("the receiver dropped");
        }

        *sender_guard = None;
    } else {
        println!("No sender available");
    }
}

fn add_flags(app: &Application) {
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
        "nohints",
        b'N'.into(),
        OptionFlags::NONE,
        glib::OptionArg::None,
        "hide search input",
        None,
    );

    app.add_main_option(
        "inputonly",
        b'I'.into(),
        OptionFlags::NONE,
        glib::OptionArg::None,
        "only show input. dmenu only.",
        None,
    );

    app.add_main_option(
        "provider",
        b'm'.into(),
        OptionFlags::NONE,
        glib::OptionArg::String,
        "exclusive provider to query",
        None,
    );

    app.add_main_option(
        "placeholder",
        b'p'.into(),
        OptionFlags::NONE,
        glib::OptionArg::String,
        "input placeholder. dmenu only.",
        None,
    );

    app.add_main_option(
        "height",
        b'h'.into(),
        OptionFlags::NONE,
        glib::OptionArg::Int64,
        "forced height. concerned with overall box. min 1. DONT USE SHORTHAND.",
        None,
    );

    app.add_main_option(
        "minheight",
        b'h'.into(),
        OptionFlags::NONE,
        glib::OptionArg::Int64,
        "min height. concerned with scrolled content. min 1.",
        None,
    );

    app.add_main_option(
        "maxheight",
        b'h'.into(),
        OptionFlags::NONE,
        glib::OptionArg::Int64,
        "max height. concerned with scrolled content. min 1.",
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
        "forced width. concerned with overall box. min 1. DONT USE SHORTHAND.",
        None,
    );

    app.add_main_option(
        "minwidth",
        b'w'.into(),
        OptionFlags::NONE,
        glib::OptionArg::Int64,
        "min width. concerned with scrolled content. min 1.",
        None,
    );

    app.add_main_option(
        "maxwidth",
        b'w'.into(),
        OptionFlags::NONE,
        glib::OptionArg::Int64,
        "max width. concerned with scrolled content. min 1.",
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
        "hideqa",
        b'H'.into(),
        OptionFlags::NONE,
        glib::OptionArg::None,
        "hides quick activation buttons",
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
}

fn handle_command_line(app: &Application, cmd: &ApplicationCommandLine) -> i32 {
    let options = cmd.options_dict();

    if options.contains("version") {
        cmd.print_literal(&format!("{}\n", env!("CARGO_PKG_VERSION")));
        return 0;
    }

    if let Some(val) = options.lookup_value("provider", Some(VariantTy::STRING)) {
        set_provider(val.str().unwrap().to_string());
    }

    set_param_close(options.contains("close"));
    set_hide_qa(options.contains("hideqa"));

    if let Some(val) = options.lookup_value("theme", Some(VariantTy::STRING)) {
        let theme = val.str().unwrap();

        if has_theme(theme) {
            set_theme(theme.to_string());
        } else {
            cmd.print_literal("theme not found. using default theme.\n");
            set_theme("default".to_string());
        }
    }

    if let Some(val) = options.lookup_value("height", Some(VariantTy::INT64)) {
        set_parameter_height(Some(val.get::<i64>().unwrap() as i32));
    }

    if let Some(val) = options.lookup_value("width", Some(VariantTy::INT64)) {
        set_parameter_width(Some(val.get::<i64>().unwrap() as i32));
    }

    if let Some(val) = options.lookup_value("minwidth", Some(VariantTy::INT64)) {
        set_parameter_min_width(Some(val.get::<i64>().unwrap() as i32));
    }

    if let Some(val) = options.lookup_value("minheight", Some(VariantTy::INT64)) {
        set_parameter_min_height(Some(val.get::<i64>().unwrap() as i32));
    }

    if let Some(val) = options.lookup_value("maxwidth", Some(VariantTy::INT64)) {
        set_parameter_max_width(Some(val.get::<i64>().unwrap() as i32));
    }

    if let Some(val) = options.lookup_value("maxheight", Some(VariantTy::INT64)) {
        set_parameter_max_height(Some(val.get::<i64>().unwrap() as i32));
    }

    set_no_search(options.contains("nosearch"));
    set_no_hints(options.contains("nohints"));

    'dmenu: {
        if !options.contains("dmenu") {
            set_dmenu_keep_open(false);
            set_is_dmenu(false);
            break 'dmenu;
        } else {
            set_is_dmenu(true);
        }

        if let Some(val) = options.lookup_value("placeholder", Some(VariantTy::STRING)) {
            set_placeholder(val.str().unwrap().to_string());
        }

        set_input_only(options.contains("inputonly"));

        if options.contains("keepopen") && app.flags().contains(ApplicationFlags::IS_SERVICE) {
            set_dmenu_keep_open(true);
        }

        if let Some(val) = options.lookup_value("current", Some(VariantTy::INT64)) {
            set_dmenu_current(val.get::<i64>().unwrap());
        }

        set_dmenu_exit_after(options.contains("exit"));

        if GLOBAL_DMENU_SENDER.read().unwrap().is_some() {
            send_message("CNCLD".to_string());
            break 'dmenu;
        }

        with_window(|w| {
            set_input_text("");

            let items = w.items.clone();
            items.remove_all();

            if is_input_only() {
                return;
            }

            let stdin = cmd.stdin();
            let data_stream = gio::DataInputStream::new(&stdin.unwrap());

            async fn read_lines_async(stream: Rc<gio::DataInputStream>, items: gio::ListStore) {
                let mut i = 0;

                loop {
                    match stream.read_line_utf8_future(Priority::DEFAULT).await {
                        Ok(Some(line)) => {
                            if line.is_empty() {
                                break;
                            }

                            let line = line.trim();

                            if !line.is_empty() {
                                let mut item = query_response::Item::new();
                                item.text = line.to_string();
                                item.provider = "dmenu".to_string();
                                item.score = 1000000 - i;
                                item.actions = vec!["select".to_string()];

                                let mut response = QueryResponse::new();
                                response.item = protobuf::MessageField::some(item);

                                items.append(&QueryResponseObject::new(response));
                            }

                            i += 1;
                        }
                        Ok(None) => {
                            set_keybind_hint();
                            break;
                        }
                        Err(e) => {
                            eprintln!("Error reading: {e}");
                            break;
                        }
                    }
                }
            }

            glib::spawn_future_local(async move {
                read_lines_async(Rc::new(data_stream), items).await;
            });
        });

        if !is_service() {
            break 'dmenu;
        }

        let (sender, receiver) = oneshot::channel();

        *GLOBAL_DMENU_SENDER.write().unwrap() = Some(sender);

        let cmd = cmd.clone();

        glib::spawn_future_local(async move {
            match receiver.await {
                Ok(message) => match message.as_str() {
                    "CNCLD" => {
                        cmd.set_exit_status(130);
                    }
                    msg => cmd.print_literal(&format!("{msg}\n")),
                },
                Err(_) => {
                    println!("the sender dropped");
                    cmd.set_exit_status(130);
                }
            }

            *GLOBAL_DMENU_SENDER.write().unwrap() = None;
        });
    }

    app.activate();
    0
}

fn activate(app: &Application) {
    let cfg = get_config();

    if is_dmenu() && is_visible() {
        return;
    }

    if (cfg.close_when_open && is_visible() && !is_dmenu_keep_open()) || is_param_close() {
        quit(app, false);
        return;
    }

    if is_dmenu_keep_open() && is_visible() {
        return;
    }

    let provider = get_provider();
    let provider = if provider.is_empty() {
        "default"
    } else {
        provider.as_str()
    };

    with_window(|w| {
        if is_input_only() {
            w.content_container.set_visible(false);
            if let Some(keybinds) = &w.keybinds {
                keybinds.set_visible(false);
            }
        }

        if let Some(placeholders) = &cfg.placeholders
            && let Some(placeholder) = placeholders.get(provider)
        {
            if let Some(input) = &w.input {
                input.set_placeholder_text(Some(&placeholder.input));
            }

            if let Some(p) = w.placeholder.as_ref() {
                p.set_text(&placeholder.list)
            }
        }

        if !get_placeholder().is_empty()
            && let Some(input) = &w.input
        {
            if let Some(placeholder) = input.placeholder_text() {
                set_initial_placeholder(placeholder.to_string());
            }

            input.set_placeholder_text(Some(&get_placeholder()));
        }

        if let Some(val) = get_parameter_height() {
            set_initial_height(Some(w.box_wrapper.height_request()));
            w.box_wrapper.set_height_request(val);
        }

        if let Some(val) = get_parameter_width() {
            set_initial_width(Some(w.box_wrapper.width_request()));
            w.box_wrapper.set_width_request(val);
        }

        if let Some(val) = get_parameter_min_width() {
            set_initial_min_width(Some(w.scroll.min_content_width()));
            w.scroll.set_min_content_width(val);
        }

        if let Some(val) = get_parameter_min_height() {
            set_initial_min_height(Some(w.scroll.min_content_height()));
            w.scroll.set_min_content_height(val);
        }

        if let Some(val) = get_parameter_max_width() {
            set_initial_max_width(Some(w.scroll.max_content_width()));
            w.scroll.set_max_content_width(val);
        }

        if let Some(val) = get_parameter_max_height() {
            set_initial_max_height(Some(w.scroll.max_content_height()));
            w.scroll.set_max_content_height(val);
        }

        if get_parameter_min_width().is_some() || get_parameter_max_width().is_some() {
            set_initial_width(Some(w.box_wrapper.width_request()));
            w.box_wrapper.set_width_request(-1);
        }

        if get_parameter_min_height().is_some() || get_parameter_max_height().is_some() {
            set_initial_height(Some(w.box_wrapper.height_request()));
            w.box_wrapper.set_height_request(-1);
        }

        if is_no_search()
            && let Some(search_container) = &w.search_container
        {
            search_container.set_visible(false);
        }

        if is_no_hints()
            && let Some(hints) = &w.keybinds
        {
            hints.set_visible(false);
        }

        setup_css(get_theme());

        if let Some(input) = &w.input {
            if is_service() && provider != "default" {
                input.emit_by_name::<()>("changed", &[]);
            }

            input.grab_focus();
        }

        if !is_connected() && !is_dmenu() {
            w.elephant_hint.set_visible(true);
            w.scroll.set_visible(false);
        } else {
            w.elephant_hint.set_visible(false);
        }

        w.window.set_visible(true);

        if !is_dmenu() && !is_connected() && has_elephant() {
            thread::spawn(|| init_socket().unwrap());
        } else if !has_elephant() && !is_dmenu() {
            println!("Please install elephant.");
            process::exit(1);
        }
    });

    set_is_visible(true);
}

fn startup(app: &Application) {
    let args: Vec<String> = env::args().collect();
    let dmenu = args.contains(&"--dmenu".to_string()) || args.contains(&"-d".to_string());
    let version = args.contains(&"--version".to_string()) || args.contains(&"-v".to_string());
    let is_service = app.flags().contains(ApplicationFlags::IS_SERVICE);

    if !is_service && (args.contains(&"--close".to_string()) || args.contains(&"-q".to_string())) {
        process::exit(0);
    }

    if version {
        return;
    }

    if !app.flags().contains(ApplicationFlags::IS_SERVICE) && !dmenu {
        println!("make sure 'walker --gapplication-service' is running!");
    }

    HOLD_GUARD.with(|h| h.set(app.hold()).expect("couldn't set hold-guard"));

    init_app_state();
    init_ui(app, dmenu);

    listen_activation_socket(app.clone());
}

fn listen_activation_socket(app_clone: Application) {
    let mut socket_path = env::var("XDG_RUNTIME_DIR")
        .map(PathBuf::from)
        .unwrap_or_else(|_| env::temp_dir());

    socket_path.push("walker");
    if !socket_path.exists() {
        fs::create_dir(&socket_path).unwrap();
    }

    socket_path.push("walker.sock");

    let _ = fs::remove_file(&socket_path);

    let listener = UnixListener::bind(&socket_path).unwrap();
    listener.set_nonblocking(true).unwrap();

    let fd = listener.as_raw_fd();

    glib::unix_fd_add_local(fd, glib::IOCondition::IN, move |_fd, condition| {
        if condition.contains(glib::IOCondition::IN) {
            match listener.accept() {
                Ok((stream, _)) => {
                    drop(stream);
                    activate(&app_clone);
                }
                Err(e) if e.kind() == std::io::ErrorKind::WouldBlock => {}
                Err(e) => {
                    eprintln!("Error accepting connection: {}", e);
                    return glib::ControlFlow::Break;
                }
            }
        }
        glib::ControlFlow::Continue
    });
}
