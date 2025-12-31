use serde::{Deserialize, Serialize};
use std::thread;
use std::{fs, sync::Mutex};
use tauri::Emitter;
use tauri::Listener;
use tauri::{
    image::Image,
    menu::{Menu, MenuItem},
    tray::TrayIconBuilder,
    AppHandle, Manager, State, WebviewWindow, Wry,
};
use std::sync::Arc;
use tauri_plugin_store::{Store, StoreExt};
use tauri_plugin_autostart::{MacosLauncher, ManagerExt};
use tauri_plugin_log::log::info;
use tiny_http::{Response, Server};

// --- Struct Definitions for State and Payloads ---

#[derive(Serialize, Deserialize, Debug, Clone)]
pub struct Settings {
    font_size: String,
    launch_at_login: bool,
    daylit_dir: Option<String>,
}

impl Default for Settings {
    fn default() -> Self {
        Self {
            font_size: "medium".into(),
            launch_at_login: false,
            daylit_dir: None,
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
    text: String,
    duration_ms: u32,
}

// Event payload for when we re-use an existing window
#[derive(Clone, serde::Serialize)]
struct UpdatePayload {
    text: String,
    duration_ms: u32,
}

// Main application state, holds settings store and last payload
pub struct AppState {
    pub settings: Arc<Store<Wry>>,
    pub payload: Mutex<Option<WebhookPayload>>,
    pub lockfile_path: Mutex<Option<std::path::PathBuf>>,
}

// --- Tauri Commands ---

#[tauri::command]
fn get_settings(state: State<AppState>) -> Result<Settings, String> {
    Ok(Settings::load(&state.settings))
}

#[tauri::command]
fn set_daylit_dir(daylit_dir: String, state: State<AppState>) -> Result<(), String> {
    let mut settings = Settings::load(&state.settings);
    settings.daylit_dir = Some(daylit_dir);

    state.settings.set(
        "settings",
        serde_json::to_value(settings).map_err(|e| e.to_string())?,
    );
    state.settings.save().map_err(|e| e.to_string())
}

#[tauri::command]
fn set_font_size(font_size: String, state: State<AppState>) -> Result<(), String> {
    let mut settings = Settings::load(&state.settings);
    settings.font_size = font_size;

    state.settings.set(
        "settings",
        serde_json::to_value(settings).map_err(|e| e.to_string())?,
    );
    state.settings.save().map_err(|e| e.to_string())
}

#[tauri::command]
async fn set_launch_at_login(enable: bool, app: AppHandle) -> Result<(), String> {
    let autostart_manager = app.autolaunch();
    if enable {
        autostart_manager.enable().map_err(|e| e.to_string())?;
    } else {
        autostart_manager.disable().map_err(|e| e.to_string())?;
    }

    let state: State<AppState> = app.state();
    let mut settings = Settings::load(&state.settings);

    settings.launch_at_login = enable;

    state.settings.set(
        "settings",
        serde_json::to_value(settings).map_err(|e| e.to_string())?,
    );
    state.settings.save().map_err(|e| e.to_string())
}

#[tauri::command]
fn get_notification_payload(state: State<AppState>) -> Option<WebhookPayload> {
    state.payload.lock().unwrap().clone()
}

#[tauri::command]
fn close_notification_window(window: WebviewWindow) {
    if window.label() == "notification_dialog" {
        window.close().unwrap();
    }
}

// --- Core Application Logic ---

fn start_webhook_server(app_handle: AppHandle) {
    thread::spawn(move || {
        // Bind to port 0 to let the OS choose an available port
        let server = Server::http("127.0.0.1:0").unwrap();
        let port = server.server_addr().to_ip().unwrap().port();

        // --- Create Lock File ---
        let state: State<AppState> = app_handle.state();
        let settings = Settings::load(&state.settings);

        let config_dir = if let Some(dir) = settings.daylit_dir {
            std::path::PathBuf::from(dir)
        } else {
            app_handle.path().app_config_dir().unwrap()
        };

        fs::create_dir_all(&config_dir).unwrap();
        let lock_file_path = config_dir.join("daylit.lock");
        let pid = std::process::id();
        let lock_content = format!("{}|{}", port, pid);
        fs::write(&lock_file_path, lock_content).expect("Failed to write lock file");
        
        // Store the path so we can delete it later
        *state.lockfile_path.lock().unwrap() = Some(lock_file_path);

        info!("Webhook server started on port: {}", port);

        for mut request in server.incoming_requests() {
            if request.method().as_str() != "POST" {
                continue;
            }

            let mut content = String::new();
            request.as_reader().read_to_string(&mut content).unwrap();

            if let Ok(payload) = serde_json::from_str::<WebhookPayload>(&content) {
                let state: State<AppState> = app_handle.state();
                *state.payload.lock().unwrap() = Some(payload.clone());

                let app_handle_clone = app_handle.clone();
                app_handle.run_on_main_thread(move || {
                    // --- Re-use or Create Window Logic ---
                    if let Some(existing_window) =
                        app_handle_clone.get_webview_window("notification_dialog")
                    {
                        info!("Dialog exists. Re-using and sending new data.");
                        existing_window.set_focus().unwrap();
                        existing_window.emit(
                                "update_notification",
                                &UpdatePayload {
                                    text: payload.text,
                                    duration_ms: payload.duration_ms,
                                },
                            )
                            .unwrap();
                    } else {
                        info!("Dialog does not exist. Creating a new one.");
                        if let Ok(Some(monitor)) = app_handle_clone
                            .get_webview_window("main")
                            .unwrap()
                            .primary_monitor()
                        {
                            let monitor_size = monitor.size();
                            let dialog_width = 1000.0;
                            let dialog_height = 100.0;
                            let pos_x = (monitor_size.width as f64 - dialog_width) / 2.0;
                            let pos_y = 60.0;

                            tauri::WebviewWindowBuilder::new(
                                &app_handle_clone,
                                "notification_dialog",
                                tauri::WebviewUrl::App("/notification".into()),
                            )
                            .inner_size(dialog_width, dialog_height)
                            .position(pos_x, pos_y)
                            .always_on_top(true)
                            .decorations(false)
                            .transparent(true)
                            .build()
                            .unwrap();
                        }
                    }
                }).unwrap();

                let response = Response::from_string("Dialog triggered");
                request.respond(response).unwrap();
            }
        }
        port
    });
}

#[cfg_attr(mobile, tauri::mobile_entry_point)]
pub fn run() {
    tauri::Builder::default()
        .plugin(tauri_plugin_store::Builder::default().build())
        .plugin(tauri_plugin_autostart::init(MacosLauncher::LaunchAgent, Some(vec!["--flag-from-autostart"])))
        .plugin(
            tauri_plugin_log::Builder::new()
                .level(tauri_plugin_log::log::LevelFilter::Info)
                .target(tauri_plugin_log::Target::new(
                    tauri_plugin_log::TargetKind::Stdout,
                ))
                .build(),
        )
        .plugin(tauri_plugin_notification::init())
        .plugin(tauri_plugin_shell::init())
        .invoke_handler(tauri::generate_handler![
            get_settings,
            set_daylit_dir,
            set_font_size,
            set_launch_at_login,
            get_notification_payload,
            close_notification_window
        ])
        .on_window_event(|window, event| match event {
            tauri::WindowEvent::CloseRequested { api, .. } => {
                if window.label() != "main" {
                    return;
                }
                api.prevent_close();
                info!("Main window close requested, hiding instead");
                window.hide().unwrap();
            }
            _ => {}
        })
        .setup(|app| {
            // --- State and Store Setup ---
            let store = app.store("settings.json")?;
            if store.get("settings").is_none() {
                store.set(
                    "settings",
                    serde_json::to_value(Settings::default()).unwrap(),
                );
                store.save().unwrap();
            }
            let app_state = AppState {
                settings: store,
                payload: Default::default(),
                lockfile_path: Mutex::new(None),
            };
            app.manage(app_state);

            // --- System Tray Menu ---
            let handle = app.handle().clone();
            let quit_i = MenuItem::with_id(app, "quit", "Quit", true, None::<&str>)?;
            let show_i = MenuItem::with_id(app, "show", "Show", true, None::<&str>)?;
            let settings_i = MenuItem::with_id(app, "settings", "Settings", true, None::<&str>)?;
            let menu = Menu::with_items(app, &[&show_i, &settings_i, &quit_i])?;
            let tray = TrayIconBuilder::new()
                .menu(&menu)
                .show_menu_on_left_click(true)
                .on_menu_event(move |_tray, event| match event.id().as_ref() {
                    "quit" => {
                        std::process::exit(0);
                    }
                    "show" => {
                        let webview_window = handle.get_webview_window("main").unwrap();
                        webview_window.show().unwrap();
                        webview_window.set_focus().unwrap();
                    }
                    "settings" => {
                        if let Some(win) = handle.get_webview_window("settings") {
                            win.set_focus().unwrap();
                        } else {
                            tauri::WebviewWindowBuilder::new(
                                &handle, "settings", tauri::WebviewUrl::App("/settings".into())
                            )
                            .title("Daylit Settings")
                            .inner_size(400.0, 300.0)
                            .resizable(false)
                            .build()
                            .unwrap();
                        }
                    }
                    _ => (),
                })
                .build(app)?;
            tray.set_icon(Some(Image::from_path("icons/tray-icon.png")?))?;

            // --- App finalization ---
            let main_window = app.get_webview_window("main").unwrap();
            main_window.hide().unwrap();
            start_webhook_server(app.handle().clone());

            // --- Lock File Cleanup on Exit ---
            let app_handle = app.handle().clone();
            app.listen("tauri://destroyed", move |_| {
                let state: State<AppState> = app_handle.state();
                if let Some(path) = state.lockfile_path.lock().unwrap().as_ref() {
                    if path.exists() {
                        fs::remove_file(path).unwrap();
                    }
                }
            });

            Ok(())
        })
        .run(tauri::generate_context!())
        .expect("error while running tauri application");
}
