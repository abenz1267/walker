use crate::providers::Provider;
use crate::providers::Item;
use crate::ui::layoutmanager::PercentageLayoutManager;

#[derive(Debug)]
pub struct Wireplumber {
    name: &'static str,
}

impl Wireplumber {
    pub fn new() -> Self {
        Self {
            name: "wireplumber",
        }
    }
}

impl Provider for Wireplumber {
    fn get_name(&self) -> &str {
        self.name
    }

    fn get_item_layout(&self) -> String {
        include_str!("../../resources/themes/default/item_progress.xml").to_string()
    }

    fn progress_transformer(&self, item: &Item, layout: PercentageLayoutManager) {
        for (i, _) in item.subtext.match_indices('%') {
            let start = item.subtext[..i]
                .rfind(|c: char| !c.is_ascii_digit() && c != '.')
                .map_or(0, |i| i + 1);

            if let Ok(value) = item.subtext[start..i].parse::<f64>() {
                layout.set_percentage((value / 100.0).clamp(0.0, 1.0));
                return
            }
        }
    }
}
