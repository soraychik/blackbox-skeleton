import React, { useMemo, useEffect } from 'react';
import { BrowserRouter, Routes, Route, useNavigate } from 'react-router-dom';
import { ThemeProvider } from '@mui/material/styles';
import CssBaseline from '@mui/material/CssBaseline';
import { getTheme } from './theme';
import { useSettings } from './utils/useSettings';
import { setNavigate } from './utils/api';
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

const NavigateSetter = () => {
  const navigate = useNavigate();
  useEffect(() => {
    setNavigate(navigate);
  }, [navigate]);
  return null;
};

function App() {
  const [settings, updateSettings] = useSettings();

  useEffect(() => {
    document.documentElement.style.setProperty('--app-scale', settings.scale);
  }, [settings.scale]);

  const theme = useMemo(
    () => getTheme(settings.darkMode ? 'dark' : 'light', settings.accentColor),
    [settings.darkMode, settings.accentColor]
  );

  return (
    <ThemeProvider theme={theme}>
      <CssBaseline />
      <BrowserRouter>
        <NavigateSetter />
        <Routes>
          <Route path="/login" element={<LoginPage />} />
          <Route element={<ProtectedRoute />}>
            <Route element={<Layout settings={settings} onSettingsChange={updateSettings} />}>
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
