use crate::state::{AppState, Settings, WebhookPayload};
use std::fs;
use tauri::{AppHandle, Emitter, Manager, State, WebviewWindow};
use tauri_plugin_autostart::ManagerExt;

#[tauri::command]
pub fn get_settings(state: State<AppState>) -> Result<Settings, String> {
    Ok(Settings::load(&state.settings))
}

#[tauri::command]
pub async fn save_settings(settings: Settings, app: AppHandle) -> Result<(), String> {
    let state: State<AppState> = app.state();

    // Handle side effects
    let autostart_manager = app.autolaunch();
    if settings.launch_at_login {
        autostart_manager.enable().map_err(|e| e.to_string())?;
    } else {
        autostart_manager.disable().map_err(|e| e.to_string())?;
    }

    // Handle lockfile location change
    {
        let mut lockfile_guard = state.lockfile_path.lock().unwrap();
        let new_config_dir = if let Some(dir) = &settings.lockfile_dir {
            std::path::PathBuf::from(dir)
        } else {
            app.path().app_config_dir().unwrap()
        };
        let new_path = new_config_dir.join("daylit-tray.lock");

        if let Some(old_path) = lockfile_guard.as_ref() {
            if *old_path != new_path {
                if old_path.exists() {
                    let content = fs::read_to_string(old_path).map_err(|e| e.to_string())?;
                    fs::remove_file(old_path).map_err(|e| e.to_string())?;

                    fs::create_dir_all(&new_config_dir).map_err(|e| e.to_string())?;
                    fs::write(&new_path, content).map_err(|e| e.to_string())?;
                }
                *lockfile_guard = Some(new_path);
            }
        }
    }

    // Save to store
    state.settings.set(
        "settings",
        serde_json::to_value(&settings).map_err(|e| e.to_string())?,
    );
    state.settings.save().map_err(|e| e.to_string())?;

    app.emit("settings-updated", &settings)
        .map_err(|e| e.to_string())
}

#[tauri::command]
pub fn get_notification_payload(state: State<AppState>) -> Option<WebhookPayload> {
    state.payload.lock().unwrap().clone()
}

#[tauri::command]
pub fn close_notification_window(window: WebviewWindow) {
    if window.label() == "notification_dialog" {
        window.close().unwrap();
    }
}
