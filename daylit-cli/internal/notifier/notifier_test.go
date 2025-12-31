package notifier

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	ps "github.com/mitchellh/go-ps"
)

// Mock Process
type mockProcess struct {
	pid        int
	executable string
}

func (m *mockProcess) Pid() int {
	return m.pid
}

func (m *mockProcess) PPid() int {
	return 0
}

func (m *mockProcess) Executable() string {
	return m.executable
}

func TestGetTrayAppConfigDir(t *testing.T) {
	// Setup temp home dir
	tempDir, err := os.MkdirTemp("", "daylit-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Mock userConfigDirFunc
	oldUserConfigDirFunc := userConfigDirFunc
	defer func() { userConfigDirFunc = oldUserConfigDirFunc }()
	userConfigDirFunc = func() (string, error) {
		return tempDir, nil
	}

	// Test 1: Default
	expectedDefault := filepath.Join(tempDir, TrayAppIdentifier)
	dir, err := getTrayAppConfigDir()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if dir != expectedDefault {
		t.Errorf("expected %s, got %s", expectedDefault, dir)
	}

	// Test 2: Custom setting
	trayConfigDir := filepath.Join(tempDir, TrayAppIdentifier)
	err = os.MkdirAll(trayConfigDir, 0755)
	if err != nil {
		t.Fatal(err)
	}

	customDir := "/custom/daylit/dir"
	settingsJSON := fmt.Sprintf(`{"settings": {"lockfile_dir": "%s"}}`, customDir)
	err = os.WriteFile(filepath.Join(trayConfigDir, "settings.json"), []byte(settingsJSON), 0644)
	if err != nil {
		t.Fatal(err)
	}

	dir, err = getTrayAppConfigDir()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if dir != customDir {
		t.Errorf("expected %s, got %s", customDir, dir)
	}
}

func TestFindAndValidateTrayProcess(t *testing.T) {
	// Mock findProcessFunc
	oldFindProcessFunc := findProcessFunc
	defer func() { findProcessFunc = oldFindProcessFunc }()

	// Setup temp lockfile
	tempDir, err := os.MkdirTemp("", "daylit-lock-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)
	lockfilePath := filepath.Join(tempDir, NotifierLockfileName)

	// Test 1: Lockfile missing
	_, err = findAndValidateTrayProcess(lockfilePath)
	if err == nil {
		t.Error("expected error for missing lockfile")
	}

	// Test 2: Malformed lockfile
	err = os.WriteFile(lockfilePath, []byte("invalid"), 0644)
	if err != nil {
		t.Fatal(err)
	}
	_, err = findAndValidateTrayProcess(lockfilePath)
	if err == nil {
		t.Error("expected error for malformed lockfile")
	}

	// Test 3: Process not running
	err = os.WriteFile(lockfilePath, []byte("8080|12345"), 0644)
	if err != nil {
		t.Fatal(err)
	}
	findProcessFunc = func(pid int) (ps.Process, error) {
		return nil, nil // Process not found
	}
	_, err = findAndValidateTrayProcess(lockfilePath)
	if err == nil {
		t.Error("expected error for missing process")
	}

	// Test 4: Wrong executable
	findProcessFunc = func(pid int) (ps.Process, error) {
		return &mockProcess{pid: pid, executable: "other-app"}, nil
	}
	_, err = findAndValidateTrayProcess(lockfilePath)
	if err == nil {
		t.Error("expected error for wrong executable")
	}

	// Test 5: Success
	findProcessFunc = func(pid int) (ps.Process, error) {
		return &mockProcess{pid: pid, executable: "daylit-tray"}, nil
	}
	port, err := findAndValidateTrayProcess(lockfilePath)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if port != "8080" {
		t.Errorf("expected port 8080, got %s", port)
	}
}

func TestSendNotification(t *testing.T) {
	// Setup mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		var payload WebhookPayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if payload.Text == "fail" {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Extract port
	parts := strings.Split(server.URL, ":")
	port := parts[len(parts)-1]

	// Test 1: Success
	err := sendNotification(port, WebhookPayload{Text: "hello"})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Test 2: Server error
	err = sendNotification(port, WebhookPayload{Text: "fail"})
	if err == nil {
		t.Error("expected error for server failure")
	}
}
