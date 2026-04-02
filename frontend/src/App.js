import React, { useState, useMemo, useEffect, useCallback } from 'react';
import { BrowserRouter, Routes, Route } from 'react-router-dom';
import { ThemeProvider } from '@mui/material/styles';
import CssBaseline from '@mui/material/CssBaseline';
import { getTheme } from './theme';
import Layout from './components/Layout';
import ProtectedRoute from './components/ProtectedRoute';
import Dashboard from './pages/Dashboard';
import Devices from './pages/Devices';
import DeviceDetails from './pages/DeviceDetails';
import DeviceDiff from './pages/DeviceDiff';
import Search from './pages/Search';
import SearchChanges from './pages/SearchChanges';
import Settings from './pages/Settings';
import LoginPage from './pages/LoginPage';

const SETTINGS_KEY = 'app_settings';
const SETTINGS_VERSION = 1;

const defaultSettings = {
  darkMode: false,
  scale: 0.8,
  accentColor: '#2563eb',
  _version: SETTINGS_VERSION,
};

const loadSettings = () => {
  try {
    const stored = localStorage.getItem(SETTINGS_KEY);
    if (stored) {
      const parsed = JSON.parse(stored);
      return { ...defaultSettings, ...parsed };
    }
  } catch (e) {
    console.warn('Failed to load settings from localStorage:', e);
  }
  return defaultSettings;
};

const saveSettings = (settings) => {
  try {
    localStorage.setItem(SETTINGS_KEY, JSON.stringify({ ...settings, _version: SETTINGS_VERSION }));
  } catch (e) {
    console.warn('Failed to save settings to localStorage:', e);
  }
};

function App() {
  const [settings, setSettings] = useState(loadSettings);

  useEffect(() => {
    document.documentElement.style.setProperty('--app-scale', settings.scale);
  }, [settings.scale]);

  const handleSettingsChange = useCallback((newSettings) => {
    setSettings(newSettings);
    saveSettings(newSettings);
  }, []);

  const theme = useMemo(
    () => getTheme(settings.darkMode ? 'dark' : 'light', settings.accentColor),
    [settings.darkMode, settings.accentColor]
  );

  return (
    <ThemeProvider theme={theme}>
      <CssBaseline />
      <BrowserRouter>
        <Routes>
          <Route path="/login" element={<LoginPage />} />
          <Route element={<ProtectedRoute />}>
            <Route element={<Layout settings={settings} onSettingsChange={handleSettingsChange} />}>
              <Route index element={<Dashboard />} />
              <Route path="/devices" element={<Devices />} />
              <Route path="/devices/:id" element={<DeviceDetails />} />
              <Route path="/diff" element={<DeviceDiff />} />
              <Route path="/search" element={<Search />} />
              <Route path="/search-changes" element={<SearchChanges />} />
              <Route path="/settings" element={<Settings />} />
            </Route>
          </Route>
        </Routes>
      </BrowserRouter>
    </ThemeProvider>
  );
}

export default App;
