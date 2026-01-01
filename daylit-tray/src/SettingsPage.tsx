import { invoke } from "@tauri-apps/api/core";
import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import "./SettingsPage.css";

interface Settings {
  font_size: string;
  launch_at_login: boolean;
  lockfile_dir: string | null;
}

const SettingsPage = () => {
  const [settings, setSettings] = useState<Settings>({
    font_size: "medium",
    launch_at_login: false,
    lockfile_dir: null,
  });
  const [initialSettings, setInitialSettings] = useState<Settings>({
    font_size: "medium",
    launch_at_login: false,
    lockfile_dir: null,
  });
  const [status, setStatus] = useState<{
    type: "success" | "error";
    message: string;
  } | null>(null);

  useEffect(() => {
    loadSettings();
  }, []);

  const loadSettings = async () => {
    try {
      const loadedSettings = await invoke<Settings>("get_settings");
      setSettings(loadedSettings);
      setInitialSettings(loadedSettings);
    } catch (error) {
      console.error("Failed to load settings:", error);
      setStatus({ type: "error", message: "Failed to load settings" });
    }
  };

  const handleFontSizeChange = (e: React.ChangeEvent<HTMLSelectElement>) => {
    const newSize = e.target.value;
    setSettings((prev) => ({ ...prev, font_size: newSize }));
  };

  const handleLaunchAtLoginChange = (
    e: React.ChangeEvent<HTMLInputElement>,
  ) => {
    const enabled = e.target.checked;
    setSettings((prev) => ({ ...prev, launch_at_login: enabled }));
  };

  const handleDaylitDirChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const value = e.target.value;
    setSettings((prev) => ({
      ...prev,
      lockfile_dir: value === "" ? null : value,
    }));
  };

  const handleSave = async () => {
    try {
      await invoke("save_settings", { settings });
      setInitialSettings(settings);

      setStatus({ type: "success", message: "Settings saved successfully" });

      // Clear success message after 3 seconds
      setTimeout(() => setStatus(null), 3000);
    } catch (error) {
      console.error("Failed to save settings:", error);
      setStatus({ type: "error", message: "Failed to save settings" });
    }
  };

  const hasChanges =
    JSON.stringify(settings) !== JSON.stringify(initialSettings);

  return (
    <div className="settings-container">
      <div className="settings-header">
        <h1 className="settings-title">Settings</h1>
        <Link to="/" className="back-link">
          ← Back
        </Link>
      </div>

      <section className="settings-section">
        <h3 className="settings-section-title">Appearance</h3>
        <div className="setting-item">
          <label htmlFor="font-size" className="setting-label">
            Font Size
          </label>
          <select
            id="font-size"
            value={settings.font_size}
            onChange={handleFontSizeChange}
            className="setting-control"
          >
            <option value="small">Small</option>
            <option value="medium">Medium</option>
            <option value="large">Large</option>
          </select>
        </div>
      </section>

      <section className="settings-section">
        <h3 className="settings-section-title">Configuration</h3>
        <div className="setting-item">
          <label htmlFor="daylit-dir" className="setting-label">
            Daylit Directory
          </label>
          <input
            type="text"
            id="daylit-dir"
            value={settings.lockfile_dir || ""}
            onChange={handleDaylitDirChange}
            placeholder="Leave empty for default"
            className="setting-control"
          />
          <p className="setting-hint">
            Default: %APPDATA%/com.daylit.daylit-tray on Windows,
            $XDG_CONFIG_HOME/com.daylit.daylit-tray or
            ~/.config/com.daylit.daylit-tray on Linux, and ~/Library/Application
            Support/com.daylit.daylit-tray on macOS.
          </p>
        </div>
        <div className="setting-item">
          <label className="setting-checkbox-label">
            <input
              type="checkbox"
              checked={settings.launch_at_login}
              onChange={handleLaunchAtLoginChange}
              className="setting-checkbox"
            />
            <span className="setting-label">Launch at Login</span>
          </label>
        </div>
      </section>

      {status && (
        <div
          className={`status-message ${status.type === "success" ? "status-success" : "status-error"}`}
        >
          {status.message}
        </div>
      )}

      <button
        onClick={handleSave}
        className="save-button"
        disabled={!hasChanges}
      >
        Save
      </button>
    </div>
  );
};

export default SettingsPage;
