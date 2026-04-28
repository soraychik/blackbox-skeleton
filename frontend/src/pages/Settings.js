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
  TextField,
  Select,
  MenuItem,
  FormControl,
  InputLabel,
  Grid,
  Button,
} from '@mui/material';
import {
  Add as AddIcon,
  Remove as RemoveIcon,
  Palette as PaletteIcon,
  Folder as FolderIcon,
  Security as SecurityIcon,
  Settings as SettingsIcon,
  Cloud as CloudIcon,
  Save as SaveIcon,
} from '@mui/icons-material';
import { SettingsContext } from '../components/Layout';
import { getSettings, updateSettings } from '../utils/api';

const ACCENT_COLORS = [
  { value: '#2563eb', label: 'Синий' },
  { value: '#7c3aed', label: 'Фиолетовый' },
  { value: '#059669', label: 'Зумрудный' },
  { value: '#d97706', label: 'Оранжевый' },
  { value: '#dc2626', label: 'Красный' },
  { value: '#0891b2', label: 'Бирюзовый' },
  { value: '#db2777', label: 'Розовый' },
  { value: '#4f46e5', label: 'Индиго' },
];

const ScaleSelector = ({ value, onChange }) => {
  const displayPercent = Math.round((value + 0.2) * 100);
  const canDecrease = value > 0.6;
  const canIncrease = value < 1.0;

  return (
    <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
      <Typography variant="body1" fontWeight={500}>
        Масштаб интерфейса
      </Typography>
      <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5 }}>
        <IconButton
          size="small"
          onClick={() => onChange(Math.max(0.6, value - 0.1))}
          disabled={!canDecrease}
        >
          <RemoveIcon fontSize="small" />
        </IconButton>
        <Typography
          variant="body2"
          sx={{ minWidth: 48, textAlign: 'center', fontVariantNumeric: 'tabular-nums' }}
        >
          {displayPercent}%
        </Typography>
        <IconButton
          size="small"
          onClick={() => onChange(Math.min(1.0, value + 0.1))}
          disabled={!canIncrease}
        >
          <AddIcon fontSize="small" />
        </IconButton>
      </Box>
    </Box>
  );
};

