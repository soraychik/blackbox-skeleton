import { useState, useCallback } from 'react';

const SETTINGS_KEY = 'app_settings';
const SETTINGS_VERSION = 1;

const defaultSettings = {
  darkMode: false,
  scale: 0.8,
  accentColor: '#2563eb',
  _version: SETTINGS_VERSION,
};

export const useSettings = () => {
  const [settings, setSettings] = useState(() => {
    try {
      const stored = localStorage.getItem(SETTINGS_KEY);
      return stored ? { ...defaultSettings, ...JSON.parse(stored) } : defaultSettings;
    } catch {
      return defaultSettings;
    }
  });

  const updateSettings = useCallback((newSettings) => {
    setSettings(newSettings);
    try {
      localStorage.setItem(SETTINGS_KEY, JSON.stringify({
        ...newSettings,
        _version: SETTINGS_VERSION,
      }));
    } catch (e) {
      console.warn('Failed to save settings:', e);
    }
  }, []);

  return [settings, updateSettings];
};
