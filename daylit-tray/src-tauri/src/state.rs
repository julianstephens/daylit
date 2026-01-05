use serde::{Deserialize, Serialize};
use std::sync::{Arc, Mutex};
use tauri::Wry;
use tauri_plugin_store::Store;

pub const LOCKFILE_NAME: &str = "daylit-tray.lock";

#[derive(Serialize, Deserialize, Debug, Clone)]
#[serde(default)]
pub struct Settings {
    pub font_size: String,
    pub launch_at_login: bool,
    pub lockfile_dir: Option<String>,
    pub daylit_path: Option<String>,
    pub use_native_notifications: bool,
}

impl Default for Settings {
    fn default() -> Self {
        Self {
            font_size: "medium".into(),
            launch_at_login: false,
            lockfile_dir: None,
            daylit_path: None,
            use_native_notifications: false,
        }
    }
}

impl Settings {
    pub fn load(store: &Store<Wry>) -> Self {
        store
            .get("settings")
            .and_then(|v| serde_json::from_value(v).ok())
            .unwrap_or_default()
    }
}

#[derive(Clone, Serialize, Deserialize, Debug, Default)]
pub struct WebhookPayload {
    pub text: String,
    pub duration_ms: u32,
}

// Event payload for when we re-use an existing window
#[derive(Clone, serde::Serialize)]
pub struct UpdatePayload {
    pub text: String,
    pub duration_ms: u32,
}

// Main application state, holds settings store and last payload
pub struct AppState {
    pub settings: Arc<Store<Wry>>,
    pub payload: Mutex<Option<WebhookPayload>>,
    pub lockfile_path: Mutex<Option<std::path::PathBuf>>,
    pub secret: Mutex<Option<String>>,
}
