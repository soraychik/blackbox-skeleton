import React, { useState, useEffect, useRef } from 'react';
import { useNavigate, useLocation, Outlet } from 'react-router-dom';
import {
  AppBar,
  Box,
  CssBaseline,
  Drawer,
  IconButton,
  List,
  ListItem,
  ListItemButton,
  ListItemIcon,
  ListItemText,
  Toolbar,
  Typography,
  Avatar,
  Paper,
  useTheme,
  useMediaQuery,
} from '@mui/material';
import {
  Menu as MenuIcon,
  Dashboard as DashboardIcon,
  Devices as DevicesIcon,
  Compare as CompareIcon,
  Search as SearchIcon,
  FilterList as FilterListIcon,
  DarkMode as DarkModeIcon,
  LightMode as LightModeIcon,
  Logout as LogoutIcon,
  Settings as SettingsIcon,
} from '@mui/icons-material';

const drawerWidth = 260;

const menuItems = [
  { text: 'Дашборд', icon: <DashboardIcon />, path: '/' },
  { text: 'Каталог устройств', icon: <DevicesIcon />, path: '/devices' },
  { text: 'Сравнение устройств', icon: <CompareIcon />, path: '/diff' },
  { text: 'Поиск', icon: <SearchIcon />, path: '/search' },
  { text: 'Поиск по изменениям', icon: <FilterListIcon />, path: '/search-changes' },
  { text: 'Настройки', icon: <SettingsIcon />, path: '/settings' },
];

const SETTINGS_KEY = 'app_settings';
const SETTINGS_VERSION = 1;

const defaultSettings = {
  darkMode: false,
  scale: 1.0,
  accentColor: '#2563eb',
  _version: SETTINGS_VERSION,
};

export const SettingsContext = React.createContext();

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

