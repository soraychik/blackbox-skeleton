import React, { useState, useCallback, useMemo, useEffect } from 'react';
import { useNavigate, useLocation, Outlet } from 'react-router-dom';
import {
  AppBar,
  Avatar,
  Box,
  Chip,
  CssBaseline,
  Drawer,
  IconButton,
  List,
  ListItem,
  ListItemButton,
  ListItemIcon,
  ListItemText,
  Menu,
  MenuItem,
  Toolbar,
  Tooltip,
  Typography,
  useMediaQuery,
  useTheme,
} from '@mui/material';
import { useRole } from '../utils/useRole';
import { useAuth } from '../utils/useAuth';
import {
  Compare as CompareIcon,
  Dashboard as DashboardIcon,
  DarkMode as DarkModeIcon,
  Devices as DevicesIcon,
  FilterList as FilterListIcon,
  History as HistoryIcon,
  LightMode as LightModeIcon,
  Logout as LogoutIcon,
  Menu as MenuIcon,
  MenuOpen as MenuOpenIcon,
  Search as SearchIcon,
  Settings as SettingsIcon,
} from '@mui/icons-material';

const DRAWER_WIDTH = 260;
const DRAWER_WIDTH_COLLAPSED = 72;

const ALL_MENU_ITEMS = [
  { text: 'Дашборд',             icon: <DashboardIcon />,  path: '/',               roles: ['admin', 'engineer', 'operator'] },
  { text: 'Каталог устройств',   icon: <DevicesIcon />,    path: '/devices',         roles: ['admin', 'engineer', 'operator'] },
  { text: 'Сравнение устройств', icon: <CompareIcon />,    path: '/diff',            roles: ['admin', 'engineer'] },
  { text: 'Поиск',               icon: <SearchIcon />,     path: '/search',          roles: ['admin', 'engineer'] },
  { text: 'Поиск по изменениям', icon: <FilterListIcon />, path: '/search-changes',  roles: ['admin', 'engineer'] },
  { text: 'Настройки',           icon: <SettingsIcon />,   path: '/settings',        roles: ['admin'] },
  { text: 'Журнал аудита',       icon: <HistoryIcon />,    path: '/audit',           roles: ['admin'] },
];

export const SettingsContext = React.createContext();

// ---------- SidebarContent ----------

// MUI Mini Drawer pattern: текст всегда в DOM, анимация через CSS.
// Drawer сужается, overflowX:hidden клипует содержимое — никакого условного рендера.
const SidebarContent = ({ collapsed, onToggle, isMobile, role, location, onNavigate }) => {
  const theme = useTheme();

  const menuItems = useMemo(
    () => ALL_MENU_ITEMS.filter(item => item.roles.includes(role)),
    [role],
  );

  const easing = theme.transitions.easing.sharp;
  const duration = theme.transitions.duration.enteringScreen;
  const transition = prop => theme.transitions.create(prop, { easing, duration });

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', height: '100%' }}>
      <Box
        sx={{
          display: 'flex',
          alignItems: 'center',
          px: 1.5,
          py: 1.5,
          minHeight: 56,
          overflow: 'hidden',
        }}
      >
        <Typography
          variant="h6"
          fontWeight={700}
          color="primary.main"
          noWrap
          sx={{
            flexGrow: 1,
            opacity: collapsed ? 0 : 1,
            transition: transition('opacity'),
            pl: 1,
          }}
        >
          Black Box
        </Typography>
        {!isMobile && (
          <IconButton
            onClick={onToggle}
            size="small"
            sx={{
              flexShrink: 0,
              transform: collapsed ? 'rotate(180deg)' : 'none',
              transition: transition('transform'),
            }}
          >
            <MenuOpenIcon />
          </IconButton>
        )}
      </Box>

      <List sx={{ px: 1, flexGrow: 1 }}>
        {menuItems.map(item => {
          const selected = location.pathname === item.path;
          return (
            <ListItem key={item.path} disablePadding sx={{ mb: 0.5 }}>
              <Tooltip title={collapsed ? item.text : ''} placement="right" arrow>
                <ListItemButton
                  selected={selected}
                  onClick={() => onNavigate(item.path)}
                  sx={{
                    borderRadius: 2,
                    minHeight: 48,
                    px: 2,
                    '&.Mui-selected': {
                      bgcolor: 'primary.main',
                      color: 'white',
                      '&:hover': { bgcolor: 'primary.dark' },
                      '& .MuiListItemIcon-root': { color: 'white' },
                    },
                  }}
                >
                  <ListItemIcon
                    sx={{
                      minWidth: 0,
                      mr: 2,
                      justifyContent: 'center',
                      color: selected ? 'white' : 'text.secondary',
                    }}
                  >
                    {item.icon}
                  </ListItemIcon>
                  <ListItemText
                    primary={item.text}
                    primaryTypographyProps={{ noWrap: true }}
                    sx={{
                      opacity: collapsed ? 0 : 1,
                      transition: transition('opacity'),
                      overflow: 'hidden',
                    }}
                  />
                </ListItemButton>
              </Tooltip>
            </ListItem>
          );
        })}
      </List>
    </Box>
  );
};

