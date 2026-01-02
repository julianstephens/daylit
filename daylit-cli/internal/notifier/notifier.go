package notifier

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/mitchellh/go-ps"

	"github.com/julianstephens/daylit/daylit-cli/internal/constants"
)

var (
	userConfigDirFunc = os.UserConfigDir
	findProcessFunc   = ps.FindProcess
)

type Notifier struct{}

type WebhookPayload struct {
	Text       string `json:"text"`
	DurationMs uint32 `json:"duration_ms"`
}

func New() *Notifier {
	return &Notifier{}
}

func (n *Notifier) Notify(text string) error {
	trayAppConfigPath, err := GetTrayAppConfigDir()
	if err != nil {
		return err
	}

	port, secret, err := findAndValidateTrayProcess(filepath.Join(trayAppConfigPath, constants.NotifierLockfileName))
	if err != nil {
		return err
	}

	payload := WebhookPayload{
		Text:       text,
		DurationMs: constants.NotificationDurationMs,
	}

	if err := sendNotification(port, secret, payload); err != nil {
		return err
	}

	return nil
}

// GetTrayAppConfigDir returns the configuration directory used by the tray application.
func GetTrayAppConfigDir() (string, error) {
	configDir, err := userConfigDirFunc()
	if err != nil {
		return "", fmt.Errorf("failed to get user config dir: %w", err)
	}

	trayConfigDir := filepath.Join(configDir, constants.TrayAppIdentifier)

	// Check for settings.json to see if a custom lockfile dir is set
	settingsPath := filepath.Join(trayConfigDir, "settings.json")
	if _, err := os.Stat(settingsPath); err == nil {
		data, err := os.ReadFile(settingsPath)
		if err == nil {
			var store struct {
				Settings struct {
					LockfileDir *string `json:"lockfile_dir"`
				} `json:"settings"`
			}
			if err := json.Unmarshal(data, &store); err == nil {
				if store.Settings.LockfileDir != nil && *store.Settings.LockfileDir != "" {
					return *store.Settings.LockfileDir, nil
				}
			}
		}
	}

	return trayConfigDir, nil
}

func findAndValidateTrayProcess(lockfilePath string) (string, string, error) {
	content, err := os.ReadFile(lockfilePath)
	if err != nil {
		return "", "", errors.New("daylit-tray is not running")
	}

	parts := strings.Split(strings.TrimSpace(string(content)), "|")
	if len(parts) != 3 {
		return "", "", errors.New("lockfile is malformed")
	}

	port := parts[0]
	if strings.TrimSpace(port) == "" {
		return "", "", errors.New("port in lockfile is empty")
	}
	// Validate port is a valid number in the valid TCP range (1-65535)
	portNum, err := strconv.Atoi(port)
	if err != nil {
		return "", "", errors.New("invalid port number in lockfile")
	}
	if portNum < 1 || portNum > 65535 {
		return "", "", fmt.Errorf("port number %d is outside valid range (1-65535)", portNum)
	}

	pid, err := strconv.Atoi(parts[1])
	if err != nil {
		return "", "", errors.New("invalid process ID in lockfile")
	}
	secret := parts[2]
	if strings.TrimSpace(secret) == "" {
		return "", "", errors.New("secret in lockfile is empty")
	}

	process, err := findProcessFunc(pid)
	if err != nil || process == nil {
		return "", "", errors.New("daylit-tray process not running")
	}

	if !strings.HasPrefix(process.Executable(), "daylit-tray") {
		return "", "", fmt.Errorf("process with PID %d is not daylit-tray (is %s)", pid, process.Executable())
	}

	return port, secret, nil
}

func sendNotification(port string, secret string, payload WebhookPayload) error {
	url := fmt.Sprintf("http://127.0.0.1:%s", port)

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Daylit-Secret", secret)

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode == http.StatusOK {
		return nil
	}

	body, _ := io.ReadAll(res.Body)
	return fmt.Errorf("notification failed with status %d: %s", res.StatusCode, string(body))
}
