import React, { useState, useEffect, useCallback } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import {
  Alert,
  Box,
  Button,
  Card,
  CardContent,
  CircularProgress,
  Dialog,
  DialogContent,
  Grid,
  IconButton,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Typography,
  Breadcrumbs,
  Link,
  TextField,
  MenuItem,
  useMediaQuery,
  useTheme,
} from '@mui/material';
import {
  ArrowBack as ArrowBackIcon,
  Download as DownloadIcon,
  Compare as CompareIcon,
  Close as CloseIcon,
  History as HistoryIcon,
  CalendarToday as CalendarTodayIcon,
  FileDownload as FileDownloadIcon,
} from '@mui/icons-material';
import { getDeviceVersions, getVersionContent, getVersionDiff, getDiffByDate, exportConfigByDate } from '../utils/api';
import { formatDateTime, toDDMMYYYY } from '../utils/dateFormatter';
import { downloadText } from '../utils/downloadFile';

const parseDateError = (error, toDDMMYYYY) => {
  const msg = error.response?.data?.error || error.message || '';
  if (typeof msg !== 'string') return String(msg);
  const withLast = msg.match(/no version for date (\S+); last config registered: (\S+)/);
  if (withLast) return `Версия для даты ${toDDMMYYYY(withLast[1])} не обнаружена. Последняя дата регистрации: ${toDDMMYYYY(withLast[2])}`;
  const onlyDate = msg.match(/no version for date (\S+)/);
  if (onlyDate) return `Версия для даты ${toDDMMYYYY(onlyDate[1])} не обнаружена.`;
  return msg;
};
import ChangesTab from '../components/ChangesTab';
import ConfigViewDialog from '../components/ConfigViewDialog';

