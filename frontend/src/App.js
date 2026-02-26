import React, { useState, useMemo } from 'react';
import { BrowserRouter, Routes, Route } from 'react-router-dom';
import { ThemeProvider } from '@mui/material/styles';
import CssBaseline from '@mui/material/CssBaseline';
import { getTheme } from './theme';
import Layout from './components/Layout';
import Dashboard from './pages/Dashboard';
import Devices from './pages/Devices';
import DeviceDetails from './pages/DeviceDetails';
import DeviceDiff from './pages/DeviceDiff';
import Search from './pages/Search';
import SearchChanges from './pages/SearchChanges';

function App() {
  const [darkMode, setDarkMode] = useState(false);
  const theme = useMemo(() => getTheme(darkMode ? 'dark' : 'light'), [darkMode]);

  return (
    <ThemeProvider theme={theme}>
      <CssBaseline />
      <BrowserRouter>
        <Layout darkMode={darkMode} setDarkMode={setDarkMode}>
          <Routes>
            <Route path="/" element={<Dashboard />} />
            <Route path="/devices" element={<Devices />} />
            <Route path="/devices/:id" element={<DeviceDetails />} />
            <Route path="/diff" element={<DeviceDiff />} />
            <Route path="/search" element={<Search />} />
            <Route path="/search-changes" element={<SearchChanges />} />
          </Routes>
        </Layout>
      </BrowserRouter>
    </ThemeProvider>
  );
}

export default App;
