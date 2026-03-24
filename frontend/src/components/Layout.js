import React, { useState, useEffect, useRef } from 'react';
import { useNavigate, useLocation } from 'react-router-dom';
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
];

export const ThemeToggleContext = React.createContext();

const Layout = ({ children, darkMode, setDarkMode }) => {
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

  const handleMenuClose = () => {
    setUserMenuOpen(false);
  };

  const handleNavigation = (path) => {
    navigate(path);
    if (isMobile) {
      setMobileOpen(false);
    }
  };

  const handleThemeToggle = () => {
    setDarkMode(!darkMode);
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
      <Box sx={{ p: 2, borderTop: 1, borderColor: 'divider' }}>
        <ListItemButton sx={{ borderRadius: 2 }}>
          <ListItemIcon sx={{ minWidth: 40 }}>
            <SettingsIcon />
          </ListItemIcon>
          <ListItemText primary="Настройки" />
        </ListItemButton>
      </Box>
    </Box>
  );

  return (
    <ThemeToggleContext.Provider value={{ darkMode, setDarkMode }}>
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
                  <ListItemButton onClick={handleMenuClose} dense>
                    <ListItemIcon><SettingsIcon fontSize="small" /></ListItemIcon>
                    <ListItemText primary="Профиль" primaryTypographyProps={{ variant: 'body2' }} />
                  </ListItemButton>
                  <ListItemButton onClick={handleMenuClose} dense>
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
          {children}
        </Box>
      </Box>
    </ThemeToggleContext.Provider>
  );
};

export default Layout;
