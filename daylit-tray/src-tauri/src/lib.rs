use std::fs;
use std::sync::Mutex;
use tauri::{
    Listener, LogicalPosition, LogicalSize, Manager, State,
    image::Image,
    menu::{Menu, MenuItem},
    tray::TrayIconBuilder,
};
use tauri_plugin_autostart::MacosLauncher;
use tauri_plugin_log::log::info;
use tauri_plugin_store::StoreExt;

mod commands;
mod scheduler;
mod server;
mod state;

use commands::*;
use server::start_webhook_server;
use state::{AppState, Settings};

use crate::scheduler::start_scheduler_thread;

const WINDOW_WIDTH: f64 = 560.0;
const WINDOW_HEIGHT: f64 = 600.0;
const WINDOW_X: f64 = 400.0;
const WINDOW_Y: f64 = 400.0;

#[cfg_attr(mobile, tauri::mobile_entry_point)]
pub fn run() {
    tauri::Builder::default()
        .plugin(tauri_plugin_store::Builder::default().build())
        .plugin(tauri_plugin_autostart::init(
            MacosLauncher::LaunchAgent,
            Some(vec!["--flag-from-autostart"]),
        ))
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
            save_settings,
            get_notification_payload,
            close_notification_window
        ])
        .on_window_event(|window, event| {
            if let tauri::WindowEvent::CloseRequested { api, .. } = event {
                if window.label() != "main" {
                    return;
                }
                api.prevent_close();
                info!("Main window close requested, hiding instead");
                window.hide().unwrap();
            }
        })
        .setup(|app| {
            // Debugging path
            eprintln!(
                "Env XDG_CONFIG_HOME: {:?}",
                std::env::var("XDG_CONFIG_HOME")
            );
            eprintln!("Env XDG_DATA_HOME: {:?}", std::env::var("XDG_DATA_HOME"));

            match app.path().app_config_dir() {
                Ok(path) => eprintln!("App config dir: {:?}", path),
                Err(e) => eprintln!("Failed to get app config dir: {:?}", e),
            }

            match app.path().app_data_dir() {
                Ok(path) => eprintln!("App data dir: {:?}", path),
                Err(e) => eprintln!("Failed to get app data dir: {:?}", e),
            }

            // --- State and Store Setup ---
            let store = match app.store("settings.json") {
                Ok(s) => s,
                Err(e) => {
                    eprintln!("Store creation failed: {:?}", e);
                    return Err(e.into());
                }
            };
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
                secret: Mutex::new(None),
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
                        webview_window
                            .set_size(LogicalSize::new(WINDOW_WIDTH, WINDOW_HEIGHT))
                            .unwrap();
                        webview_window
                            .set_position(LogicalPosition::new(WINDOW_X, WINDOW_Y))
                            .unwrap();
                        webview_window.show().unwrap();
                        webview_window.set_focus().unwrap();
                    }
                    "settings" => {
                        if let Some(win) = handle.get_webview_window("settings") {
                            win.set_focus().unwrap();
                        } else {
                            tauri::WebviewWindowBuilder::new(
                                &handle,
                                "settings",
                                tauri::WebviewUrl::App("/settings".into()),
                            )
                            .title("Daylit Tray Settings")
                            .inner_size(WINDOW_WIDTH, WINDOW_HEIGHT)
                            .position(WINDOW_X, WINDOW_Y)
                            .resizable(false)
                            .build()
                            .unwrap();
                        }
                    }
                    _ => (),
                })
                .build(app)?;
            tray.set_icon(Some(Image::from_path("icons/tray-icon.png")?))?;
            tray.set_tooltip(Some("Daylit Tray"))?;

            // --- App finalization ---
            let main_window = app.get_webview_window("main").unwrap();
            main_window
                .set_size(LogicalSize::new(WINDOW_WIDTH, WINDOW_HEIGHT))
                .unwrap();
            main_window
                .set_position(LogicalPosition::new(WINDOW_X, WINDOW_Y))
                .unwrap();
            main_window.hide().unwrap();

            start_webhook_server(app.handle().clone());
            start_scheduler_thread(app.handle().clone());

            // --- Lock File Cleanup on Exit ---
            let app_handle = app.handle().clone();
            app.listen("tauri://destroyed", move |_| {
                let state: State<AppState> = app_handle.state();
                if let Some(path) = state.lockfile_path.lock().unwrap().as_ref()
                    && path.exists()
                {
                    fs::remove_file(path).unwrap();
                }
            });

            Ok(())
        })
        .run(tauri::generate_context!())
        .expect("error while running tauri application");
}
