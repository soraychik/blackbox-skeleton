import React, { useReducer, useEffect, useContext } from 'react';
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
  Save as SaveIcon,
} from '@mui/icons-material';
import { SettingsContext } from '../components/Layout';
import { getSettings, updateSettings } from '../utils/api';

// --- state ---

const initialState = {
  // Источник конфигов
  fileServerType: 'local',
  configSourcePath: '',
  smbUsername: '',
  smbPassword: '',
  smbDomain: 'WORKGROUP',
  sourceSaving: false,
  sourceMessage: '',
  // Параметры сканирования
  scanInterval: 300,
  diffThreshold: 0.1,
  scanSaving: false,
  scanMessage: '',
  // LDAP
  ldapEnabled: false,
  ldapUrl: '',
  ldapBindDn: '',
  ldapBindPassword: '',
  ldapUserBase: '',
  ldapUserFilter: '(sAMAccountName=%s)',
  ldapRoleAdmin: '',
  ldapRoleEngineer: '',
  ldapRoleOperator: '',
  ldapSaving: false,
  ldapMessage: '',
};

const reducer = (state, action) => {
  switch (action.type) {
    case 'SET': return { ...state, [action.field]: action.value };
    case 'LOAD': return { ...state, ...action.settings };
    default: return state;
  }
};

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
    useContext(SettingsContext);

  const [s, dispatch] = useReducer(reducer, initialState);
  const set = (field, value) => dispatch({ type: 'SET', field, value });
  const flash = (msgField, savingField, msg) => {
    set(msgField, msg);
    setTimeout(() => set(msgField, ''), 3000);
    set(savingField, false);
  };

  useEffect(() => {
    getSettings()
      .then(cfg => dispatch({
        type: 'LOAD',
        settings: {
          fileServerType: cfg.config_source_type || 'local',
          configSourcePath: cfg.config_source_path || '',
          smbUsername: cfg.smb_username || '',
          smbDomain: cfg.smb_domain || 'WORKGROUP',
          scanInterval: cfg.scan_interval_seconds || 300,
          diffThreshold: cfg.diff_threshold || 0.1,
          ldapEnabled: !!cfg.ldap_enabled,
          ldapUrl: cfg.ldap_url || '',
          ldapBindDn: cfg.ldap_bind_dn || '',
          ldapUserBase: cfg.ldap_user_base || '',
          ldapUserFilter: cfg.ldap_user_filter || '(sAMAccountName=%s)',
          ldapRoleAdmin: cfg.ldap_role_admin || '',
          ldapRoleEngineer: cfg.ldap_role_engineer || '',
          ldapRoleOperator: cfg.ldap_role_operator || '',
        },
      }))
      .catch(err => console.error('Failed to load settings:', err));
  }, []);

  const handleSaveConfigSource = async () => {
    set('sourceSaving', true);
    set('sourceMessage', '');
    try {
      const payload = { config_source_type: s.fileServerType, config_source_path: s.configSourcePath };
      if (s.fileServerType === 'smb') {
        payload.smb_username = s.smbUsername;
        if (s.smbPassword) payload.smb_password = s.smbPassword;
        payload.smb_domain = s.smbDomain || 'WORKGROUP';
      }
      await updateSettings(payload);
      flash('sourceMessage', 'sourceSaving', 'Настройки сохранены');
    } catch (err) {
      flash('sourceMessage', 'sourceSaving', 'Ошибка: ' + (err.response?.data?.error || err.message));
    }
  };

  const handleSaveScanParams = async () => {
    set('scanSaving', true);
    set('scanMessage', '');
    try {
      await updateSettings({ scan_interval_seconds: Number(s.scanInterval), diff_threshold: Number(s.diffThreshold) });
      flash('scanMessage', 'scanSaving', 'Сохранено');
    } catch (err) {
      flash('scanMessage', 'scanSaving', 'Ошибка: ' + (err.response?.data?.error || err.message));
    }
  };

  const handleSaveLDAP = async () => {
    set('ldapSaving', true);
    set('ldapMessage', '');
    try {
      const payload = {
        ldap_enabled: s.ldapEnabled,
        ldap_url: s.ldapUrl,
        ldap_bind_dn: s.ldapBindDn,
        ldap_user_base: s.ldapUserBase,
        ldap_user_filter: s.ldapUserFilter,
        ldap_role_admin: s.ldapRoleAdmin,
        ldap_role_engineer: s.ldapRoleEngineer,
        ldap_role_operator: s.ldapRoleOperator,
      };
      if (s.ldapBindPassword) payload.ldap_bind_password = s.ldapBindPassword;
      await updateSettings(payload);
      set('ldapBindPassword', '');
      flash('ldapMessage', 'ldapSaving', 'Настройки сохранены');
    } catch (err) {
      flash('ldapMessage', 'ldapSaving', 'Ошибка: ' + (err.response?.data?.error || err.message));
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
                    value={s.fileServerType}
                    onChange={(e) => { set('fileServerType', e.target.value); set('configSourcePath', ''); }}
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
                  label={s.fileServerType === 'local' ? 'Путь к директории' : 'Адрес сетевого ресурса'}
                  value={s.configSourcePath}
                  onChange={(e) => set('configSourcePath', e.target.value)}
                  placeholder={pathConfig[s.fileServerType]?.placeholder}
                  helperText={pathConfig[s.fileServerType]?.helper}
                />
              </Grid>

              {/* SMB: учётные данные */}
              {s.fileServerType === 'smb' && (
                <>
                  <Grid item xs={12} sm={4}>
                    <TextField fullWidth label="Имя пользователя" placeholder="guest"
                      value={s.smbUsername} onChange={(e) => set('smbUsername', e.target.value)} />
                  </Grid>
                  <Grid item xs={12} sm={4}>
                    <TextField fullWidth label="Пароль" type="password" placeholder="••••••••"
                      value={s.smbPassword} onChange={(e) => set('smbPassword', e.target.value)}
                      helperText="Оставьте пустым, чтобы сохранить текущий пароль" />
                  </Grid>
                  <Grid item xs={12} sm={4}>
                    <TextField fullWidth label="Домен" placeholder="WORKGROUP"
                      value={s.smbDomain} onChange={(e) => set('smbDomain', e.target.value)} />
                  </Grid>
                </>
              )}

              {/* Кнопка сохранить */}
              <Grid item xs={12}>
                <Box sx={{ display: 'flex', alignItems: 'center', gap: 2 }}>
                  <Button variant="contained" onClick={handleSaveConfigSource}
                    disabled={s.sourceSaving} startIcon={<SaveIcon />}>
                    {s.sourceSaving ? 'Сохранение...' : 'Сохранить'}
                  </Button>
                  {s.sourceMessage && (
                    <Typography variant="body2" color={s.sourceMessage.includes('Ошибка') ? 'error' : 'success.main'}>
                      {s.sourceMessage}
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
              <Typography variant="h5" fontWeight={600}>Active Directory / LDAP</Typography>
            </Box>
            <Divider sx={{ mb: 3 }} />

            <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', mb: 3 }}>
              <Box>
                <Typography variant="body1" fontWeight={500}>Включить LDAP авторизацию</Typography>
                <Typography variant="body2" color="text.secondary">
                  Пользователи будут входить с корпоративными учётными данными
                </Typography>
              </Box>
              <Switch checked={s.ldapEnabled} onChange={(e) => set('ldapEnabled', e.target.checked)} color="primary" />
            </Box>

            <Grid container spacing={3}>
              <Grid item xs={12} sm={6}>
                <TextField
                  fullWidth
                  label="LDAP URL"
                  value={s.ldapUrl}
                  onChange={(e) => set('ldapUrl', e.target.value)}
                  placeholder="ldap://192.168.1.100:389"
                  helperText="Адрес сервера Active Directory"
                  disabled={!s.ldapEnabled}
                />
              </Grid>
              <Grid item xs={12} sm={6}>
                <TextField
                  fullWidth
                  label="Фильтр поиска пользователя"
                  value={s.ldapUserFilter}
                  onChange={(e) => set('ldapUserFilter', e.target.value)}
                  placeholder="(sAMAccountName=%s)"
                  helperText="%s будет заменён на введённый логин"
                  disabled={!s.ldapEnabled}
                />
              </Grid>
              <Grid item xs={12} sm={8}>
                <TextField
                  fullWidth
                  label="DN сервисного аккаунта (Bind DN)"
                  value={s.ldapBindDn}
                  onChange={(e) => set('ldapBindDn', e.target.value)}
                  placeholder="CN=svc-blackbox,CN=Users,DC=corp,DC=local"
                  helperText="Учётная запись для поиска пользователей в каталоге"
                  disabled={!s.ldapEnabled}
                />
              </Grid>
              <Grid item xs={12} sm={4}>
                <TextField
                  fullWidth
                  label="Пароль сервисного аккаунта"
                  type="password"
                  value={s.ldapBindPassword}
                  onChange={(e) => set('ldapBindPassword', e.target.value)}
                  placeholder="••••••••"
                  helperText="Оставьте пустым, чтобы сохранить текущий"
                  disabled={!s.ldapEnabled}
                />
              </Grid>
              <Grid item xs={12}>
                <TextField
                  fullWidth
                  label="База поиска пользователей (User Base DN)"
                  value={s.ldapUserBase}
                  onChange={(e) => set('ldapUserBase', e.target.value)}
                  placeholder="CN=Users,DC=corp,DC=local"
                  helperText="Раздел каталога, в котором будет выполняться поиск пользователей"
                  disabled={!s.ldapEnabled}
                />
              </Grid>

              <Grid item xs={12}>
                <Typography variant="body2" fontWeight={500} color="text.secondary" sx={{ mb: 1 }}>
                  Маппинг групп AD → роли системы
                </Typography>
              </Grid>
              <Grid item xs={12} sm={4}>
                <TextField
                  fullWidth
                  label="Группа: Администратор"
                  value={s.ldapRoleAdmin}
                  onChange={(e) => set('ldapRoleAdmin', e.target.value)}
                  placeholder="CN=BB-Admins,CN=Users,DC=corp,DC=local"
                  disabled={!s.ldapEnabled}
                />
              </Grid>
              <Grid item xs={12} sm={4}>
                <TextField
                  fullWidth
                  label="Группа: Инженер"
                  value={s.ldapRoleEngineer}
                  onChange={(e) => set('ldapRoleEngineer', e.target.value)}
                  placeholder="CN=BB-Engineers,CN=Users,DC=corp,DC=local"
                  disabled={!s.ldapEnabled}
                />
              </Grid>
              <Grid item xs={12} sm={4}>
                <TextField
                  fullWidth
                  label="Группа: Оператор"
                  value={s.ldapRoleOperator}
                  onChange={(e) => set('ldapRoleOperator', e.target.value)}
                  placeholder="CN=BB-Operators,CN=Users,DC=corp,DC=local"
                  disabled={!s.ldapEnabled}
                />
              </Grid>

              <Grid item xs={12}>
                <Box sx={{ display: 'flex', alignItems: 'center', gap: 2 }}>
                  <Button variant="contained" onClick={handleSaveLDAP}
                    disabled={s.ldapSaving} startIcon={<SaveIcon />}>
                    {s.ldapSaving ? 'Сохранение...' : 'Сохранить'}
                  </Button>
                  {s.ldapMessage && (
                    <Typography variant="body2" color={s.ldapMessage.includes('Ошибка') ? 'error' : 'success.main'}>
                      {s.ldapMessage}
                    </Typography>
                  )}
                </Box>
              </Grid>
            </Grid>
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
            <Grid container spacing={3}>
              <Grid item xs={12} sm={6}>
                <TextField
                  fullWidth
                  label="Интервал автосканирования (сек)"
                  type="number"
                  value={s.scanInterval}
                  onChange={(e) => set('scanInterval', e.target.value)}
                  helperText="Период между автоматическими сканированиями (мин. 5 сек)"
                  inputProps={{ min: 5 }}
                />
              </Grid>
              <Grid item xs={12} sm={6}>
                <TextField
                  fullWidth
                  label="Порог фиксации изменений (0–1)"
                  type="number"
                  value={s.diffThreshold}
                  onChange={(e) => set('diffThreshold', e.target.value)}
                  helperText="Минимальная доля изменений для сохранения новой версии"
                  inputProps={{ step: 0.01, min: 0.01, max: 1 }}
                />
              </Grid>
              <Grid item xs={12}>
                <Box sx={{ display: 'flex', alignItems: 'center', gap: 2 }}>
                  <Button variant="contained" onClick={handleSaveScanParams}
                    disabled={s.scanSaving} startIcon={<SaveIcon />}>
                    {s.scanSaving ? 'Сохранение...' : 'Сохранить'}
                  </Button>
                  {s.scanMessage && (
                    <Typography variant="body2" color={s.scanMessage.includes('Ошибка') ? 'error' : 'success.main'}>
                      {s.scanMessage}
                    </Typography>
                  )}
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