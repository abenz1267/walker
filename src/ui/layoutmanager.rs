use gtk4::glib;
use gtk4::glib::Properties;
use gtk4::glib::subclass::prelude::*;
use gtk4::prelude::*;
use gtk4::subclass::prelude::*;
use gtk4::{LayoutManager, Orientation, Widget};
use std::cell::Cell;

mod imp {
    use super::*;

    #[derive(Properties, Default)]
    #[properties(wrapper_type = super::PercentageLayoutManager)]
    pub struct PercentageLayoutManager {
        #[property(get, set = Self::set_percentage, minimum = 0.0, maximum = 1.0, default = 0.0)]
        percentage: Cell<f64>,
    }

    impl PercentageLayoutManager {
        fn set_percentage(&self, value: f64) {
            self.percentage.set(value);
            self.obj().layout_changed();
        }
    }

    #[glib::derived_properties]
    impl ObjectImpl for PercentageLayoutManager {}

    #[glib::object_subclass]
    impl ObjectSubclass for PercentageLayoutManager {
        const NAME: &'static str = "PercentageLayoutManager";
        type Type = super::PercentageLayoutManager;
        type ParentType = LayoutManager;
    }

    impl LayoutManagerImpl for PercentageLayoutManager {
        fn measure(
            &self,
            widget: &Widget,
            orientation: Orientation,
            for_size: i32,
        ) -> (i32, i32, i32, i32) {
            let mut minimum = 0;
            let mut natural = 0;
            let mut minimum_baseline = -1;
            let mut natural_baseline = -1;

            if let Some(child) = widget.first_child() {
                let (child_min, child_nat, child_min_baseline, child_nat_baseline) =
                    child.measure(orientation, for_size);

                minimum = minimum.max(child_min);
                natural = natural.max(child_nat);

                if child_min_baseline > -1 {
                    minimum_baseline = minimum_baseline.max(child_min_baseline);
                }
                if child_nat_baseline > -1 {
                    natural_baseline = natural_baseline.max(child_nat_baseline);
                }
            }

            (minimum, natural, minimum_baseline, natural_baseline)
        }

        fn allocate(&self, widget: &Widget, width: i32, height: i32, _baseline: i32) {
            let child_width = (width as f64 * self.percentage.get()).round() as i32;

            if let Some(child) = widget.first_child() {
                child.allocate(child_width, height, -1, None);
            }
        }
    }
}

glib::wrapper! {
    pub struct PercentageLayoutManager(ObjectSubclass<imp::PercentageLayoutManager>)
        @extends LayoutManager;
}

impl PercentageLayoutManager {
    pub fn new(percentage: f64) -> Self {
        let this: Self = glib::Object::new();
        this.set_percentage(percentage);
        this
    }
}
