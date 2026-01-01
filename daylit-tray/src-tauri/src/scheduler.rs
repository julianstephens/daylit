use std::time::Duration;
use std::{process::Command, thread};
use tauri::AppHandle;

pub fn start_scheduler_thread(_app_handle: AppHandle) {
    thread::spawn(move || {
        loop {
            // Run every minute to check for upcoming tasks
            thread::sleep(Duration::from_secs(60));

            // Execute 'daylit notify'
            // This command checks the schedule in the database and sends a webhook
            // back to the tray app's server if a notification is due.
            let _ = Command::new("daylit").arg("notify").output();
        }
    });
}
