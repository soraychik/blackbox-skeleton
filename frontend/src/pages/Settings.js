import React from 'react';
import {
  Box,
  Card,
  CardContent,
  Divider,
  IconButton,
  Switch,
  Typography,
  Tooltip,
} from '@mui/material';
import {
  Add as AddIcon,
  Remove as RemoveIcon,
  Palette as PaletteIcon,
} from '@mui/icons-material';
import { SettingsContext } from '../components/Layout';

const ACCENT_COLORS = [
  { value: '#2563eb', label: 'Синий' },
  { value: '#7c3aed', label: 'Фиолетовый' },
  { value: '#059669', label: 'Изумрудный' },
  { value: '#d97706', label: 'Оранжевый' },
  { value: '#dc2626', label: 'Красный' },
  { value: '#0891b2', label: 'Бирюзовый' },
  { value: '#db2777', label: 'Розовый' },
  { value: '#4f46e5', label: 'Индиго' },
];

const SCALE_OPTIONS = [
  { value: 0.8, label: '80%' },
  { value: 0.9, label: '90%' },
  { value: 1.0, label: '100%' },
  { value: 1.1, label: '110%' },
  { value: 1.2, label: '120%' },
];

const ScaleSelector = ({ value, onChange }) => {
  const scalePercent = Math.round(value * 100);
  const canDecrease = value > 0.8;
  const canIncrease = value < 1.2;

  return (
    <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
      <Typography variant="body1" fontWeight={500}>
        Масштаб интерфейса
      </Typography>
      <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5 }}>
        <IconButton
          size="small"
          onClick={() => onChange(Math.max(0.8, value - 0.1))}
          disabled={!canDecrease}
        >
          <RemoveIcon fontSize="small" />
        </IconButton>
        <Typography
          variant="body2"
          sx={{
            minWidth: 48,
            textAlign: 'center',
            fontVariantNumeric: 'tabular-nums',
          }}
        >
          {scalePercent}%
        </Typography>
        <IconButton
          size="small"
          onClick={() => onChange(Math.min(1.2, value + 0.1))}
          disabled={!canIncrease}
        >
          <AddIcon fontSize="small" />
        </IconButton>
      </Box>
    </Box>
  );
};

const Settings = () => {
  const { darkMode, setDarkMode, scale, setScale, accentColor, setAccentColor } = React.useContext(SettingsContext);

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
                onChange={() => setDarkMode((prev) => !prev)}
                color="primary"
              />
            </Box>

            <Divider sx={{ my: 3 }} />

            <Box sx={{ mb: 2 }}>
              <ScaleSelector value={scale} onChange={setScale} />
            </Box>

            <Box sx={{ mb: 3 }}>
              <Typography variant="body1" fontWeight={500} sx={{ mb: 1.5 }}>
                Язык интерфейса
              </Typography>
              <Typography variant="body2" color="text.secondary">
                Русский (скоро доступно)
              </Typography>
            </Box>

            <Box>
              <Typography variant="body1" fontWeight={500} sx={{ mb: 1.5 }}>
                Акцентный цвет
              </Typography>
              <Box sx={{ display: 'flex', gap: 1.5, flexWrap: 'wrap' }}>
                {ACCENT_COLORS.map((color) => (
                  <Tooltip key={color.value} title={color.label}>
                    <Box
                      onClick={() => setAccentColor(color.value)}
                      sx={{
                        width: 36,
                        height: 36,
                        borderRadius: '50%',
                        bgcolor: color.value,
                        cursor: 'pointer',
                        border: accentColor === color.value ? '3px solid' : '3px solid transparent',
                        borderColor: accentColor === color.value ? 'text.primary' : 'transparent',
                        transition: 'all 0.2s ease',
                        '&:hover': {
                          transform: 'scale(1.1)',
                        },
                      }}
                    />
                  </Tooltip>
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
