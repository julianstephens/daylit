use crate::state::LOCKFILE_NAME;
use crate::state::{AppState, Settings, UpdatePayload, WebhookPayload};
use std::fs;
#[cfg(unix)]
use std::os::unix::fs::PermissionsExt;
use std::thread;
use tauri::{AppHandle, Emitter, Manager, State};
use tauri_plugin_log::log::{error, info};
use tiny_http::{Response, Server};

pub fn start_webhook_server(app_handle: AppHandle) {
    thread::spawn(move || {
        // Bind to port 0 to let the OS choose an available port
        let server = match Server::http("127.0.0.1:0") {
            Ok(s) => s,
            Err(e) => {
                error!("Failed to create webhook server: {}", e);
                return;
            }
        };
        
        let port = match server.server_addr().to_ip() {
            Some(addr) => addr.port(),
            None => {
                error!("Failed to get webhook server IP address");
                return;
            }
        };

        // --- Create Lock File ---
        let state: State<AppState> = app_handle.state();
        let settings = Settings::load(&state.settings);

        let config_dir = if let Some(dir) = settings.lockfile_dir {
            std::path::PathBuf::from(dir)
        } else {
            app_handle.path().app_config_dir().unwrap()
        };

        fs::create_dir_all(&config_dir).unwrap();
        let lock_file_path = config_dir.join(LOCKFILE_NAME);
        let pid = std::process::id();
        let lock_content = format!("{}|{}", port, pid);
        fs::write(&lock_file_path, lock_content).expect("Failed to write lock file");

        // Set file permissions to 0600 (rw-------)
        #[cfg(unix)]
        {
            let permissions = fs::Permissions::from_mode(0o600);
            fs::set_permissions(&lock_file_path, permissions)
                .expect("Failed to set lock file permissions");
        }

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
                app_handle
                    .run_on_main_thread(move || {
                        // --- Re-use or Create Window Logic ---
                        if let Some(existing_window) =
                            app_handle_clone.get_webview_window("notification_dialog")
                        {
                            info!("Dialog exists. Re-using and sending new data.");
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
                    })
                    .unwrap();

                let response = Response::from_string("Dialog triggered");
                request.respond(response).unwrap();
            }
        }
    });
}
