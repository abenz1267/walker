use std::collections::HashSet;
use std::sync::{OnceLock, RwLock};

use crate::keybinds::AfterAction;

static STATE: OnceLock<RwLock<AppState>> = OnceLock::new();

#[derive(Debug, Clone, Default)]
pub struct AppState {
    async_after: Option<AfterAction>,
    hide_qa: bool,
    has_elephant: bool,
    is_connected: bool,
    is_connecting: bool,
    dmenu_keep_open: bool,
    dmenu_exit_after: bool,
    dmenu_current: i64,
    initial_height: Option<i32>,
    initial_width: Option<i32>,
    initial_max_height: Option<i32>,
    initial_max_width: Option<i32>,
    initial_min_height: Option<i32>,
    initial_min_width: Option<i32>,
    parameter_width: Option<i32>,
    parameter_height: Option<i32>,
    parameter_min_height: Option<i32>,
    parameter_min_width: Option<i32>,
    parameter_max_height: Option<i32>,
    parameter_max_width: Option<i32>,
    last_query: String,
    placeholder: String,
    initial_placeholder: String,
    available_themes: HashSet<String>,
    provider: String,
    theme: String,
    is_service: bool,
    no_search: bool,
    input_only: bool,
    is_dmenu: bool,
    is_param_close: bool,
    current_prefix: String,
    is_visible: bool,
    query: String,
}

pub fn init_app_state() {
    STATE
        .set(RwLock::new(AppState::default()))
        .expect("can't init appstate");
}

pub fn get_theme() -> String {
    STATE.get().unwrap().read().unwrap().theme.clone()
}

pub fn set_theme(val: String) {
    STATE.get().unwrap().write().unwrap().theme = val
}

pub fn get_async_after() -> Option<AfterAction> {
    STATE.get().unwrap().read().unwrap().async_after.clone()
}

pub fn set_async_after(val: Option<AfterAction>) {
    STATE.get().unwrap().write().unwrap().async_after = val
}

pub fn get_current_prefix() -> String {
    STATE.get().unwrap().read().unwrap().current_prefix.clone()
}

pub fn set_current_prefix(val: String) {
    STATE.get().unwrap().write().unwrap().current_prefix = val
}

pub fn get_provider() -> String {
    STATE.get().unwrap().read().unwrap().provider.clone()
}

pub fn set_provider(val: String) {
    STATE.get().unwrap().write().unwrap().provider = val
}

pub fn get_initial_placeholder() -> String {
    STATE
        .get()
        .unwrap()
        .read()
        .unwrap()
        .initial_placeholder
        .clone()
}

pub fn set_initial_placeholder(val: String) {
    STATE.get().unwrap().write().unwrap().initial_placeholder = val
}

pub fn get_placeholder() -> String {
    STATE.get().unwrap().read().unwrap().placeholder.clone()
}

pub fn set_placeholder(val: String) {
    STATE.get().unwrap().write().unwrap().placeholder = val
}

pub fn get_last_query() -> String {
    STATE.get().unwrap().read().unwrap().last_query.clone()
}

pub fn set_last_query(val: String) {
    STATE.get().unwrap().write().unwrap().last_query = val
}

pub fn set_is_service(val: bool) {
    STATE.get().unwrap().write().unwrap().is_service = val
}

pub fn is_visible() -> bool {
    STATE.get().unwrap().read().unwrap().is_visible
}

pub fn set_is_visible(val: bool) {
    STATE.get().unwrap().write().unwrap().is_visible = val;
}

pub fn has_elephant() -> bool {
    STATE.get().unwrap().read().unwrap().has_elephant
}

pub fn set_has_elephant(val: bool) {
    STATE.get().unwrap().write().unwrap().has_elephant = val
}

pub fn is_connected() -> bool {
    STATE.get().unwrap().read().unwrap().is_connected
}

pub fn set_is_connected(val: bool) {
    STATE.get().unwrap().write().unwrap().is_connected = val
}

pub fn is_connecting() -> bool {
    STATE.get().unwrap().read().unwrap().is_connecting
}

pub fn set_is_connecting(val: bool) {
    STATE.get().unwrap().write().unwrap().is_connecting = val
}

pub fn is_input_only() -> bool {
    STATE.get().unwrap().read().unwrap().input_only
}

pub fn set_input_only(val: bool) {
    STATE.get().unwrap().write().unwrap().input_only = val
}

pub fn is_param_close() -> bool {
    STATE.get().unwrap().read().unwrap().is_param_close
}

pub fn set_param_close(val: bool) {
    STATE.get().unwrap().write().unwrap().is_param_close = val
}

pub fn is_dmenu_keep_open() -> bool {
    STATE.get().unwrap().read().unwrap().dmenu_keep_open
}

pub fn set_dmenu_keep_open(val: bool) {
    STATE.get().unwrap().write().unwrap().dmenu_keep_open = val
}

