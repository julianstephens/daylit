use crate::state::{AppState, Settings};
use std::time::Duration;
use std::{process::Command, thread};
use tauri::AppHandle;
use tauri::Manager;
use tauri_plugin_log::log::{error, info};

// --- Abstraction for testing ---
pub struct CommandOutput {
    pub success: bool,
    pub status_code: Option<i32>,
    pub stderr: Vec<u8>,
}

pub trait CommandRunner {
    fn run(&self, program: &str, args: &[&str]) -> std::io::Result<CommandOutput>;
}

struct RealCommandRunner;

impl CommandRunner for RealCommandRunner {
    fn run(&self, program: &str, args: &[&str]) -> std::io::Result<CommandOutput> {
        let output = Command::new(program).args(args).output()?;
        Ok(CommandOutput {
            success: output.status.success(),
            status_code: output.status.code(),
            stderr: output.stderr,
        })
    }
}

fn run_notify_check<R: CommandRunner>(daylit_path: &str, runner: &R) {
    match runner.run(daylit_path, &["notify"]) {
        Ok(output) => {
            if output.success {
                // Successfully ran 'daylit notify' with no errors
                info!("daylit notify executed successfully");
            } else {
                // 'daylit notify' ran but returned a non-zero exit code
                error!(
                    "daylit notify failed with status: {:?} stderr: {}",
                    output.status_code,
                    String::from_utf8_lossy(&output.stderr)
                );
            }
        }
        Err(e) => {
            error!(
                "Failed to execute daylit notify command at '{}': {}",
                daylit_path, e
            );
        }
    }
}

fn get_scheduler_interval() -> u64 {
    std::env::var("DAYLIT_SCHEDULER_INTERVAL_MS")
        .ok()
        .and_then(|v| v.parse::<u64>().ok())
        .unwrap_or(60000)
}

pub fn start_scheduler_thread(app_handle: AppHandle) {
    thread::spawn(move || {
        let runner = RealCommandRunner;
        loop {
            // Determine sleep interval from env var or default to 60 seconds
            let interval_ms = get_scheduler_interval();

            // Run every minute (or configured interval) to check for upcoming tasks
            thread::sleep(Duration::from_millis(interval_ms));

            // Get the configured daylit path or default to "daylit"
            let daylit_path = {
                let state: tauri::State<AppState> = app_handle.state();
                let settings = Settings::load(&state.settings);
                settings
                    .daylit_path
                    .clone()
                    .unwrap_or_else(|| "daylit".to_string())
            };

            // Execute 'daylit notify' (or custom path to daylit)
            run_notify_check(&daylit_path, &runner);
        }
    });
}

#[cfg(test)]
mod tests {
    use super::*;
    use serial_test::serial;
    use std::cell::RefCell;
    use temp_env::with_var;

    struct MockCommandRunner {
        expected_program: String,
        expected_args: Vec<String>,
        result: std::io::Result<CommandOutput>,
        called: RefCell<bool>,
    }

    impl MockCommandRunner {
        fn new(program: &str, result: std::io::Result<CommandOutput>) -> Self {
            Self {
                expected_program: program.to_string(),
                expected_args: vec!["notify".to_string()],
                result,
                called: RefCell::new(false),
            }
        }
    }

    impl CommandRunner for MockCommandRunner {
        fn run(&self, program: &str, args: &[&str]) -> std::io::Result<CommandOutput> {
            *self.called.borrow_mut() = true;
            assert_eq!(program, self.expected_program);
            assert_eq!(args, &self.expected_args[..]);

            // Clone the result
            match &self.result {
                Ok(out) => Ok(CommandOutput {
                    success: out.success,
                    status_code: out.status_code,
                    stderr: out.stderr.clone(),
                }),
                Err(e) => Err(std::io::Error::new(e.kind(), e.to_string())),
            }
        }
    }

    #[test]
    fn test_run_notify_check_success() {
        let runner = MockCommandRunner::new(
            "daylit",
            Ok(CommandOutput {
                success: true,
                status_code: Some(0),
                stderr: vec![],
            }),
        );

        run_notify_check("daylit", &runner);
        assert!(*runner.called.borrow());
    }

    #[test]
    fn test_run_notify_check_failure_exit_code() {
        let runner = MockCommandRunner::new(
            "custom/path/daylit",
            Ok(CommandOutput {
                success: false,
                status_code: Some(1),
                stderr: b"error message".to_vec(),
            }),
        );

        run_notify_check("custom/path/daylit", &runner);
        assert!(*runner.called.borrow());
    }

    #[test]
    fn test_run_notify_check_execution_error() {
        let runner = MockCommandRunner::new(
            "daylit",
            Err(std::io::Error::new(
                std::io::ErrorKind::NotFound,
                "not found",
            )),
        );

        run_notify_check("daylit", &runner);
        assert!(*runner.called.borrow());
    }

    #[test]
    #[serial]
    fn test_get_scheduler_interval_default() {
        // Test default interval when env var is not set
        with_var("DAYLIT_SCHEDULER_INTERVAL_MS", None::<String>, || {
            assert_eq!(get_scheduler_interval(), 60000);
        });
    }

    #[test]
    #[serial]
    fn test_get_scheduler_interval_custom() {
        // Test custom interval when env var is set
        with_var("DAYLIT_SCHEDULER_INTERVAL_MS", Some("500"), || {
            assert_eq!(get_scheduler_interval(), 500);
        });
    }

    #[test]
    #[serial]
    fn test_get_scheduler_interval_invalid() {
        // Test that invalid values fall back to default
        with_var("DAYLIT_SCHEDULER_INTERVAL_MS", Some("invalid"), || {
            assert_eq!(get_scheduler_interval(), 60000);
        });
    }
}
