use crate::state::LOCKFILE_NAME;
use crate::state::{AppState, Settings, UpdatePayload, WebhookPayload};
use rand::distributions::Alphanumeric;
use rand::{thread_rng, Rng};
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
                error!(
                    "Failed to get webhook server IP address - webhook server will not be available for notifications"
                );
                return;
            }
        };

        // --- Create Lock File ---
        let state: State<AppState> = app_handle.state();
        let settings = Settings::load(&state.settings);

        // Generate a secure random secret (32 characters)
        let secret: String = thread_rng()
            .sample_iter(&Alphanumeric)
            .take(32)
            .map(char::from)
            .collect();

        // Store the secret in AppState for validation
        *state
            .secret
            .lock()
            .expect("Failed to acquire secret lock") = Some(secret.clone());

        let config_dir = if let Some(dir) = settings.lockfile_dir {
            std::path::PathBuf::from(dir)
        } else {
            match app_handle.path().app_config_dir() {
                Ok(dir) => dir,
                Err(e) => {
                    error!(
                        "Failed to get app config directory: {} - webhook server will not be available",
                        e
                    );
                    return;
                }
            }
        };

        if let Err(e) = fs::create_dir_all(&config_dir) {
            error!(
                "Failed to create config directory: {} - webhook server will not be available",
                e
            );
            return;
        }
        let lock_file_path = config_dir.join(LOCKFILE_NAME);
        let pid = std::process::id();
        let lock_content = format!("{}|{}|{}", port, pid, secret);
        if let Err(e) = fs::write(&lock_file_path, lock_content) {
            error!("Failed to write lock file: {}", e);
            return;
        }

        // Set file permissions to 0600 (rw-------)
        #[cfg(unix)]
        {
            let permissions = fs::Permissions::from_mode(0o600);
            if let Err(e) = fs::set_permissions(&lock_file_path, permissions) {
                error!("Failed to set lock file permissions: {}", e);
                return;
            }
        }

        // Store the path so we can delete it later
        *state
            .lockfile_path
            .lock()
            .expect("Failed to acquire lockfile_path lock") = Some(lock_file_path);

        info!("Webhook server started on port: {}", port);

        for mut request in server.incoming_requests() {
            if request.method().as_str() != "POST" {
                continue;
            }

            // Validate X-Daylit-Secret header
            let auth_valid = {
                let state: State<AppState> = app_handle.state();
                let expected_secret = state.secret.lock().expect("Failed to acquire secret lock");

                if let Some(expected) = expected_secret.as_ref() {
                    // Check for X-Daylit-Secret header
                    request
                        .headers()
                        .iter()
                        .find(|h| h.field.as_str().eq_ignore_ascii_case("X-Daylit-Secret"))
                        .and_then(|h| Some(h.value.as_str() == expected))
                        .unwrap_or(false)
                } else {
                    // If no secret is set (shouldn't happen), reject
                    false
                }
            };

            if !auth_valid {
                error!("Unauthorized request: missing or invalid X-Daylit-Secret header");
                let response = Response::from_string("Unauthorized").with_status_code(401);
                if let Err(e) = request.respond(response) {
                    error!("Failed to respond with error: {}", e);
                }
                continue;
            }

            let mut content = String::new();
            if let Err(e) = request.as_reader().read_to_string(&mut content) {
                error!("Failed to read request body: {}", e);
                continue;
            }

            if let Ok(payload) = serde_json::from_str::<WebhookPayload>(&content) {
                let state: State<AppState> = app_handle.state();
                *state
                    .payload
                    .lock()
                    .expect("Failed to acquire payload lock") = Some(payload.clone());

                let app_handle_clone = app_handle.clone();
                if let Err(e) = app_handle.run_on_main_thread(move || {
                    // --- Re-use or Create Window Logic ---
                    if let Some(existing_window) =
                        app_handle_clone.get_webview_window("notification_dialog")
                    {
                        info!("Dialog exists. Re-using and sending new data.");
                        if let Err(e) = existing_window.set_focus() {
                            error!("Failed to set window focus: {}", e);
                        }
                        if let Err(e) = existing_window.emit(
                            "update_notification",
                            &UpdatePayload {
                                text: payload.text,
                                duration_ms: payload.duration_ms,
                            },
                        ) {
                            error!("Failed to emit update notification: {}", e);
                        }
                    } else {
                        info!("Dialog does not exist. Creating a new one.");
                        if let Some(main_window) = app_handle_clone.get_webview_window("main") {
                            if let Ok(Some(monitor)) = main_window.primary_monitor() {
                                let monitor_size = monitor.size();
                                let dialog_width = 1000.0;
                                let dialog_height = 100.0;
                                let pos_x = (monitor_size.width as f64 - dialog_width) / 2.0;
                                let pos_y = 60.0;

                                if let Err(e) = tauri::WebviewWindowBuilder::new(
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
                                {
                                    error!("Failed to build notification dialog: {}", e);
                                }
                            } else {
                                error!("Failed to get primary monitor");
                            }
                        } else {
                            error!("Main window not found");
                        }
                    }
                }) {
                    error!("Failed to run on main thread: {}", e);
                }

                let response = Response::from_string("Dialog triggered");
                if let Err(e) = request.respond(response) {
                    error!("Failed to respond to webhook request: {}", e);
                }
            } else {
                error!("Failed to parse webhook payload");
                let response = Response::from_string("Invalid payload").with_status_code(400);
                if let Err(e) = request.respond(response) {
                    error!("Failed to respond with error: {}", e);
                }
            }
        }
    });
}