pub fn is_dmenu_exit_after() -> bool {
    STATE.get().unwrap().read().unwrap().dmenu_exit_after
}

pub fn set_dmenu_exit_after(val: bool) {
    STATE.get().unwrap().write().unwrap().dmenu_exit_after = val
}

pub fn is_dmenu() -> bool {
    STATE.get().unwrap().read().unwrap().is_dmenu
}

pub fn set_is_dmenu(val: bool) {
    STATE.get().unwrap().write().unwrap().is_dmenu = val
}

pub fn is_hide_qa() -> bool {
    STATE.get().unwrap().read().unwrap().hide_qa
}

pub fn set_hide_qa(val: bool) {
    STATE.get().unwrap().write().unwrap().hide_qa = val
}

pub fn query() -> String {
    STATE.get().unwrap().read().unwrap().query.clone()
}

pub fn set_query(val: &str) {
    STATE.get().unwrap().write().unwrap().query = val.to_string()
}

pub fn is_no_search() -> bool {
    STATE.get().unwrap().read().unwrap().no_search
}

pub fn is_service() -> bool {
    STATE.get().unwrap().read().unwrap().is_service
}

pub fn set_no_search(val: bool) {
    STATE.get().unwrap().write().unwrap().no_search = val
}

pub fn set_initial_height(val: Option<i32>) {
    STATE.get().unwrap().write().unwrap().initial_height = val
}

pub fn set_initial_width(val: Option<i32>) {
    STATE.get().unwrap().write().unwrap().initial_width = val
}

pub fn set_initial_max_height(val: Option<i32>) {
    STATE.get().unwrap().write().unwrap().initial_max_height = val
}

pub fn set_initial_max_width(val: Option<i32>) {
    STATE.get().unwrap().write().unwrap().initial_max_width = val
}

pub fn set_initial_min_height(val: Option<i32>) {
    STATE.get().unwrap().write().unwrap().initial_min_height = val
}

pub fn set_initial_min_width(val: Option<i32>) {
    STATE.get().unwrap().write().unwrap().initial_min_width = val
}

pub fn get_initial_height() -> Option<i32> {
    STATE.get().unwrap().read().unwrap().initial_height
}

pub fn get_initial_width() -> Option<i32> {
    STATE.get().unwrap().read().unwrap().initial_width
}

pub fn get_initial_min_height() -> Option<i32> {
    STATE.get().unwrap().read().unwrap().initial_min_height
}

pub fn get_initial_min_width() -> Option<i32> {
    STATE.get().unwrap().read().unwrap().initial_min_width
}

pub fn get_initial_max_height() -> Option<i32> {
    STATE.get().unwrap().read().unwrap().initial_max_height
}

pub fn get_initial_max_width() -> Option<i32> {
    STATE.get().unwrap().read().unwrap().initial_max_width
}

pub fn set_parameter_height(val: Option<i32>) {
    STATE.get().unwrap().write().unwrap().parameter_height = val
}

pub fn set_parameter_width(val: Option<i32>) {
    STATE.get().unwrap().write().unwrap().parameter_width = val
}

pub fn set_parameter_min_height(val: Option<i32>) {
    STATE.get().unwrap().write().unwrap().parameter_min_height = val
}

pub fn set_parameter_min_width(val: Option<i32>) {
    STATE.get().unwrap().write().unwrap().parameter_min_width = val
}

pub fn set_parameter_max_height(val: Option<i32>) {
    STATE.get().unwrap().write().unwrap().parameter_max_height = val
}

pub fn set_parameter_max_width(val: Option<i32>) {
    STATE.get().unwrap().write().unwrap().parameter_max_width = val
}

pub fn get_parameter_height() -> Option<i32> {
    STATE.get().unwrap().read().unwrap().parameter_height
}

pub fn get_parameter_width() -> Option<i32> {
    STATE.get().unwrap().read().unwrap().parameter_width
}

pub fn get_parameter_min_height() -> Option<i32> {
    STATE.get().unwrap().read().unwrap().parameter_min_height
}

pub fn get_parameter_min_width() -> Option<i32> {
    STATE.get().unwrap().read().unwrap().parameter_min_width
}

pub fn get_parameter_max_height() -> Option<i32> {
    STATE.get().unwrap().read().unwrap().parameter_max_height
}

pub fn get_parameter_max_width() -> Option<i32> {
    STATE.get().unwrap().read().unwrap().parameter_max_width
}

pub fn get_dmenu_current() -> i64 {
    STATE.get().unwrap().read().unwrap().dmenu_current
}

pub fn set_dmenu_current(val: i64) {
    STATE.get().unwrap().write().unwrap().dmenu_current = val
}

pub fn add_theme(val: String) {
    STATE
        .get()
        .unwrap()
        .write()
        .unwrap()
        .available_themes
        .insert(val);
}

pub fn has_theme(val: &str) -> bool {
    STATE
        .get()
        .unwrap()
        .read()
        .unwrap()
        .available_themes
        .contains(val)
}
