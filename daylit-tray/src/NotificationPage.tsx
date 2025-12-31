import { invoke } from '@tauri-apps/api/core';
import { listen } from '@tauri-apps/api/event';
import { useEffect, useRef, useState } from 'react';
import './NotificationPage.css';

interface WebhookPayload {
    text: string;
    duration_ms: number;
}

function NotificationPage() {
    const [notification, setNotification] = useState<WebhookPayload | null>(null);
    const timerRef = useRef<number | null>(null);

    const handleClose = () => {
        invoke('close_notification_window');
    };

    const setupNotification = (payload: WebhookPayload) => {
        if (timerRef.current) {
            clearTimeout(timerRef.current);
        }

        setNotification(payload);

        timerRef.current = setTimeout(handleClose, payload.duration_ms || 7000) as unknown as number;
    };

    useEffect(() => {
        const fetchPayload = async () => {
            try {
                const payload = await invoke<WebhookPayload>('get_notification_payload');
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
    }, []);

    useEffect(() => {
        const unlisten = listen<WebhookPayload>('update_notification', (event) => {
            console.log('Received live update:', event.payload);
            setupNotification(event.payload);
        });

        return () => {
            unlisten.then(f => f());
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