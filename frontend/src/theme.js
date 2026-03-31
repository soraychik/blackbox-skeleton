import { createTheme } from '@mui/material/styles';

const getPaletteColors = (mode, accentColor) => {
  const basePrimary = accentColor || (mode === 'light' ? '#2563eb' : '#3b82f6');

  const hexToRgb = (hex) => {
    const result = /^#?([a-f\d]{2})([a-f\d]{2})([a-f\d]{2})$/i.exec(hex);
    return result ? {
      r: parseInt(result[1], 16),
      g: parseInt(result[2], 16),
      b: parseInt(result[3], 16),
    } : null;
  };

  const lighten = (hex, amount) => {
    const rgb = hexToRgb(hex);
    if (!rgb) return hex;
    const lightenAmount = Math.round(amount * 255);
    const r = Math.min(255, rgb.r + lightenAmount);
    const g = Math.min(255, rgb.g + lightenAmount);
    const b = Math.min(255, rgb.b + lightenAmount);
    return `#${((1 << 24) + (r << 16) + (g << 8) + b).toString(16).slice(1)}`;
  };

  const darken = (hex, amount) => {
    const rgb = hexToRgb(hex);
    if (!rgb) return hex;
    const darkenAmount = Math.round(amount * 255);
    const r = Math.max(0, rgb.r - darkenAmount);
    const g = Math.max(0, rgb.g - darkenAmount);
    const b = Math.max(0, rgb.b - darkenAmount);
    return `#${((1 << 24) + (r << 16) + (g << 8) + b).toString(16).slice(1)}`;
  };

  return {
    primary: {
      main: basePrimary,
      light: lighten(basePrimary, 0.15),
      dark: darken(basePrimary, 0.1),
    },
    secondary: {
      main: mode === 'light' ? '#64748b' : '#94a3b8',
    },
    background: mode === 'light'
      ? { default: '#f8fafc', paper: '#ffffff' }
      : { default: '#0f172a', paper: '#1e293b' },
    text: mode === 'light'
      ? { primary: '#1e293b', secondary: '#64748b' }
      : { primary: '#f1f5f9', secondary: '#94a3b8' },
  };
};

export const getTheme = (mode, accentColor) =>
  createTheme({
    palette: {
      mode,
      ...getPaletteColors(mode, accentColor),
    },
    typography: {
      fontFamily: '"Inter", "Roboto", "Helvetica", "Arial", sans-serif',
      h1: { fontSize: '2rem', fontWeight: 600 },
      h2: { fontSize: '1.5rem', fontWeight: 600 },
      h3: { fontSize: '1.25rem', fontWeight: 600 },
      h4: { fontSize: '1.125rem', fontWeight: 600 },
    },
    shape: {
      borderRadius: 8,
    },
    components: {
      MuiButton: {
        styleOverrides: {
          root: {
            textTransform: 'none',
            fontWeight: 500,
          },
        },
      },
      MuiCard: {
        styleOverrides: {
          root: {
            boxShadow: '0 1px 3px 0 rgb(0 0 0 / 0.1), 0 1px 2px -1px rgb(0 0 0 / 0.1)',
          },
        },
      },
      MuiPaper: {
        styleOverrides: {
          root: {
            backgroundImage: 'none',
          },
        },
      },
      MuiTableCell: {
        styleOverrides: {
          head: {
            fontWeight: 600,
            backgroundColor: mode === 'light' ? '#f1f5f9' : '#1e293b',
          },
        },
      },
    },
  });
