#[cfg_attr(mobile, tauri::mobile_entry_point)]
use serde::{Deserialize, Serialize};
use std::io::Write;
use std::thread;
use std::{fs, sync::Mutex};
use tauri::Emitter;
use tauri::Listener;
use tauri::{
    image::Image,
    menu::{Menu, MenuItem},
    tray::TrayIconBuilder,
    AppHandle, Manager, State, WebviewWindow,
};
use tauri_plugin_log::log::info;
use tiny_http::{Response, Server};

// --- Struct Definitions for State and Payloads ---

#[derive(Serialize, Deserialize, Debug, Clone)]
pub struct Settings {
    font_size: String,
    launch_at_login: bool,
}

impl Default for Settings {
    fn default() -> Self {
        Self {
            font_size: "medium".into(),
            launch_at_login: false,
        }
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
    pub settings: Mutex<Store<Settings>>,
    pub payload: Mutex<Option<WebhookPayload>>,
}

// --- Tauri Commands ---

#[tauri::command]
fn get_settings(state: State<AppState>) -> Result<Settings, String> {
    Ok(state
        .settings
        .lock()
        .unwrap()
        .get("settings")
        .cloned()
        .unwrap_or_default())
}

#[tauri::command]
fn set_font_size(font_size: String, state: State<AppState>) -> Result<(), String> {
    let mut settings = state
        .settings
        .lock()
        .unwrap()
        .get("settings")
        .cloned()
        .unwrap_or_default();
    settings.font_size = font_size;
    state
        .settings
        .lock()
        .unwrap()
        .insert("settings".into(), settings)
        .map_err(|e| e.to_string())?;
    state
        .settings
        .lock()
        .unwrap()
        .save()
        .map_err(|e| e.to_string())
}

#[tauri::command]
async fn set_launch_at_login(enable: bool, app: AppHandle) -> Result<(), String> {
    let autostart_manager = app.autostart_manager();
    if enable {
        autostart_manager.enable().await.map_err(|e| e.to_string())?;
    } else {
        autostart_manager.disable().await.map_err(|e| e.to_string())?;
    }
    
    let state: State<AppState> = app.state();
    let mut settings = state.settings.lock().unwrap().get("settings").cloned().unwrap_or_default();
    settings.launch_at_login = enable;
    state.settings.lock().unwrap().insert("settings".into(), settings).map_err(|e| e.to_string())?;
    state.settings.lock().unwrap().save().map_err(|e| e.to_string())
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
        let config_dir = app_handle.path().app_config_dir().unwrap();
        fs::create_dir_all(&config_dir).unwrap();
        let lock_file_path = config_dir.join("daylit.lock");
        let pid = std::process::id();
        let lock_content = format!("{}|{}", port, pid);
        let mut file = fs::File::create(lock_file_path).unwrap();
        file.write_all(lock_content.as_bytes()).unwrap();

        println!("Webhook server started on port: {}", port);

        thread::spawn(move || {
            for mut request in server.incoming_requests() {
                if request.method().as_str() != "POST" {
                    continue;
                }

                let mut content = String::new();
                request.as_reader().read_to_string(&mut content).unwrap();

                if let Ok(payload) = serde_json::from_str::<WebhookPayload>(&content) {
                    let state: State<AppState> = app_handle.state();
                    *state.0.lock().unwrap() = Some(payload.clone());

                    let app_handle_clone = app_handle.clone();
                    app_handle
                        .run_on_main_thread(move || {
                            if let Some(existing_window) =
                                app_handle_clone.get_webview_window("notification_dialog")
                            {
                                println!("Dialog exists. Re-using and sending new data.");

                                existing_window.set_focus().unwrap();

                                existing_window
                                    .emit(
                                        "update_notification",
                                        &UpdatePayload {
                                            text: payload.text,
                                            duration_ms: payload.duration_ms,
                                        },
                                    )
                                    .unwrap();
                            } else {
                                println!("Dialog does not exist. Creating a new one.");
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
                        })
                        .unwrap();

                    let response = Response::from_string("Dialog triggered");
                    request.respond(response).unwrap();
                }
            }
        });
        port
    });
}

pub fn run() {
    tauri::Builder::default()
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
        .manage(AppState(Default::default()))
        .invoke_handler(tauri::generate_handler![
            get_notification_payload,
            close_notification_window
        ])
        .on_window_event(|window, event| match event {
            tauri::WindowEvent::CloseRequested { api, .. } => {
                if window.label() != "main" {
                    return;
                }
                api.prevent_close();
                info!("Main window close requested, hiding instead.");
                window.hide().unwrap();
            }
            _ => {}
        })
        .setup(|app| {
            let quit_i = MenuItem::with_id(app, "quit", "Quit", true, None::<&str>)?;
            let show_i = MenuItem::with_id(app, "show", "Show", true, None::<&str>)?;
            let menu = Menu::with_items(app, &[&show_i, &quit_i])?;
            let handle = app.handle().clone();
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
                    _ => (),
                })
                .build(app)?;
            tray.set_icon(Some(Image::from_path("icons/tray-icon.png")?))?;

            start_webhook_server(app.handle().clone());
            let main_window = app.get_webview_window("main").unwrap();
            main_window.hide().unwrap();

            let app_handle = app.handle().clone();
            app.listen("tauri://destroyed", move |_| {
                let config_dir = app_handle.path().app_config_dir().unwrap();
                let lock_file_path = config_dir.join("daylit.lock");
                if lock_file_path.exists() {
                    fs::remove_file(lock_file_path).unwrap();
                }
            });

            Ok(())
        })
        .run(tauri::generate_context!())
        .expect("error while running tauri application");
}
