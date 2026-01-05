use crate::state::LOCKFILE_NAME;
use crate::state::{AppState, Settings, UpdatePayload, WebhookPayload};
use rand::Rng;
use rand::distributions::Alphanumeric;
use rand::rngs::OsRng;
use std::fs;
#[cfg(unix)]
use std::os::unix::fs::PermissionsExt;
use std::thread;
use subtle::ConstantTimeEq;
use tauri::{AppHandle, Emitter, Manager, State};
use tauri_plugin_log::log::{error, info};
use tauri_plugin_notification::NotificationExt;
use tiny_http::{Header, Response, Server};

fn validate_request(headers: &[Header], expected_secret: &str) -> bool {
    headers
        .iter()
        .find(|h| {
            h.field
                .as_str()
                .as_str()
                .eq_ignore_ascii_case("X-Daylit-Secret")
        })
        .map(|h| {
            // Use constant-time comparison to prevent timing-based side-channel attacks
            h.value
                .as_str()
                .as_bytes()
                .ct_eq(expected_secret.as_bytes())
                .into()
        })
        .unwrap_or(false)
}

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

        // Generate a cryptographically secure random secret (32 characters)
        // Using OsRng which is explicitly a cryptographically secure RNG
        let secret: String = OsRng
            .sample_iter(&Alphanumeric)
            .take(32)
            .map(char::from)
            .collect();

        // Store the secret in AppState for validation
        *state.secret.lock().expect("Failed to acquire secret lock") = Some(secret.clone());

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
                    validate_request(request.headers(), expected)
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

                // Check if we should use native notifications
                let settings = Settings::load(&state.settings);
                
                if settings.use_native_notifications {
                    // Use native system notifications
                    // Note: The duration_ms field from the payload is not used here as
                    // native notification duration is controlled by the operating system.
                    // Custom notifications (else branch) do respect the duration_ms setting.
                    info!("Using native notification");
                    if let Err(e) = app_handle
                        .notification()
                        .builder()
                        .title("Daylit")
                        .body(&payload.text)
                        .show()
                    {
                        error!("Failed to show native notification: {}", e);
                    }
                } else {
                    // Use custom window notification (existing behavior)
                    info!("Received webhook payload. Scheduling on main thread.");
                    let app_handle_clone = app_handle.clone();
                    if let Err(e) = app_handle.run_on_main_thread(move || {
                        info!("Running on main thread.");
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
                }

                let response = Response::from_string("Notification triggered");
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

#[cfg(test)]
mod tests {
    use super::*;
    use tiny_http::Header;

    #[test]
    fn test_validate_request_success() {
        let secret = "my_secret_token";
        let headers = vec![
            Header::from_bytes("Content-Type", "application/json").unwrap(),
            Header::from_bytes("X-Daylit-Secret", "my_secret_token").unwrap(),
        ];
        assert!(validate_request(&headers, secret));
    }

    #[test]
    fn test_validate_request_failure_wrong_secret() {
        let secret = "my_secret_token";
        let headers = vec![Header::from_bytes("X-Daylit-Secret", "wrong_token").unwrap()];
        assert!(!validate_request(&headers, secret));
    }

    #[test]
    fn test_validate_request_failure_missing_header() {
        let secret = "my_secret_token";
        let headers = vec![Header::from_bytes("Content-Type", "application/json").unwrap()];
        assert!(!validate_request(&headers, secret));
    }

    #[test]
    fn test_validate_request_case_insensitive_header_name() {
        let secret = "my_secret_token";
        let headers = vec![Header::from_bytes("x-daylit-secret", "my_secret_token").unwrap()];
        assert!(validate_request(&headers, secret));
    }
}