const Layout = ({ settings: externalSettings, onSettingsChange }) => {
  const [internalSettings, setInternalSettings] = React.useState(loadSettings);
  const settings = externalSettings || internalSettings;
  const setSettings = React.useCallback((updater) => {
    const newSettings = typeof updater === 'function' ? updater(settings) : updater;
    if (externalSettings && onSettingsChange) {
      onSettingsChange(newSettings);
    } else {
      setInternalSettings(newSettings);
    }
    saveSettings(newSettings);
  }, [settings, externalSettings, onSettingsChange]);

  const darkMode = settings.darkMode;
  const setDarkMode = React.useCallback((value) => {
    setSettings((prev) => ({ ...prev, darkMode: typeof value === 'function' ? value(prev.darkMode) : value }));
  }, [setSettings]);

  const scale = settings.scale;
  const setScale = React.useCallback((value) => {
    setSettings((prev) => ({ ...prev, scale: typeof value === 'function' ? value(prev.scale) : value }));
  }, [setSettings]);

  const accentColor = settings.accentColor;
  const setAccentColor = React.useCallback((value) => {
    setSettings((prev) => ({ ...prev, accentColor: value }));
  }, [setSettings]);
  const theme = useTheme();
  const isMobile = useMediaQuery(theme.breakpoints.down('md'));
  const [mobileOpen, setMobileOpen] = useState(false);
  const [userMenuOpen, setUserMenuOpen] = useState(false);
  const userMenuRef = useRef(null);

  const navigate = useNavigate();
  const location = useLocation();

  useEffect(() => {
    if (!userMenuOpen) return;
    const handleClickOutside = (e) => {
      if (userMenuRef.current && !userMenuRef.current.contains(e.target)) {
        setUserMenuOpen(false);
      }
    };
    document.addEventListener('click', handleClickOutside);
    return () => document.removeEventListener('click', handleClickOutside);
  }, [userMenuOpen]);

  const handleDrawerToggle = () => {
    setMobileOpen(!mobileOpen);
  };

  const handleMenuToggle = (e) => {
    e.stopPropagation();
    setUserMenuOpen((prev) => !prev);
  };

  const handleNavigation = (path) => {
    navigate(path);
    if (isMobile) {
      setMobileOpen(false);
    }
  };

  const handleThemeToggle = () => {
    setDarkMode((prev) => !prev);
  };

  const handleLogout = () => {
    localStorage.removeItem('token');
    localStorage.removeItem('user');
    navigate('/login');
  };

  const handleProfile = () => {
    setUserMenuOpen(false);
    navigate('/settings');
  };

  const drawer = (
    <Box sx={{ display: 'flex', flexDirection: 'column', height: '100%' }}>
      <Toolbar>
        <Typography variant="h6" noWrap component="div" sx={{ fontWeight: 700, color: 'primary.main' }}>
          Black Box
        </Typography>
      </Toolbar>
      <List sx={{ px: 2, flexGrow: 1 }}>
        {menuItems.map((item) => (
          <ListItem key={item.text} disablePadding sx={{ mb: 0.5 }}>
            <ListItemButton
              selected={location.pathname === item.path}
              onClick={() => handleNavigation(item.path)}
              sx={{
                borderRadius: 2,
                '&.Mui-selected': {
                  backgroundColor: 'primary.main',
                  color: 'white',
                  '&:hover': {
                    backgroundColor: 'primary.dark',
                  },
                  '& .MuiListItemIcon-root': {
                    color: 'white',
                  },
                },
              }}
            >
              <ListItemIcon sx={{ minWidth: 40, color: location.pathname === item.path ? 'white' : 'text.secondary' }}>
                {item.icon}
              </ListItemIcon>
              <ListItemText primary={item.text} />
            </ListItemButton>
          </ListItem>
        ))}
      </List>
    </Box>
  );

  return (
    <SettingsContext.Provider value={{ darkMode, setDarkMode, scale, setScale, accentColor, setAccentColor }}>
      <Box sx={{ display: 'flex' }}>
        <CssBaseline />
        <AppBar
          position="fixed"
          sx={{
            width: { md: `calc(100% - ${drawerWidth}px)` },
            ml: { md: `${drawerWidth}px` },
            boxShadow: 'none',
            borderBottom: 1,
            borderColor: 'divider',
            bgcolor: 'background.paper',
            color: 'text.primary',
          }}
        >
          <Toolbar>
            <IconButton
              color="inherit"
              edge="start"
              onClick={handleDrawerToggle}
              sx={{ mr: 2, display: { md: 'none' } }}
            >
              <MenuIcon />
            </IconButton>
            <Box sx={{ flexGrow: 1 }} />
            <IconButton onClick={handleThemeToggle} sx={{ mr: 1 }}>
              {darkMode ? <LightModeIcon /> : <DarkModeIcon />}
            </IconButton>
            <Box ref={userMenuRef} sx={{ position: 'relative' }}>
              <IconButton onClick={handleMenuToggle} aria-haspopup="true" aria-expanded={userMenuOpen}>
                <Avatar sx={{ width: 32, height: 32, bgcolor: 'primary.main' }}>U</Avatar>
              </IconButton>
              {userMenuOpen && (
                <Paper
                  elevation={8}
                  sx={{
                    position: 'absolute',
                    right: 0,
                    top: '100%',
                    mt: 1,
                    minWidth: 160,
                    py: 0.5,
                    zIndex: (t) => t.zIndex.tooltip + 1,
                    '& .MuiListItemIcon-root': { minWidth: 36 },
                  }}
                >
                  <ListItemButton onClick={handleProfile} dense>
                    <ListItemIcon><SettingsIcon fontSize="small" /></ListItemIcon>
                    <ListItemText primary="Профиль" primaryTypographyProps={{ variant: 'body2' }} />
                  </ListItemButton>
                  <ListItemButton onClick={handleLogout} dense>
                    <ListItemIcon><LogoutIcon fontSize="small" /></ListItemIcon>
                    <ListItemText primary="Выйти" primaryTypographyProps={{ variant: 'body2' }} />
                  </ListItemButton>
                </Paper>
              )}
            </Box>
          </Toolbar>
        </AppBar>
        <Box
          component="nav"
          sx={{ width: { md: drawerWidth }, flexShrink: { md: 0 } }}
        >
          <Drawer
            variant="temporary"
            open={mobileOpen}
            onClose={handleDrawerToggle}
            ModalProps={{ keepMounted: true }}
            sx={{
              display: { xs: 'block', md: 'none' },
              '& .MuiDrawer-paper': { boxSizing: 'border-box', width: drawerWidth },
            }}
          >
            {drawer}
          </Drawer>
          <Drawer
            variant="permanent"
            sx={{
              display: { xs: 'none', md: 'block' },
              '& .MuiDrawer-paper': { boxSizing: 'border-box', width: drawerWidth, borderRight: 1, borderColor: 'divider' },
            }}
            open
          >
            {drawer}
          </Drawer>
        </Box>
        <Box
          component="main"
          sx={{
            flexGrow: 1,
            p: 3,
            width: { md: `calc(100% - ${drawerWidth}px)` },
            minHeight: '100vh',
            bgcolor: 'background.default',
          }}
        >
          <Toolbar />
          <Outlet />
        </Box>
      </Box>
    </SettingsContext.Provider>
  );
};

export default Layout;
