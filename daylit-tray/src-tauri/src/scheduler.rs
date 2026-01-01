use std::time::Duration;
use std::{process::Command, thread};
use tauri::AppHandle;
use tauri_plugin_log::log::{error, info};

pub fn start_scheduler_thread(_app_handle: AppHandle) {
    thread::spawn(move || {
        loop {
            // Run every minute to check for upcoming tasks
            thread::sleep(Duration::from_secs(60));

            // Execute 'daylit notify'
            // This command checks the schedule in the database and sends a webhook
            // back to the tray app's server if a notification is due.
            match Command::new("daylit").arg("notify").output() {
                Ok(output) => {
                    if output.status.success() {
                        info!("daylit notify executed successfully");
                    } else {
                        error!(
                            "daylit notify failed with status: {} stderr: {}",
                            output.status,
                            String::from_utf8_lossy(&output.stderr)
                        );
                    }
                }
                Err(e) => {
                    error!("Failed to execute daylit notify command: {}", e);
                }
            }
        }
    });
}