// ---------- UserMenu ----------

const UserMenu = ({ login, label, role, onLogout, onSettings }) => {
  const [anchor, setAnchor] = useState(null);
  const open = Boolean(anchor);

  const handleClose = () => setAnchor(null);

  return (
    <>
      <IconButton
        onClick={e => setAnchor(e.currentTarget)}
        aria-haspopup="true"
        aria-expanded={open}
      >
        <Avatar sx={{ width: 32, height: 32, bgcolor: 'primary.main' }}>
          {login ? login[0].toUpperCase() : 'U'}
        </Avatar>
      </IconButton>

      <Menu
        anchorEl={anchor}
        open={open}
        onClose={handleClose}
        transformOrigin={{ horizontal: 'right', vertical: 'top' }}
        anchorOrigin={{ horizontal: 'right', vertical: 'bottom' }}
        slotProps={{ paper: { sx: { minWidth: 200, mt: 1 } } }}
      >
        <Box sx={{ px: 2, py: 1.5, borderBottom: 1, borderColor: 'divider' }}>
          <Typography variant="body2" fontWeight={600}>{login}</Typography>
          <Chip
            label={label}
            size="small"
            color="primary"
            variant="outlined"
            sx={{ mt: 0.5, height: 20, fontSize: 11 }}
          />
        </Box>

        {role === 'admin' && (
          <MenuItem
            dense
            onClick={() => { handleClose(); onSettings(); }}
          >
            <ListItemIcon><SettingsIcon fontSize="small" /></ListItemIcon>
            <ListItemText primaryTypographyProps={{ variant: 'body2' }}>Настройки</ListItemText>
          </MenuItem>
        )}

        <MenuItem dense onClick={() => { handleClose(); onLogout(); }}>
          <ListItemIcon><LogoutIcon fontSize="small" /></ListItemIcon>
          <ListItemText primaryTypographyProps={{ variant: 'body2' }}>Выйти</ListItemText>
        </MenuItem>
      </Menu>
    </>
  );
};

// ---------- Layout ----------

