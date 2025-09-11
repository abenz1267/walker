use gtk4::glib::{self, subclass::types::ObjectSubclassIsExt};

pub mod generated_proto {
    include!(concat!(env!("OUT_DIR"), "/generated_proto/mod.rs"));
}

// GObject wrapper for QueryResponse
mod imp {
    use crate::protos::generated_proto::query::QueryResponse;

    use super::*;
    use gtk4::subclass::prelude::*;
    use std::cell::RefCell;

    #[derive(Debug, Default)]
    pub struct QueryResponseObject {
        pub response: RefCell<Option<QueryResponse>>,
        pub dmenu_score: RefCell<u32>,
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

    pub fn dmenu_score(&self) -> u32 {
        *self.imp().dmenu_score.borrow()
    }

    pub fn set_dmenu_score(&self, val: u32) {
        *self.imp().dmenu_score.borrow_mut() = val;
    }
}
