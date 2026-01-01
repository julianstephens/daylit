import { invoke } from "@tauri-apps/api/core";
import { listen } from "@tauri-apps/api/event";
import { useEffect } from "react";

interface Settings {
  font_size: string;
}

export const FontSizeManager = () => {
  useEffect(() => {
    const applyFontSize = (size: string) => {
      let fontSizePx = "16px"; // medium (default)
      if (size === "small") fontSizePx = "14px";
      if (size === "large") fontSizePx = "20px";

      document.documentElement.style.fontSize = fontSizePx;
    };

    const loadSettings = async () => {
      try {
        const settings = await invoke<Settings>("get_settings");
        if (settings && settings.font_size) {
          applyFontSize(settings.font_size);
        }
      } catch (e) {
        console.error("Failed to load settings for font size:", e);
      }
    };

    loadSettings();

    let unlistenFn: (() => void) | null = null;

    const setupListener = async () => {
      unlistenFn = await listen<Settings>("settings-updated", (event) => {
        if (event.payload && event.payload.font_size) {
          applyFontSize(event.payload.font_size);
        }
      });
    };

    setupListener();

    return () => {
      if (unlistenFn) {
        unlistenFn();
      }
    };
  }, []);

  return null;
};
