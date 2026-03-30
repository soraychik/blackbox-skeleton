import React, { useState, useEffect, useRef } from 'react';
import { useNavigate } from 'react-router-dom';
import {
  Alert,
  Box,
  Button,
  Card,
  CardContent,
  CircularProgress,
  Grid,
  Skeleton,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Typography,
  Chip,
} from '@mui/material';
import {
  Devices as DevicesIcon,
  Update as UpdateIcon,
  ChangeCircle as ChangeCircleIcon,
  PlayArrow as PlayArrowIcon,
} from '@mui/icons-material';
import { getDashboardStats, triggerScan, getScanStatus } from '../utils/api';
import { formatDateTime } from '../utils/dateFormatter';

const StatCard = ({ title, value, icon, color, loading }) => (
  <Card sx={{ height: '100%' }}>
    <CardContent sx={{ display: 'flex', alignItems: 'center', gap: 2 }}>
      <Box
        sx={{
          width: 56,
          height: 56,
          borderRadius: 2,
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          bgcolor: `${color}.main`,
          color: 'white',
        }}
      >
        {icon}
      </Box>
      <Box>
        <Typography variant="body2" color="text.secondary">
          {title}
        </Typography>
        {loading ? (
          <Skeleton width={60} />
        ) : (
          <Typography variant="h4" fontWeight={600}>
            {value}
          </Typography>
        )}
      </Box>
    </CardContent>
  </Card>
);

const Dashboard = () => {
  const navigate = useNavigate();
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [scanning, setScanning] = useState(false);
  const lastFinishedAtRef = useRef(null);
  const [stats, setStats] = useState({
    totalDevices: 0,
    updatedToday: 0,
    devicesWithChanges: 0,
  });
  const [topDevices, setTopDevices] = useState([]);

  useEffect(() => {
    loadData();
    pollScanStatus();

    const timer = setInterval(() => {
      pollScanStatus();
    }, 3000);

    return () => clearInterval(timer);
  }, []);

  const loadData = async () => {
    try {
      setLoading(true);
      const data = await getDashboardStats();

      setStats({
        totalDevices: data.total_devices ?? 0,
        updatedToday: data.updated_today ?? 0,
        devicesWithChanges: data.devices_with_changes ?? 0,
      });

      setTopDevices(
        (data.top_devices || []).map((t) => ({
          deviceId: t.device_id,
          hostname: t.hostname,
          changeCount: t.change_count,
          lastChange: t.last_change,
        }))
      );
    } catch {
      setError('Не удалось загрузить данные дашборда');
    } finally {
      setLoading(false);
    }
  };

  const handleTriggerScan = async () => {
    try {
      setScanning(true);
      setError(null);
      await triggerScan();
      await loadData();
    } catch (err) {
      setError(err.response?.data?.error || 'Не удалось запустить сканирование');
      setScanning(false);
    }
  };

  const pollScanStatus = async () => {
    try {
      const status = await getScanStatus();
      const inProgress = Boolean(status?.in_progress);
      const finishedAt = status?.last_finished_at || null;

      if (finishedAt && finishedAt !== lastFinishedAtRef.current) {
        const hadPreviousFinish = Boolean(lastFinishedAtRef.current);
        lastFinishedAtRef.current = finishedAt;
        if (hadPreviousFinish) {
          await loadData();
        }
      }

      setScanning(inProgress);
    } catch {
      // Не прерываем интерфейс из-за временной ошибки статуса сканирования.
    }
  };

  return (
    <Box>
      <Typography variant="h4" fontWeight={600} gutterBottom>
        Дашборд
      </Typography>
      <Typography variant="body1" color="text.secondary" sx={{ mb: 4 }}>
        Обзор состояния системы
      </Typography>

      {error && (
        <Alert severity="error" onClose={() => setError(null)} sx={{ mb: 3 }}>
          {error}
        </Alert>
      )}

      <Grid container spacing={3} sx={{ mb: 4 }}>
        <Grid item xs={12} sm={6} md={4}>
          <StatCard
            title="Всего устройств"
            value={stats.totalDevices}
            icon={<DevicesIcon />}
            color="primary"
            loading={loading}
          />
        </Grid>
        <Grid item xs={12} sm={6} md={4}>
          <StatCard
            title="Обновлено сегодня"
            value={stats.updatedToday}
            icon={<UpdateIcon />}
            color="success"
            loading={loading}
          />
        </Grid>
        <Grid item xs={12} sm={6} md={4}>
          <StatCard
            title="Устройств с изменениями"
            value={stats.devicesWithChanges}
            icon={<ChangeCircleIcon />}
            color="warning"
            loading={loading}
          />
        </Grid>
      </Grid>

      <Box sx={{ mb: 3 }}>
        <Button
          variant="contained"
          color="primary"
          startIcon={scanning ? <CircularProgress size={20} color="inherit" /> : <PlayArrowIcon />}
          onClick={handleTriggerScan}
          disabled={scanning}
        >
          {scanning ? 'Сканирование…' : 'Запустить сканирование'}
        </Button>
      </Box>

      <Card>
        <CardContent>
          <Typography variant="h6" fontWeight={600} gutterBottom>
            Топ устройств по изменениям
          </Typography>
          <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
            Устройства с наибольшим количеством изменений конфигурации
          </Typography>

          {loading ? (
            <Box sx={{ display: 'flex', justifyContent: 'center', py: 4 }}>
              <CircularProgress />
            </Box>
          ) : (
            <TableContainer>
              <Table>
                <TableHead>
                  <TableRow>
                    <TableCell sx={{ width: '80%' }}>Устройство</TableCell>
                    <TableCell align="center">Изменений</TableCell>
                    <TableCell align="right">Последнее изменение</TableCell>
                  </TableRow>
                </TableHead>
                <TableBody>
                  {topDevices.map((item) => (
                    <TableRow
                      key={item.deviceId}
                      hover
                      sx={{ cursor: 'pointer' }}
                      onClick={() => navigate(`/devices/${item.deviceId}`)}
                    >
                      <TableCell>
                        <Typography fontWeight={500}>{item.hostname}</Typography>
                      </TableCell>
                      <TableCell align="center">
                        <Chip label={item.changeCount} color="primary" size="small" />
                      </TableCell>
                      <TableCell align="right">
                        {item.lastChange ? formatDateTime(item.lastChange) : '-'}
                      </TableCell>
                    </TableRow>
                  ))}
                  {topDevices.length === 0 && (
                    <TableRow>
                      <TableCell colSpan={3} align="center" sx={{ py: 4 }}>
                        <Typography color="text.secondary">
                          Нет данных об изменениях
                        </Typography>
                      </TableCell>
                    </TableRow>
                  )}
                </TableBody>
              </Table>
            </TableContainer>
          )}
        </CardContent>
      </Card>
    </Box>
  );
};

export default Dashboard;