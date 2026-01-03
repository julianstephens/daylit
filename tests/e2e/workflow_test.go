package e2e

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestEndToEndWorkflow(t *testing.T) {
	// 1. Setup Environment
	// Allow overriding bin dir via env var, default to ../../bin (relative to tests/e2e)
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get cwd: %v", err)
	}

	var binDir string
	if os.Getenv("DAYLIT_BIN_DIR") != "" {
		binDir = os.Getenv("DAYLIT_BIN_DIR")
	} else {
		// Check if we are in root
		if _, err := os.Stat(filepath.Join(cwd, "daylit-cli")); err == nil {
			// We are in root
			binDir = filepath.Join(cwd, "bin")
		} else {
			// Assume we are in tests/e2e
			binDir = filepath.Join(cwd, "..", "..", "bin")
		}
	}

	binDir, _ = filepath.Abs(binDir)
	t.Logf("Using bin dir: %s", binDir)

	cliPath := filepath.Join(binDir, "daylit")
	trayPath := filepath.Join(binDir, "daylit-tray")

	// Verify binaries exist
	if _, err := os.Stat(cliPath); os.IsNotExist(err) {
		t.Logf("CLI binary not found at %s. Attempting to build...", cliPath)
		t.Fatalf("CLI binary not found at %s. Please build it first.", cliPath)
	}
	if _, err := os.Stat(trayPath); os.IsNotExist(err) {
		t.Fatalf("Tray binary not found at %s. Please build it first.", trayPath)
	}

	// Create temp home for isolation
	tempDir := t.TempDir()
	t.Logf("Running test in temp dir: %s", tempDir)

	// Set environment variables for isolation
	env := os.Environ()
	var cleanEnv []string
	for _, e := range env {
		if !strings.HasPrefix(e, "XDG_CONFIG_HOME=") && !strings.HasPrefix(e, "HOME=") && !strings.HasPrefix(e, "DAYLIT_CONFIG=") {
			cleanEnv = append(cleanEnv, e)
		}
	}

	cleanEnv = append(cleanEnv, fmt.Sprintf("XDG_CONFIG_HOME=%s", tempDir))
	cleanEnv = append(cleanEnv, fmt.Sprintf("HOME=%s", tempDir))
	cleanEnv = append(cleanEnv, fmt.Sprintf("DAYLIT_CONFIG=%s", filepath.Join(tempDir, "daylit", "daylit.db")))

	// Add binDir to PATH
	pathEnv := fmt.Sprintf("PATH=%s%c%s", binDir, os.PathListSeparator, os.Getenv("PATH"))
	foundPath := false
	for i, e := range cleanEnv {
		if strings.HasPrefix(e, "PATH=") {
			cleanEnv[i] = pathEnv
			foundPath = true
			break
		}
	}
	if !foundPath {
		cleanEnv = append(cleanEnv, pathEnv)
	}

	// 2. Initialize CLI
	t.Log("Initializing CLI...")
	runCmd(t, cliPath, cleanEnv, "init")

	// Enable notifications
	t.Log("Enabling notifications...")
	runCmd(t, cliPath, cleanEnv, "config", "set", "notifications_enabled", "true")

	// 3. Start Tray (Background)
	t.Log("Starting Tray...")
	// Set short interval for testing
	trayEnv := append(cleanEnv, "DAYLIT_SCHEDULER_INTERVAL_MS=2000")
	trayEnv = append(trayEnv, "RUST_LOG=info")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	trayCmd := exec.CommandContext(ctx, trayPath)
	trayCmd.Env = trayEnv

	stdoutPipe, err := trayCmd.StdoutPipe()
	if err != nil {
		t.Fatalf("Failed to get stdout pipe: %v", err)
	}

	// Capture stderr
	var stderrBuf bytes.Buffer
	trayCmd.Stderr = &stderrBuf

	if err := trayCmd.Start(); err != nil {
		t.Fatalf("Failed to start tray: %v", err)
	}
	t.Log("Tray started")

	defer func() {
		cancel()
		trayCmd.Wait()
		if t.Failed() {
			t.Logf("Tray Stderr: %s", stderrBuf.String())
		}
	}()

	// 4. Wait for Lockfile (Tray Ready)
	// Lockfile location: $XDG_CONFIG_HOME/com.daylit.daylit-tray/daylit-tray.lock
	lockfilePath := filepath.Join(tempDir, "com.daylit.daylit-tray", "daylit-tray.lock")
	t.Logf("Waiting for lockfile at %s", lockfilePath)
	waitForFile(t, lockfilePath, 10*time.Second)
	t.Log("Lockfile found, Tray is ready")

	// 5. Add Task for "Now"
	t.Log("Adding task...")
	runCmd(t, cliPath, cleanEnv, "task", "add", "Test Task", "--duration", "30")

	// Schedule it for now.
	now := time.Now()
	timeStr := now.Format("15:04")
	t.Logf("Scheduling task for %s", timeStr)
	runCmd(t, cliPath, cleanEnv, "plan", "add", "Test Task", timeStr)

	// 6. Monitor Logs for Success
	t.Log("Waiting for notification logs...")
	scanner := bufio.NewScanner(stdoutPipe)
	success := false

	// Channel to signal success
	doneCh := make(chan bool)

	go func() {
		for scanner.Scan() {
			line := scanner.Text()
			// t.Logf("Tray: %s", line) // Uncomment for debugging

			// Check for success message
			// "daylit notify executed successfully" means the scheduler ran the command
			// "Received live update" or similar means the webhook was hit
			if strings.Contains(line, "Received live update") || strings.Contains(line, "Notification received") {
				t.Logf("Found success log: %s", line)
				doneCh <- true
				return
			}
		}
		if err := scanner.Err(); err != nil {
			t.Logf("Scanner error: %v", err)
		}
	}()

	select {
	case <-doneCh:
		success = true
		t.Log("Verified notification flow!")
	case <-time.After(30 * time.Second):
		t.Errorf("Timed out waiting for notification log message")
	}

	if !success {
		t.Fail()
	}
}

func runCmd(t *testing.T, path string, env []string, args ...string) {
	cmd := exec.Command(path, args...)
	cmd.Env = env
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Command %s %v failed: %v\nOutput: %s", path, args, err, out)
	}
}

func waitForFile(t *testing.T, path string, timeout time.Duration) {
	start := time.Now()
	for {
		if _, err := os.Stat(path); err == nil {
			return
		}
		if time.Since(start) > timeout {
			t.Fatalf("Timed out waiting for file: %s", path)
		}
		time.Sleep(100 * time.Millisecond)
	}
}
