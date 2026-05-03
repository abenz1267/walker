use std::cell::RefCell;
use std::rc::Rc;

use gdk4_wayland::prelude::WaylandSurfaceExtManual;
use gdk4_wayland::{WaylandDisplay, WaylandSurface};
use gtk4::prelude::*;
use gtk4::{Box as GtkBox, Window};
use wayland_client::protocol::wl_compositor::WlCompositor;
use wayland_client::protocol::wl_region::WlRegion;
use wayland_client::protocol::wl_registry::WlRegistry;
use wayland_client::{
    Connection, Dispatch, EventQueue, Proxy, QueueHandle, delegate_noop,
    globals::{GlobalListContents, registry_queue_init},
};

mod ext_background_effect_v1 {
    use wayland_client;
    use wayland_client::protocol::*;

    pub mod __interfaces {
        use wayland_client::protocol::__interfaces::*;
        wayland_scanner::generate_interfaces!("src/protos/ext-background-effect-v1.xml");
    }
    use self::__interfaces::*;

    wayland_scanner::generate_client_code!("src/protos/ext-background-effect-v1.xml");
}

use ext_background_effect_v1::ext_background_effect_manager_v1::ExtBackgroundEffectManagerV1;
use ext_background_effect_v1::ext_background_effect_surface_v1::ExtBackgroundEffectSurfaceV1;

struct AppState;

impl Dispatch<WlRegistry, GlobalListContents> for AppState {
    fn event(
        _: &mut Self,
        _: &WlRegistry,
        _: <WlRegistry as Proxy>::Event,
        _: &GlobalListContents,
        _: &Connection,
        _: &QueueHandle<Self>,
    ) {
    }
}

impl Dispatch<ExtBackgroundEffectManagerV1, ()> for AppState {
    fn event(
        _: &mut Self,
        _: &ExtBackgroundEffectManagerV1,
        _: <ExtBackgroundEffectManagerV1 as Proxy>::Event,
        _: &(),
        _: &Connection,
        _: &QueueHandle<Self>,
    ) {
    }
}

delegate_noop!(AppState: ignore WlCompositor);
delegate_noop!(AppState: ignore WlRegion);
delegate_noop!(AppState: ignore ExtBackgroundEffectSurfaceV1);

struct BlurContext {
    queue: EventQueue<AppState>,
    qh: QueueHandle<AppState>,
    compositor: WlCompositor,
    bg_effect: ExtBackgroundEffectSurfaceV1,
    last_rect: Option<(i32, i32, i32, i32)>,
}

impl BlurContext {
    fn update_region(&mut self, rect: (i32, i32, i32, i32)) {
        if self.last_rect == Some(rect) {
            return;
        }
        self.last_rect = Some(rect);

        let region = self.compositor.create_region(&self.qh, ());
        let (x, y, w, h) = rect;
        region.add(x, y, w.max(0), h.max(0));
        self.bg_effect.set_blur_region(Some(&region));
        region.destroy();

        let _ = self.queue.flush();
    }
}

impl Drop for BlurContext {
    fn drop(&mut self) {
        self.bg_effect.destroy();
        let _ = self.queue.flush();
    }
}

pub fn attach_blur(window: &Window, target: &GtkBox) {
    let ctx: Rc<RefCell<Option<BlurContext>>> = Rc::new(RefCell::new(None));

    {
        let ctx = ctx.clone();
        let target = target.clone();
        window.connect_realize(move |window| {
            let Ok(c) = init_context(window) else {
                eprintln!("background-effect-v1 unavailable");
                return;
            };
            *ctx.borrow_mut() = Some(c);

            if let Some(fc) = window.frame_clock() {
                let ctx = ctx.clone();
                let target = target.clone();
                let window = window.clone();
                fc.connect_layout(move |_| {
                    if let Some(c) = ctx.borrow_mut().as_mut()
                        && let Some(bounds) = target.compute_bounds(&window)
                    {
                        c.update_region((
                            bounds.x() as i32,
                            bounds.y() as i32,
                            bounds.width() as i32,
                            bounds.height() as i32,
                        ));
                    }
                });
            }
        });
    }

    {
        let ctx = ctx.clone();
        window.connect_unrealize(move |_| {
            ctx.borrow_mut().take();
        });
    }
}

fn init_context(window: &Window) -> Result<BlurContext, String> {
    let display = WidgetExt::display(window);
    let wd = display
        .downcast_ref::<WaylandDisplay>()
        .ok_or_else(|| "not running on wayland".to_string())?;

    let wl_disp = wd
        .wl_display()
        .ok_or_else(|| "wl_display unavailable".to_string())?;
    let compositor = wd
        .wl_compositor()
        .ok_or_else(|| "wl_compositor unavailable".to_string())?;

    let gdk_surface = window
        .surface()
        .ok_or_else(|| "no gdk surface for window".to_string())?;
    let ws = gdk_surface
        .downcast_ref::<WaylandSurface>()
        .ok_or_else(|| "gdk surface is not wayland".to_string())?;
    let surface = ws
        .wl_surface()
        .ok_or_else(|| "wl_surface unavailable".to_string())?;

    let backend = wl_disp
        .backend()
        .upgrade()
        .ok_or_else(|| "wayland backend unavailable".to_string())?;
    let connection = Connection::from_backend(backend);

    let (globals, mut queue) = registry_queue_init::<AppState>(&connection)
        .map_err(|e| format!("registry init failed: {e}"))?;
    let qh = queue.handle();

    let manager: ExtBackgroundEffectManagerV1 = globals
        .bind(&qh, 1..=1, ())
        .map_err(|e| format!("compositor lacks ext_background_effect_v1: {e}"))?;

    let mut state = AppState;
    let _ = queue.roundtrip(&mut state);

    let bg_effect = manager.get_background_effect(&surface, &qh, ());
    manager.destroy();

    let _ = connection.flush();

    Ok(BlurContext {
        queue,
        qh,
        compositor,
        bg_effect,
        last_rect: None,
    })
}
