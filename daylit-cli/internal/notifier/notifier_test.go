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

	"github.com/julianstephens/daylit/daylit-cli/internal/constants"
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
	expectedDefault := filepath.Join(tempDir, constants.TrayAppIdentifier)
	dir, err := GetTrayAppConfigDir()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if dir != expectedDefault {
		t.Errorf("expected %s, got %s", expectedDefault, dir)
	}

	// Test 2: Custom setting
	trayConfigDir := filepath.Join(tempDir, constants.TrayAppIdentifier)
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

	dir, err = GetTrayAppConfigDir()
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
	lockfilePath := filepath.Join(tempDir, constants.NotifierLockfileName)

	// Test 1: Lockfile missing
	_, _, err = findAndValidateTrayProcess(lockfilePath)
	if err == nil {
		t.Error("expected error for missing lockfile")
	}

	// Test 2: Malformed lockfile (old 2-part format)
	err = os.WriteFile(lockfilePath, []byte("8080|12345"), 0644)
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = findAndValidateTrayProcess(lockfilePath)
	if err == nil {
		t.Error("expected error for malformed lockfile")
	}

	// Test 3: Malformed lockfile (invalid format)
	err = os.WriteFile(lockfilePath, []byte("invalid"), 0644)
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = findAndValidateTrayProcess(lockfilePath)
	if err == nil {
		t.Error("expected error for malformed lockfile")
	}

	// Test 4: Empty secret
	err = os.WriteFile(lockfilePath, []byte("8080|12345|"), 0644)
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = findAndValidateTrayProcess(lockfilePath)
	if err == nil {
		t.Error("expected error for empty secret")
	}
	if err != nil && !strings.Contains(err.Error(), "secret") {
		t.Errorf("expected error about empty secret, got: %v", err)
	}

	// Test 5: Invalid port (empty)
	err = os.WriteFile(lockfilePath, []byte("|12345|testsecret123"), 0644)
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = findAndValidateTrayProcess(lockfilePath)
	if err == nil {
		t.Error("expected error for empty port")
	}

	// Test 6: Invalid port (out of range)
	err = os.WriteFile(lockfilePath, []byte("99999|12345|testsecret123"), 0644)
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = findAndValidateTrayProcess(lockfilePath)
	if err == nil {
		t.Error("expected error for port out of range")
	}

	// Test 7: Process not running
	err = os.WriteFile(lockfilePath, []byte("8080|12345|testsecret123"), 0644)
	if err != nil {
		t.Fatal(err)
	}
	findProcessFunc = func(pid int) (ps.Process, error) {
		return nil, nil // Process not found
	}
	_, _, err = findAndValidateTrayProcess(lockfilePath)
	if err == nil {
		t.Error("expected error for missing process")
	}

	// Test 8: Wrong executable
	findProcessFunc = func(pid int) (ps.Process, error) {
		return &mockProcess{pid: pid, executable: "other-app"}, nil
	}
	_, _, err = findAndValidateTrayProcess(lockfilePath)
	if err == nil {
		t.Error("expected error for wrong executable")
	}

	// Test 9: Success
	findProcessFunc = func(pid int) (ps.Process, error) {
		return &mockProcess{pid: pid, executable: "daylit-tray"}, nil
	}
	port, secret, err := findAndValidateTrayProcess(lockfilePath)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if port != "8080" {
		t.Errorf("expected port 8080, got %s", port)
	}
	if secret != "testsecret123" {
		t.Errorf("expected secret testsecret123, got %s", secret)
	}
}

func TestSendNotification(t *testing.T) {
	// Setup mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		// Check for secret header
		secret := r.Header.Get("X-Daylit-Secret")
		if secret == "" {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("Unauthorized"))
			return
		}
		if secret != "test-secret" {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("Unauthorized"))
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
	err := sendNotification(port, "test-secret", WebhookPayload{Text: "hello"})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Test 2: Missing secret
	err = sendNotification(port, "", WebhookPayload{Text: "hello"})
	if err == nil {
		t.Error("expected error for missing secret")
	}

	// Test 3: Wrong secret
	err = sendNotification(port, "wrong-secret", WebhookPayload{Text: "hello"})
	if err == nil {
		t.Error("expected error for wrong secret")
	}

	// Test 4: Server error
	err = sendNotification(port, "test-secret", WebhookPayload{Text: "fail"})
	if err == nil {
		t.Error("expected error for server failure")
	}
}
