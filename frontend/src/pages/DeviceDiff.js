import React, { useState, useEffect } from 'react';
import {
  Alert,
  Box,
  Button,
  Card,
  CardContent,
  CircularProgress,
  Grid,
  TextField,
  Typography,
  Autocomplete,
  Dialog,
  DialogContent,
  IconButton,
} from '@mui/material';
import {
  Compare as CompareIcon,
  Close as CloseIcon,
} from '@mui/icons-material';
import { getDevices, getDeviceVersions, getVersionDiff } from '../utils/api';
import { formatDateTime } from '../utils/dateFormatter';
import ChangesTab from '../components/ChangesTab';

const DeviceDiff = () => {
  const [loading, setLoading] = useState(true);
  const [devices, setDevices] = useState([]);
  const [leftDevice, setLeftDevice] = useState(null);
  const [rightDevice, setRightDevice] = useState(null);
  const [compareLoading, setCompareLoading] = useState(false);
  const [diffData, setDiffData] = useState(null);
  const [dialogOpen, setDialogOpen] = useState(false);
  const [error, setError] = useState(null);
  const [versions1, setVersions1] = useState(null);
  const [versions2, setVersions2] = useState(null);

  useEffect(() => {
    loadDevices();
  }, []);

  const loadDevices = async () => {
    try {
      setLoading(true);
      setError(null);
      const data = await getDevices();
      setDevices(data);
    } catch {
      setError('Не удалось загрузить список устройств');
    } finally {
      setLoading(false);
    }
  };

  const handleCompare = async () => {
    if (!leftDevice || !rightDevice) return;

    try {
      setCompareLoading(true);
      setError(null);
      // API сравнения принимает id версий, а не устройств — берём последнюю версию каждого устройства
      const [data1, data2] = await Promise.all([
        getDeviceVersions(leftDevice.id),
        getDeviceVersions(rightDevice.id),
      ]);
      const versions1 = data1?.versions || [];
      const versions2 = data2?.versions || [];
      if (versions1.length === 0) {
        setError(`У устройства «${leftDevice.hostname}» нет сохранённых версий конфигурации`);
        return;
      }
      if (versions2.length === 0) {
        setError(`У устройства «${rightDevice.hostname}» нет сохранённых версий конфигурации`);
        return;
      }
      const leftVersionId = versions1[0].id;
      const rightVersionId = versions2[0].id;
      const diff = await getVersionDiff(leftVersionId, rightVersionId);
      setVersions1(versions1);
      setVersions2(versions2);
      setDiffData(diff);
      setDialogOpen(true);
    } catch (error) {
      const msg = error.response?.data?.error || error.message;
      setError(msg || 'Не удалось выполнить сравнение. Проверьте соединение с сервером');
    } finally {
      setCompareLoading(false);
    }
  };

  const closeDialog = () => {
    setDialogOpen(false);
    setDiffData(null);
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
      <Typography variant="h4" fontWeight={600} gutterBottom>
        Сравнение устройств
      </Typography>
      <Typography variant="body1" color="text.secondary" sx={{ mb: 4 }}>
        Сравните конфигурации двух разных устройств
      </Typography>

      {error && (
        <Alert severity="error" onClose={() => setError(null)} sx={{ mb: 2 }}>
          {error}
        </Alert>
      )}

      <Card>
        <CardContent>
          <Grid container spacing={3} alignItems="center">
            <Grid item xs={12} md={5}>
              <Autocomplete
                options={devices}
                getOptionLabel={(option) => option.hostname}
                value={leftDevice}
                onChange={(e, value) => setLeftDevice(value)}
                renderInput={(params) => (
                  <TextField {...params} label="Устройство 1 (левое)" fullWidth />
                )}
              />
            </Grid>
            <Grid item xs={12} md={2} sx={{ textAlign: 'center' }}>
              <CompareIcon color="action" sx={{ fontSize: 32 }} />
            </Grid>
            <Grid item xs={12} md={5}>
              <Autocomplete
                options={devices}
                getOptionLabel={(option) => option.hostname}
                value={rightDevice}
                onChange={(e, value) => setRightDevice(value)}
                renderInput={(params) => (
                  <TextField {...params} label="Устройство 2 (правое)" fullWidth />
                )}
              />
            </Grid>
            <Grid item xs={12}>
              <Button
                variant="contained"
                size="large"
                startIcon={<CompareIcon />}
                onClick={handleCompare}
                disabled={!leftDevice || !rightDevice || compareLoading}
              >
                {compareLoading ? 'Сравнение...' : 'Сравнить устройства'}
              </Button>
            </Grid>
          </Grid>
        </CardContent>
      </Card>

      <Dialog
        open={dialogOpen}
        onClose={closeDialog}
        maxWidth="xl"
        fullWidth
      >
        <DialogContent sx={{ p: 0 }}>
          <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', p: 2, borderBottom: 1, borderColor: 'divider' }}>
            <Typography variant="h6">
              {leftDevice?.hostname} vs {rightDevice?.hostname}
            </Typography>
            <IconButton onClick={closeDialog}>
              <CloseIcon />
            </IconButton>
          </Box>
          <Box sx={{ p: 2 }}>
            {diffData ? (
              <ChangesTab 
                embedded 
                initialDiffData={diffData}
                deviceName={`${leftDevice?.hostname} vs ${rightDevice?.hostname}`}
                version1Date={versions1?.[0]?.created_at ? formatDateTime(versions1[0].created_at) : null}
                version2Date={versions2?.[0]?.created_at ? formatDateTime(versions2[0].created_at) : null}
              />
            ) : (
              <Box sx={{ display: 'flex', justifyContent: 'center', py: 8 }}>
                <CircularProgress />
              </Box>
            )}
          </Box>
        </DialogContent>
      </Dialog>
    </Box>
  );
};

export default DeviceDiff;
