import React, { useState, useEffect } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import {
  Box,
  Button,
  Card,
  CardContent,
  Chip,
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
} from '@mui/material';
import {
  ArrowBack as ArrowBackIcon,
  Visibility as VisibilityIcon,
  Download as DownloadIcon,
  Compare as CompareIcon,
  Close as CloseIcon,
  History as HistoryIcon,
} from '@mui/icons-material';
import { getDeviceVersions, getVersionContent, getVersionDiff } from '../utils/api';
import { formatDateTime } from '../utils/dateFormatter';
import ChangesTab from '../components/ChangesTab';

const DeviceDetails = () => {
  const { id } = useParams();
  const navigate = useNavigate();
  const [loading, setLoading] = useState(true);
  const [versions, setVersions] = useState([]);
  const [device, setDevice] = useState(null);
  const [selectedVersions, setSelectedVersions] = useState({ left: null, right: null });
  const [compareLoading, setCompareLoading] = useState(false);
  const [viewDialog, setViewDialog] = useState({ open: false, content: '', versionId: null });
  const [compareDialog, setCompareDialog] = useState({ open: false, diffData: null, loading: false });

  useEffect(() => {
    loadData();
  }, [id]);

  const loadData = async () => {
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
  };

  const handleViewVersion = async (versionId) => {
    try {
      const content = await getVersionContent(versionId);
      setViewDialog({ open: true, content, versionId });
    } catch (error) {
      console.error('Failed to load version content:', error);
    }
  };

  const handleDownloadVersion = async (versionId) => {
    try {
      const content = await getVersionContent(versionId);
      const blob = new Blob([content], { type: 'text/plain' });
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `config_v${versionId}.txt`;
      a.click();
      URL.revokeObjectURL(url);
    } catch (error) {
      console.error('Failed to download version:', error);
    }
  };

  const handleCompare = async () => {
    if (!selectedVersions.left || !selectedVersions.right) return;
    
    try {
      setCompareDialog({ open: true, diffData: null, loading: true });
      const diff = await getVersionDiff(selectedVersions.left, selectedVersions.right);
      setCompareDialog({ open: true, diffData: diff, loading: false });
    } catch (error) {
      console.error('Failed to compare versions:', error);
      setCompareDialog({ open: false, diffData: null, loading: false });
    }
  };

  const closeCompareDialog = () => {
    setCompareDialog({ open: false, diffData: null, loading: false });
    setSelectedVersions({ left: null, right: null });
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
                    {formatDateTime(v.created_at)} ({v.storage_type})
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
                    {formatDateTime(v.created_at)} ({v.storage_type})
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
                  <TableCell>ID</TableCell>
                  <TableCell>Дата</TableCell>
                  <TableCell>Тип</TableCell>
                  <TableCell>Размер</TableCell>
                  <TableCell>Хэш</TableCell>
                  <TableCell align="right">Действия</TableCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {versions.map((version) => (
                  <TableRow key={version.id} hover>
                    <TableCell>{version.id}</TableCell>
                    <TableCell>{formatDateTime(version.created_at)}</TableCell>
                    <TableCell>
                      <Chip
                        label={version.storage_type === 'base' ? 'База' : 'Diff'}
                        size="small"
                        color={version.storage_type === 'base' ? 'primary' : 'secondary'}
                      />
                    </TableCell>
                    <TableCell>{version.original_size ? `${version.original_size} байт` : '-'}</TableCell>
                    <TableCell>
                      <Typography variant="body2" sx={{ fontFamily: 'monospace', fontSize: '0.75rem' }}>
                        {version.version_hash?.substring(0, 12)}...
                      </Typography>
                    </TableCell>
                    <TableCell align="right">
                      <IconButton size="small" onClick={() => handleViewVersion(version.id)}>
                        <VisibilityIcon fontSize="small" />
                      </IconButton>
                      <IconButton size="small" onClick={() => handleDownloadVersion(version.id)}>
                        <DownloadIcon fontSize="small" />
                      </IconButton>
                    </TableCell>
                  </TableRow>
                ))}
                {versions.length === 0 && (
                  <TableRow>
                    <TableCell colSpan={6} align="center" sx={{ py: 4 }}>
                      <Typography color="text.secondary">Нет доступных версий</Typography>
                    </TableCell>
                  </TableRow>
                )}
              </TableBody>
            </Table>
          </TableContainer>
        </CardContent>
      </Card>

      {/* View Dialog */}
      <Dialog
        open={viewDialog.open}
        onClose={() => setViewDialog({ open: false, content: '', versionId: null })}
        maxWidth="lg"
        fullWidth
      >
        <DialogContent sx={{ p: 0 }}>
          <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', p: 2, borderBottom: 1, borderColor: 'divider' }}>
            <Typography variant="h6">Версия {viewDialog.versionId}</Typography>
            <IconButton onClick={() => setViewDialog({ open: false, content: '', versionId: null })}>
              <CloseIcon />
            </IconButton>
          </Box>
          <Box
            component="pre"
            sx={{
              p: 2,
              m: 0,
              maxHeight: '70vh',
              overflow: 'auto',
              bgcolor: 'background.default',
              fontFamily: 'monospace',
              fontSize: '0.875rem',
              whiteSpace: 'pre-wrap',
              wordBreak: 'break-all',
            }}
          >
            {viewDialog.content}
          </Box>
        </DialogContent>
      </Dialog>

      {/* Compare Dialog */}
      <Dialog
        open={compareDialog.open}
        onClose={closeCompareDialog}
        maxWidth="xl"
        fullWidth
      >
        <DialogContent sx={{ p: 0 }}>
          <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', p: 2, borderBottom: 1, borderColor: 'divider' }}>
            <Typography variant="h6">Сравнение версий</Typography>
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
              <ChangesTab embedded initialDiffData={compareDialog.diffData} />
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
