import { invoke } from "@tauri-apps/api/core";
import { listen } from "@tauri-apps/api/event";
import { useEffect, useRef, useState } from "react";
import "./NotificationPage.css";

interface WebhookPayload {
  text: string;
  duration_ms: number;
}

function NotificationPage() {
  const [notification, setNotification] = useState<WebhookPayload | null>(null);
  const timerRef = useRef<number | null>(null);

  const handleClose = () => {
    invoke("close_notification_window");
  };

  const setupNotification = (payload: WebhookPayload) => {
    if (timerRef.current) {
      clearTimeout(timerRef.current);
    }

    setNotification(payload);

    timerRef.current = setTimeout(
      handleClose,
      payload.duration_ms || 7000,
    ) as unknown as number;
  };

  useEffect(() => {
    const fetchPayload = async () => {
      try {
        const payload = await invoke<WebhookPayload>(
          "get_notification_payload",
        );
        if (payload) {
          setupNotification(payload);
        } else {
          handleClose();
        }
      } catch (e) {
        handleClose();
      }
    };
    fetchPayload();
  }, [handleClose, setupNotification]);

  useEffect(() => {
    let unlistenFn: (() => void) | null = null;
    let isMounted = true;

    const setupListener = async () => {
      try {
        const unlisten = await listen<WebhookPayload>("update_notification", (event) => {
          console.log("Received live update:", event.payload);
          setupNotification(event.payload);
        });

        if (isMounted) {
          unlistenFn = unlisten;
        } else {
          // Component unmounted before listener was set up, clean it up immediately
          unlisten();
        }
      } catch (error) {
        console.error("Failed to set up notification listener:", error);
      }
    };

    setupListener();

    return () => {
      isMounted = false;
      if (unlistenFn) {
        unlistenFn();
      }
    };
  }, []);

  if (!notification) {
    return null;
  }

  return (
    <div className="notification-bar" onClick={handleClose}>
      <p className="notification-text">{notification.text}</p>
    </div>
  );
}

export default NotificationPage;