const Settings = () => {
  const { darkMode, setDarkMode, scale, setScale, accentColor, setAccentColor } =
    React.useContext(SettingsContext);

  const [fileServerType, setFileServerType] = React.useState('local');
  const [configSourcePath, setConfigSourcePath] = React.useState('');
  const [smbUsername, setSmbUsername] = React.useState('');
  const [smbPassword, setSmbPassword] = React.useState('');
  const [smbDomain, setSmbDomain] = React.useState('WORKGROUP');
  const [saving, setSaving] = React.useState(false);
  const [saveMessage, setSaveMessage] = React.useState('');

  React.useEffect(() => {
    getSettings()
      .then((settings) => {
        if (settings.config_source_type) setFileServerType(settings.config_source_type);
        if (settings.config_source_path) setConfigSourcePath(settings.config_source_path);
        if (settings.smb_username) setSmbUsername(settings.smb_username);
        if (settings.smb_domain) setSmbDomain(settings.smb_domain);
        // Пароль не загружаем из соображений безопасности
      })
      .catch((err) => console.error('Failed to load settings:', err));
  }, []);

  const handleSaveConfigSource = async () => {
    setSaving(true);
    setSaveMessage('');
    try {
      const payload = {
        config_source_type: fileServerType,
        config_source_path: configSourcePath,
      };
      if (fileServerType === 'smb') {
        payload.smb_username = smbUsername;
        if (smbPassword) payload.smb_password = smbPassword;
        payload.smb_domain = smbDomain || 'WORKGROUP';
      }
      await updateSettings(payload);
      setSaveMessage('Настройки сохранены');
    } catch (err) {
      setSaveMessage('Ошибка сохранения: ' + (err.response?.data?.error || err.message));
    } finally {
      setSaving(false);
      setTimeout(() => setSaveMessage(''), 3000);
    }
  };

  const pathConfig = {
    local: {
      placeholder: '/srv/configs   или   C:\\configs',
      helper: 'Абсолютный путь к директории на хосте (Linux или Windows)',
    },
    smb: {
      placeholder: '//192.168.1.1/configs',
      helper: '//адрес_сервера/имя_ресурса',
    },
    nfs: {
      placeholder: '//192.168.1.1/srv/configs',
      helper: '//адрес_сервера/путь   или   сервер:/путь',
    },
  };

  return (
    <Box sx={{ maxWidth: 720, mx: 'auto' }}>
      <Typography variant="h4" fontWeight={600} gutterBottom>
        Настройки
      </Typography>
      <Typography variant="body1" color="text.secondary" sx={{ mb: 4 }}>
        Конфигурация системы
      </Typography>

      <Box sx={{ display: 'flex', flexDirection: 'column', gap: 3 }}>
        {/* Внешний вид */}
        <Card>
          <CardContent sx={{ p: 4 }}>
            <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5, mb: 1 }}>
              <PaletteIcon color="primary" sx={{ fontSize: 28 }} />
              <Typography variant="h5" fontWeight={600}>Внешний вид</Typography>
            </Box>
            <Divider sx={{ mb: 3 }} />

            <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', mb: 4 }}>
              <Box>
                <Typography variant="body1" fontWeight={500}>Тёмная тема</Typography>
                <Typography variant="body2" color="text.secondary">Переключить на тёмный режим</Typography>
              </Box>
              <Switch checked={darkMode} onChange={() => setDarkMode((prev) => !prev)} color="primary" />
            </Box>

            <Divider sx={{ my: 3 }} />
            <Box sx={{ mb: 2 }}>
              <ScaleSelector value={scale} onChange={setScale} />
            </Box>

            <Box>
              <Typography variant="body1" fontWeight={500} sx={{ mb: 1.5 }}>Акцентный цвет</Typography>
              <Box sx={{ display: 'flex', gap: 1.5, flexWrap: 'wrap' }}>
                {ACCENT_COLORS.map((color) => (
                  <Tooltip key={color.value} title={color.label}>
                    <Box
                      onClick={() => setAccentColor(color.value)}
                      sx={{
                        width: 36, height: 36, borderRadius: '50%', bgcolor: color.value,
                        cursor: 'pointer',
                        border: accentColor === color.value ? '3px solid' : '3px solid transparent',
                        borderColor: accentColor === color.value ? 'text.primary' : 'transparent',
                        transition: 'all 0.2s ease',
                        '&:hover': { transform: 'scale(1.1)' },
                      }}
                    />
                  </Tooltip>
                ))}
              </Box>
            </Box>
          </CardContent>
        </Card>

        {/* Источник конфигов */}
        <Card>
          <CardContent sx={{ p: 4 }}>
            <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5, mb: 1 }}>
              <FolderIcon color="primary" sx={{ fontSize: 28 }} />
              <Typography variant="h5" fontWeight={600}>Источник конфигурационных файлов</Typography>
            </Box>
            <Divider sx={{ mb: 3 }} />

            <Grid container spacing={3}>
              {/* Тип подключения */}
              <Grid item xs={12} sm={4}>
                <FormControl fullWidth>
                  <InputLabel>Тип подключения</InputLabel>
                  <Select
                    label="Тип подключения"
                    value={fileServerType}
                    onChange={(e) => {
                      setFileServerType(e.target.value);
                      setConfigSourcePath('');
                    }}
                  >
                    <MenuItem value="local">Локальная папка</MenuItem>
                    <MenuItem value="smb">SMB / CIFS</MenuItem>
                    <MenuItem value="nfs">NFS</MenuItem>
                  </Select>
                </FormControl>
              </Grid>

              {/* Путь */}
              <Grid item xs={12} sm={8}>
                <TextField
                  fullWidth
                  label={fileServerType === 'local' ? 'Путь к директории' : 'Адрес сетевого ресурса'}
                  value={configSourcePath}
                  onChange={(e) => setConfigSourcePath(e.target.value)}
                  placeholder={pathConfig[fileServerType]?.placeholder}
                  helperText={pathConfig[fileServerType]?.helper}
                />
              </Grid>

              {/* SMB: учётные данные */}
              {fileServerType === 'smb' && (
                <>
                  <Grid item xs={12} sm={4}>
                    <TextField
                      fullWidth
                      label="Имя пользователя"
                      placeholder="guest"
                      value={smbUsername}
                      onChange={(e) => setSmbUsername(e.target.value)}
                    />
                  </Grid>
                  <Grid item xs={12} sm={4}>
                    <TextField
                      fullWidth
                      label="Пароль"
                      type="password"
                      placeholder="••••••••"
                      value={smbPassword}
                      onChange={(e) => setSmbPassword(e.target.value)}
                      helperText="Оставьте пустым, чтобы сохранить текущий пароль"
                    />
                  </Grid>
                  <Grid item xs={12} sm={4}>
                    <TextField
                      fullWidth
                      label="Домен"
                      placeholder="WORKGROUP"
                      value={smbDomain}
                      onChange={(e) => setSmbDomain(e.target.value)}
                    />
                  </Grid>
                </>
              )}

              {/* Кнопка сохранить */}
              <Grid item xs={12}>
                <Box sx={{ display: 'flex', alignItems: 'center', gap: 2 }}>
                  <Button
                    variant="contained"
                    onClick={handleSaveConfigSource}
                    disabled={saving}
                    startIcon={<SaveIcon />}
                  >
                    {saving ? 'Сохранение...' : 'Сохранить'}
                  </Button>
                  {saveMessage && (
                    <Typography
                      variant="body2"
                      color={saveMessage.includes('Ошибка') ? 'error' : 'success.main'}
                    >
                      {saveMessage}
                    </Typography>
                  )}
                </Box>
              </Grid>
            </Grid>
          </CardContent>
        </Card>

        {/* Active Directory */}
        <Card>
          <CardContent sx={{ p: 4 }}>
            <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5, mb: 1 }}>
              <SecurityIcon color="primary" sx={{ fontSize: 28 }} />
              <Typography variant="h5" fontWeight={600}>Active Directory</Typography>
            </Box>
            <Divider sx={{ mb: 3 }} />
            <Typography variant="body2" color="text.secondary" sx={{ mb: 4 }}>
              Интеграция с LDAP для авторизации пользователей
            </Typography>
            <TextField
              fullWidth
              label="LDAP URL"
              placeholder="ldap://domain.com:389"
              helperText="Адрес LDAP сервера для авторизации"
            />
          </CardContent>
        </Card>

        {/* Параметры сканирования */}
        <Card>
          <CardContent sx={{ p: 4 }}>
            <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5, mb: 1 }}>
              <SettingsIcon color="primary" sx={{ fontSize: 28 }} />
              <Typography variant="h5" fontWeight={600}>Параметры сканирования</Typography>
            </Box>
            <Divider sx={{ mb: 3 }} />
            <Typography variant="body2" color="text.secondary" sx={{ mb: 3 }}>
              Параметры задаются через переменные окружения. Управление через интерфейс запланировано.
            </Typography>
            <Grid container spacing={3}>
              <Grid item xs={12} sm={6}>
                <TextField
                  fullWidth label="Интервал автосканирования (сек)" type="number"
                  placeholder="300" helperText="SCAN_INTERVAL_SECONDS — период между автоматическими сканированиями"
                  inputProps={{ min: 5 }}
                  disabled
                />
              </Grid>
              <Grid item xs={12} sm={6}>
                <TextField
                  fullWidth label="Порог фиксации изменений" type="number"
                  placeholder="0.1" helperText="DIFF_THRESHOLD — минимальная доля изменений для сохранения новой версии (0–1)"
                  inputProps={{ step: 0.01, min: 0, max: 1 }}
                  disabled
                />
              </Grid>
              <Grid item xs={12} sm={6}>
                <TextField
                  fullWidth label="Таймаут подключения (сек)" type="number"
                  placeholder="15" helperText="Максимальное время ожидания при установке соединения с источником"
                  inputProps={{ min: 1 }}
                  disabled
                />
              </Grid>
              <Grid item xs={12} sm={6}>
                <TextField
                  fullWidth label="Максимальный размер файла (МБ)" type="number"
                  placeholder="50" helperText="Файлы, превышающие указанный размер, будут пропущены при обработке"
                  inputProps={{ min: 1 }}
                  disabled
                />
              </Grid>
              <Grid item xs={12} sm={6}>
                <TextField
                  fullWidth label="Количество повторных попыток" type="number"
                  placeholder="3" helperText="Число повторных попыток при сбое подключения к источнику"
                  inputProps={{ min: 0, max: 10 }}
                  disabled
                />
              </Grid>
            </Grid>
          </CardContent>
        </Card>

        {/* MinIO */}
        <Card>
          <CardContent sx={{ p: 4 }}>
            <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5, mb: 1 }}>
              <CloudIcon color="primary" sx={{ fontSize: 28 }} />
              <Typography variant="h5" fontWeight={600}>MinIO (хранилище)</Typography>
            </Box>
            <Divider sx={{ mb: 3 }} />
            <Typography variant="body2" color="text.secondary" sx={{ mb: 4 }}>
              Настройка подключения к объектному хранилищу
            </Typography>
            <Grid container spacing={3}>
              <Grid item xs={12} sm={6}>
                <TextField fullWidth label="Endpoint" placeholder="minio:9000" />
              </Grid>
              <Grid item xs={12} sm={6}>
                <TextField fullWidth label="Bucket" placeholder="blackbox" />
              </Grid>
              <Grid item xs={12} sm={6}>
                <TextField fullWidth label="Access Key" placeholder="minioadmin" />
              </Grid>
              <Grid item xs={12} sm={6}>
                <TextField fullWidth label="Secret Key" type="password" placeholder="minioadmin123" />
              </Grid>
              <Grid item xs={12}>
                <Box sx={{ display: 'flex', alignItems: 'center', gap: 2 }}>
                  <Typography variant="body1">Использовать SSL</Typography>
                  <Switch defaultChecked={false} />
                </Box>
              </Grid>
            </Grid>
          </CardContent>
        </Card>

      </Box>
    </Box>
  );
};

export default Settings;