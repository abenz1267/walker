use crate::providers::Provider;

#[derive(Debug)]
pub struct DefaultProvider {
    name: String,
}

impl DefaultProvider {
    pub fn new(name: String) -> Self {
        Self { name }
    }
}

impl Provider for DefaultProvider {
    fn get_name(&self) -> &str {
        self.name.as_str()
    }
}