const DeviceDetails = () => {
  const theme = useTheme();
  const isMobile = useMediaQuery(theme.breakpoints.down('sm'));
  const { id } = useParams();
  const navigate = useNavigate();
  const [loading, setLoading] = useState(true);
  const [versions, setVersions] = useState([]);
  const [device, setDevice] = useState(null);
  const [selectedVersions, setSelectedVersions] = useState({ left: null, right: null });
  const [viewDialog, setViewDialog] = useState({ open: false, content: '', versionId: null });
  const [compareDialog, setCompareDialog] = useState({ open: false, diffData: null, loading: false, leftTitle: null, rightTitle: null });
  const [dateCompare, setDateCompare] = useState({ date1: '', date2: '', loading: false, error: '' });
  const [exportState, setExportState] = useState({ date: '', loading: false, error: '' });

  const loadData = useCallback(async () => {
    try {
      setLoading(true);
      const data = await getDeviceVersions(id);
      setVersions(data.versions || []);
      setDevice(data.device || { id });
    } catch (error) {
      console.error('Failed to load device data:', error);
    } finally {
      setLoading(false);
    }
  }, [id]);

  useEffect(() => {
    loadData();
  }, [loadData]);

  const handleViewVersion = async (versionId) => {
    try {
      const content = await getVersionContent(versionId);
      setViewDialog({ open: true, content, versionId });
    } catch (error) {
      console.error('Failed to load version content:', error);
    }
  };

  const buildVersionFilename = (versionId) => {
    const version = versions.find((v) => Number(v.id) === Number(versionId));
    const safeHostname = String(device?.hostname || id).replace(/[\\/:*?"<>|]/g, '_');
    let datePart = 'unknown-date';

    if (version?.created_at) {
      const date = new Date(version.created_at);
      if (!Number.isNaN(date.getTime())) {
        datePart = date.toISOString().slice(0, 10);
      }
    }

    return `config_${safeHostname}_${datePart}.txt`;
  };

  const handleDownloadVersion = async (versionId) => {
    try {
      const content = await getVersionContent(versionId);
      downloadText(content, buildVersionFilename(versionId));
    } catch (error) {
      console.error('Failed to download version:', error);
    }
  };

  const handleCompare = async () => {
    if (!selectedVersions.left || !selectedVersions.right) return;
    
    try {
      setCompareDialog({ open: true, diffData: null, loading: true, leftTitle: null, rightTitle: null });
      const diff = await getVersionDiff(selectedVersions.left, selectedVersions.right);
      const left = versions.find((v) => Number(v.id) === Number(selectedVersions.left));
      const right = versions.find((v) => Number(v.id) === Number(selectedVersions.right));
      const leftTitle = left?.created_at ? `${device?.hostname || 'Конфиг'} (${formatDateTime(left.created_at)})` : null;
      const rightTitle = right?.created_at ? `${device?.hostname || 'Конфиг'} (${formatDateTime(right.created_at)})` : null;
      setCompareDialog({ open: true, diffData: diff, loading: false, leftTitle, rightTitle });
    } catch (error) {
      console.error('Failed to compare versions:', error);
      setCompareDialog({ open: false, diffData: null, loading: false, leftTitle: null, rightTitle: null });
    }
  };

  const closeCompareDialog = () => {
    setCompareDialog({ open: false, diffData: null, loading: false, leftTitle: null, rightTitle: null });
    setSelectedVersions({ left: null, right: null });
  };

  const handleCompareByDate = async () => {
    if (!dateCompare.date1 || !dateCompare.date2) return;
    setDateCompare(p => ({ ...p, loading: true, error: '' }));
    setCompareDialog({ open: true, diffData: null, loading: true, leftTitle: null, rightTitle: null });
    try {
      const diff = await getDiffByDate(id, dateCompare.date1, dateCompare.date2);
      const left = versions.find(v => Number(v.id) === Number(diff.left_version_id));
      const right = versions.find(v => Number(v.id) === Number(diff.right_version_id));
      const hostname = device?.hostname || 'Конфиг';
      setCompareDialog({
        open: true, diffData: diff, loading: false,
        leftTitle: `${hostname} (${left?.created_at ? formatDateTime(left.created_at) : toDDMMYYYY(dateCompare.date1)})`,
        rightTitle: `${hostname} (${right?.created_at ? formatDateTime(right.created_at) : toDDMMYYYY(dateCompare.date2)})`,
      });
    } catch (error) {
      setCompareDialog({ open: false, diffData: null, loading: false, leftTitle: null, rightTitle: null });
      setDateCompare(p => ({ ...p, error: parseDateError(error, toDDMMYYYY) }));
    } finally {
      setDateCompare(p => ({ ...p, loading: false }));
    }
  };

  const handleExportByDate = async () => {
    if (!exportState.date) return;
    setExportState(p => ({ ...p, loading: true, error: '' }));
    try {
      const response = await exportConfigByDate(id, exportState.date);
      const disposition = response.headers['content-disposition'];
      const filenameMatch = disposition && /filename="?([^";\n]+)"?/.exec(disposition);
      const filename = filenameMatch ? filenameMatch[1] : `config_${device?.hostname || id}_${exportState.date}.txt`;
      const url = URL.createObjectURL(response.data);
      const a = document.createElement('a');
      a.href = url;
      a.download = filename;
      a.click();
      URL.revokeObjectURL(url);
    } catch (error) {
      let msg = error.response?.data?.error || error.message;
      if (error.response?.status === 404 && error.response.data instanceof Blob) {
        try {
          const parsed = JSON.parse(await error.response.data.text());
          msg = parsed.last_registered_date
            ? `Конфиг для даты ${toDDMMYYYY(exportState.date)} не обнаружен. Последняя дата регистрации: ${toDDMMYYYY(parsed.last_registered_date)}`
            : `Конфиг для даты ${toDDMMYYYY(exportState.date)} не обнаружен.`;
        } catch { /* msg уже установлен */ }
      }
      setExportState(p => ({ ...p, error: msg }));
    } finally {
      setExportState(p => ({ ...p, loading: false }));
    }
  };

  if (loading) {
    return (
      <Box sx={{ display: 'flex', justifyContent: 'center', py: 8 }}>
        <CircularProgress />
      </Box>
    );
  }

  return (
    <Box>
      <Box sx={{ mb: 3 }}>
        <Button
          startIcon={<ArrowBackIcon />}
          onClick={() => navigate('/devices')}
          sx={{ mb: 2 }}
        >
          К списку устройств
        </Button>
        <Breadcrumbs>
          <Link
            component="button"
            variant="body1"
            onClick={() => navigate('/devices')}
            underline="hover"
            color="inherit"
          >
            Устройства
          </Link>
          <Typography color="text.primary">{device?.hostname || `Устройство #${id}`}</Typography>
        </Breadcrumbs>
      </Box>

      <Typography variant="h4" fontWeight={600} gutterBottom>
        {device?.hostname || `Устройство #${id}`}
      </Typography>

      <Grid container spacing={3} sx={{ mb: 4 }}>
        <Grid item xs={12} sm={6} md={3}>
          <Card>
            <CardContent>
              <Typography variant="body2" color="text.secondary">IP адрес</Typography>
              <Typography fontWeight={500}>{device?.mgmt_ip || '-'}</Typography>
            </CardContent>
          </Card>
        </Grid>
        <Grid item xs={12} sm={6} md={3}>
          <Card>
            <CardContent>
              <Typography variant="body2" color="text.secondary">Вендор</Typography>
              <Typography fontWeight={500}>{device?.vendor || '-'}</Typography>
            </CardContent>
          </Card>
        </Grid>
        <Grid item xs={12} sm={6} md={3}>
          <Card>
            <CardContent>
              <Typography variant="body2" color="text.secondary">Модель</Typography>
              <Typography fontWeight={500}>{device?.model || '-'}</Typography>
            </CardContent>
          </Card>
        </Grid>
        <Grid item xs={12} sm={6} md={3}>
          <Card>
            <CardContent>
              <Typography variant="body2" color="text.secondary">Версий</Typography>
              <Typography fontWeight={500}>{versions.length}</Typography>
            </CardContent>
          </Card>
        </Grid>
      </Grid>

      <Card sx={{ mb: 3 }}>
        <CardContent>
          <Typography variant="h6" fontWeight={600} gutterBottom>
            Сравнение версий
          </Typography>
          <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
            Выберите две версии для сравнения
          </Typography>
          <Grid container spacing={2} alignItems="center">
            <Grid item xs={12} sm={5}>
              <TextField
                select
                fullWidth
                size="small"
                label="Версия 1 (левая)"
                value={selectedVersions.left || ''}
                onChange={(e) => setSelectedVersions(prev => ({ ...prev, left: e.target.value }))}
              >
                <MenuItem value="">Выберите версию...</MenuItem>
                {versions.map((v) => (
                  <MenuItem key={v.id} value={v.id}>
                    {formatDateTime(v.created_at)}
                  </MenuItem>
                ))}
              </TextField>
            </Grid>
            <Grid item xs={12} sm={2} sx={{ textAlign: 'center' }}>
              <CompareIcon color="action" />
            </Grid>
            <Grid item xs={12} sm={5}>
              <TextField
                select
                fullWidth
                size="small"
                label="Версия 2 (правая)"
                value={selectedVersions.right || ''}
                onChange={(e) => setSelectedVersions(prev => ({ ...prev, right: e.target.value }))}
              >
                <MenuItem value="">Выберите версию...</MenuItem>
                {versions.map((v) => (
                  <MenuItem key={v.id} value={v.id}>
                    {formatDateTime(v.created_at)}
                  </MenuItem>
                ))}
              </TextField>
            </Grid>
            <Grid item xs={12}>
              <Button
                variant="contained"
                startIcon={<CompareIcon />}
                onClick={handleCompare}
                disabled={!selectedVersions.left || !selectedVersions.right}
              >
                Сравнить
              </Button>
            </Grid>
          </Grid>
        </CardContent>
      </Card>

      {/* UC-2: Сравнение конфигурации между датами */}
      <Card sx={{ mb: 3 }}>
        <CardContent>
          <Typography variant="h6" fontWeight={600} gutterBottom>
            <CalendarTodayIcon sx={{ mr: 1, verticalAlign: 'middle' }} />
            Сравнение по датам
          </Typography>
          <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
            Выберите две даты для сравнения конфигурации устройства
          </Typography>
          <Grid container spacing={2} alignItems="center">
            <Grid item xs={12} sm={5}>
              <TextField
                fullWidth
                size="small"
                type="date"
                label="Дата 1 (старая)"
                InputLabelProps={{ shrink: true }}
                value={dateCompare.date1}
                onChange={(e) => setDateCompare(p => ({ ...p, date1: e.target.value }))}
              />
            </Grid>
            <Grid item xs={12} sm={2} sx={{ textAlign: 'center' }}>
              <CompareIcon color="action" />
            </Grid>
            <Grid item xs={12} sm={5}>
              <TextField
                fullWidth
                size="small"
                type="date"
                label="Дата 2 (новая)"
                InputLabelProps={{ shrink: true }}
                value={dateCompare.date2}
                onChange={(e) => setDateCompare(p => ({ ...p, date2: e.target.value }))}
              />
            </Grid>
            <Grid item xs={12}>
              <Button
                variant="contained"
                startIcon={dateCompare.loading ? <CircularProgress size={20} color="inherit" /> : <CompareIcon />}
                onClick={handleCompareByDate}
                disabled={!dateCompare.date1 || !dateCompare.date2 || dateCompare.loading}
              >
                {dateCompare.loading ? 'Сравнение...' : 'Сравнить по датам'}
              </Button>
            </Grid>
            {dateCompare.error && (
              <Grid item xs={12}>
                <Alert severity="error" onClose={() => setDateCompare(p => ({ ...p, error: '' }))}>
                  {dateCompare.error}
                </Alert>
              </Grid>
            )}
          </Grid>
        </CardContent>
      </Card>

      {/* Выгрузка конфига за выбранную дату */}
      <Card sx={{ mb: 3 }}>
        <CardContent>
          <Typography variant="h6" fontWeight={600} gutterBottom>
            <FileDownloadIcon sx={{ mr: 1, verticalAlign: 'middle' }} />
            Выгрузить конфиг за дату
          </Typography>
          <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
            Скачать конфигурацию устройства на выбранную дату
          </Typography>
          <Grid container spacing={2} alignItems="center">
            <Grid item xs={12} sm={4}>
              <TextField
                fullWidth
                size="small"
                type="date"
                label="Дата"
                InputLabelProps={{ shrink: true }}
                value={exportState.date}
                onChange={(e) => setExportState(p => ({ ...p, date: e.target.value }))}
              />
            </Grid>
            <Grid item xs={12} sm={4}>
              <Button
                variant="contained"
                startIcon={exportState.loading ? <CircularProgress size={20} color="inherit" /> : <DownloadIcon />}
                onClick={handleExportByDate}
                disabled={!exportState.date || exportState.loading}
              >
                {exportState.loading ? 'Выгрузка...' : 'Скачать конфиг'}
              </Button>
            </Grid>
            {exportState.error && (
              <Grid item xs={12}>
                <Alert severity="error" onClose={() => setExportState(p => ({ ...p, error: '' }))}>
                  {exportState.error}
                </Alert>
              </Grid>
            )}
          </Grid>
        </CardContent>
      </Card>

      <Card>
        <CardContent>
          <Typography variant="h6" fontWeight={600} gutterBottom>
            <HistoryIcon sx={{ mr: 1, verticalAlign: 'middle' }} />
            История версий
          </Typography>
          <TableContainer>
            <Table>
              <TableHead>
                <TableRow>
                  <TableCell>Дата</TableCell>
                  <TableCell>Размер</TableCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {versions.map((version) => (
                  <TableRow
                    key={version.id}
                    hover
                    onClick={() => handleViewVersion(version.id)}
                    sx={{ cursor: 'pointer' }}
                  >
                    <TableCell>{formatDateTime(version.created_at)}</TableCell>
                    <TableCell >{version.original_size ? (version.original_size < 1024 ? `${version.original_size} байт` : `${(version.original_size / 1024).toFixed(1)} КБ`) : '-'}</TableCell>
                  </TableRow>
                ))}
                {versions.length === 0 && (
                  <TableRow>
                    <TableCell colSpan={2} align="center" sx={{ py: 4 }}>
                      <Typography color="text.secondary">Нет доступных версий</Typography>
                    </TableCell>
                  </TableRow>
                )}
              </TableBody>
            </Table>
          </TableContainer>
        </CardContent>
      </Card>

      <ConfigViewDialog
        open={viewDialog.open}
        onClose={() => setViewDialog({ open: false, content: '', versionId: null })}
        title={`Просмотр: ${device?.hostname || 'Устройство'}${viewDialog.versionId ? ` (версия от ${formatDateTime(versions.find(v => v.id === viewDialog.versionId)?.created_at)})` : ''}`}
        content={viewDialog.content}
        onDownload={viewDialog.versionId ? () => handleDownloadVersion(viewDialog.versionId) : null}
      />

      {/* Compare Dialog */}
      <Dialog
        open={compareDialog.open}
        onClose={closeCompareDialog}
        maxWidth="xl"
        fullWidth
        fullScreen={isMobile}
      >
        <DialogContent sx={{ p: 0 }}>
          <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', p: 2, borderBottom: 1, borderColor: 'divider' }}>
            <Typography variant="h6">Сравнение версий: {device?.hostname}</Typography>
            <IconButton onClick={closeCompareDialog}>
              <CloseIcon />
            </IconButton>
          </Box>
          <Box sx={{ p: 2 }}>
            {compareDialog.loading ? (
              <Box sx={{ display: 'flex', justifyContent: 'center', py: 8 }}>
                <CircularProgress />
              </Box>
            ) : compareDialog.diffData ? (
              <ChangesTab 
                embedded 
                initialDiffData={compareDialog.diffData}
                deviceName={device?.hostname}
                leftTitle={compareDialog.leftTitle}
                rightTitle={compareDialog.rightTitle}
                leftDeviceId={device?.id}
                rightDeviceId={device?.id}
                leftVersionId={compareDialog.diffData.left_version_id}
                rightVersionId={compareDialog.diffData.right_version_id}
                onViewVersion={handleViewVersion}
              />
            ) : (
              <Typography>Ошибка при загрузке diff</Typography>
            )}
          </Box>
        </DialogContent>
      </Dialog>
    </Box>
  );
};

export default DeviceDetails;