import React, { useContext } from 'react';
import {
  Box,
  Card,
  CardContent,
  Divider,
  FormControl,
  InputLabel,
  MenuItem,
  Select,
  Switch,
  Typography,
} from '@mui/material';
import {
  Palette as PaletteIcon,
} from '@mui/icons-material';
import { ThemeToggleContext } from '../components/Layout';

const Settings = () => {
  const { darkMode, setDarkMode } = useContext(ThemeToggleContext);

  return (
    <Box sx={{ maxWidth: 720, mx: 'auto' }}>
      <Typography variant="h4" fontWeight={600} gutterBottom>
        Настройки
      </Typography>
      <Typography variant="body1" color="text.secondary" sx={{ mb: 4 }}>
        Управление параметрами системы
      </Typography>

      <Box sx={{ display: 'flex', flexDirection: 'column', gap: 3 }}>
        <Card>
          <CardContent sx={{ p: 4 }}>
            <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5, mb: 1 }}>
              <PaletteIcon color="primary" sx={{ fontSize: 28 }} />
              <Typography variant="h5" fontWeight={600}>
                Внешний вид
              </Typography>
            </Box>
            <Divider sx={{ mb: 3 }} />
            <Typography variant="body2" color="text.secondary" sx={{ mb: 4 }}>
              Настройки темы и отображения интерфейса
            </Typography>

            <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', mb: 4 }}>
              <Box>
                <Typography variant="body1" fontWeight={500}>
                  Тёмная тема
                </Typography>
                <Typography variant="body2" color="text.secondary">
                  Переключить на тёмный режим
                </Typography>
              </Box>
              <Switch
                checked={darkMode}
                onChange={() => setDarkMode(!darkMode)}
                color="primary"
              />
            </Box>

            <Divider sx={{ my: 3 }} />

            <Box sx={{ mb: 3 }}>
              <Typography variant="body1" fontWeight={500} sx={{ mb: 1.5 }}>
                Масштаб интерфейса
              </Typography>
              <FormControl fullWidth size="small" disabled sx={{ maxWidth: 300 }}>
                <Select defaultValue="100">
                  <MenuItem value="90">90%</MenuItem>
                  <MenuItem value="100">100%</MenuItem>
                  <MenuItem value="110">110%</MenuItem>
                  <MenuItem value="125">125%</MenuItem>
                </Select>
              </FormControl>
            </Box>

            <Box sx={{ mb: 3 }}>
              <Typography variant="body1" fontWeight={500} sx={{ mb: 1.5 }}>
                Язык интерфейса
              </Typography>
              <FormControl fullWidth size="small" disabled sx={{ maxWidth: 300 }}>
                <Select defaultValue="ru">
                  <MenuItem value="ru">Русский</MenuItem>
                  <MenuItem value="en">English</MenuItem>
                </Select>
              </FormControl>
            </Box>

            <Box>
              <Typography variant="body1" fontWeight={500} sx={{ mb: 1.5 }}>
                Акцентный цвет
              </Typography>
              <Box sx={{ display: 'flex', gap: 1.5 }}>
                {['#1976d2', '#9c27b0', '#2e7d32', '#ed6c02', '#d32f2f'].map((color) => (
                  <Box
                    key={color}
                    sx={{
                      width: 36,
                      height: 36,
                      borderRadius: '50%',
                      bgcolor: color,
                      cursor: 'not-allowed',
                      opacity: 0.4,
                    }}
                  />
                ))}
              </Box>
            </Box>
          </CardContent>
        </Card>
      </Box>
    </Box>
  );
};

export default Settings;