const Layout = ({ settings, onSettingsChange }) => {
  const { role, login, label } = useRole();
  const { clearSession } = useAuth();
  const theme = useTheme();
  const isMobile = useMediaQuery(theme.breakpoints.down('md'));

  const [mobileOpen, setMobileOpen] = useState(false);
  const [collapsed, setCollapsed] = useState(
    () => localStorage.getItem('sidebarCollapsed') === 'true',
  );

  const navigate = useNavigate();
  const location = useLocation();

  const { darkMode, scale, accentColor } = settings;

  const setDarkMode = useCallback(
    v => onSettingsChange(p => ({ ...p, darkMode: typeof v === 'function' ? v(p.darkMode) : v })),
    [onSettingsChange],
  );
  const setScale = useCallback(
    v => onSettingsChange(p => ({ ...p, scale: typeof v === 'function' ? v(p.scale) : v })),
    [onSettingsChange],
  );
  const setAccentColor = useCallback(
    v => onSettingsChange(p => ({ ...p, accentColor: v })),
    [onSettingsChange],
  );

  useEffect(() => {
    localStorage.setItem('sidebarCollapsed', collapsed);
  }, [collapsed]);

  // На мобайле sidebar всегда раскрыт
  const drawerWidth = !isMobile && collapsed ? DRAWER_WIDTH_COLLAPSED : DRAWER_WIDTH;

  const sidebarTransition = theme.transitions.create(['width', 'margin'], {
    easing: theme.transitions.easing.sharp,
    duration: theme.transitions.duration.enteringScreen,
  });

  const handleNavigate = useCallback(
    path => {
      navigate(path);
      if (isMobile) setMobileOpen(false);
    },
    [navigate, isMobile],
  );

  const handleLogout = useCallback(() => {
    clearSession();
    navigate('/login');
  }, [clearSession, navigate]);

  const sidebarContent = (
    <SidebarContent
      collapsed={!isMobile && collapsed}
      onToggle={() => setCollapsed(p => !p)}
      isMobile={isMobile}
      role={role}
      location={location}
      onNavigate={handleNavigate}
    />
  );

  return (
    <SettingsContext.Provider value={{ darkMode, setDarkMode, scale, setScale, accentColor, setAccentColor }}>
      <Box sx={{ display: 'flex' }}>
        <CssBaseline />

        <AppBar
          position="fixed"
          elevation={0}
          sx={{
            width: { md: `calc(100% - ${drawerWidth}px)` },
            ml: { md: `${drawerWidth}px` },
            borderBottom: 1,
            borderColor: 'divider',
            bgcolor: 'background.paper',
            color: 'text.primary',
            transition: sidebarTransition,
          }}
        >
          <Toolbar>
            <IconButton
              color="inherit"
              edge="start"
              onClick={() => setMobileOpen(o => !o)}
              sx={{ mr: 2, display: { md: 'none' } }}
            >
              <MenuIcon />
            </IconButton>
            <Box sx={{ flexGrow: 1 }} />
            <IconButton onClick={() => setDarkMode(p => !p)} sx={{ mr: 1 }}>
              {darkMode ? <LightModeIcon /> : <DarkModeIcon />}
            </IconButton>
            <UserMenu
              login={login}
              label={label}
              role={role}
              onLogout={handleLogout}
              onSettings={() => navigate('/settings')}
            />
          </Toolbar>
        </AppBar>

        <Box
          component="nav"
          sx={{
            width: { md: drawerWidth },
            flexShrink: { md: 0 },
            transition: sidebarTransition,
          }}
        >
          {/* Mobile: временный drawer */}
          <Drawer
            variant="temporary"
            open={mobileOpen}
            onClose={() => setMobileOpen(false)}
            ModalProps={{ keepMounted: true }}
            sx={{
              display: { xs: 'block', md: 'none' },
              '& .MuiDrawer-paper': { width: DRAWER_WIDTH, boxSizing: 'border-box' },
            }}
          >
            {sidebarContent}
          </Drawer>

          {/* Desktop: постоянный drawer с collapse */}
          <Drawer
            variant="permanent"
            open
            sx={{
              display: { xs: 'none', md: 'block' },
              '& .MuiDrawer-paper': {
                width: drawerWidth,
                boxSizing: 'border-box',
                borderRight: 1,
                borderColor: 'divider',
                overflowX: 'hidden',
                transition: sidebarTransition,
              },
            }}
          >
            {sidebarContent}
          </Drawer>
        </Box>

        <Box
          component="main"
          sx={{
            flexGrow: 1,
            p: { xs: 2, sm: 2.5, md: 3 },
            width: { md: `calc(100% - ${drawerWidth}px)` },
            minHeight: '100vh',
            bgcolor: 'background.default',
            transition: sidebarTransition,
            overflow: 'hidden',
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
