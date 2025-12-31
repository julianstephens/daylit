import { getVersion } from '@tauri-apps/api/app';
import { useEffect, useState } from 'react';
import './App.css';

function App() {
  const [appVersion, setAppVersion] = useState('');

  useEffect(() => {
    getVersion().then(version => {
      setAppVersion(version);
    });
  }, []);

  return (
    <div className="container">
      <div className="app-info">
        <h1 className="app-title">Daylit Tray</h1>
        <p className="app-description">
          A lightweight background application that listens for webhook events and displays native desktop notifications.
        </p>
        {appVersion && (
          <p className="app-version">
            Version: {appVersion}
          </p>
        )}
      </div>
    </div>
  );
}

export default App;