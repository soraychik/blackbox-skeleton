import React, { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import {
  Box,
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
} from '@mui/icons-material';
import { getDevices, getVersions } from '../utils/api';
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
  const [devices, setDevices] = useState([]);
  const [versions, setVersions] = useState([]);
  const [stats, setStats] = useState({
    totalDevices: 0,
    updatedToday: 0,
    devicesWithChanges: 0,
  });
  const [topDevices, setTopDevices] = useState([]);

  useEffect(() => {
    loadData();
  }, []);

  const loadData = async () => {
    try {
      setLoading(true);
      const [devicesData, versionsData] = await Promise.all([
        getDevices(),
        getVersions(),
      ]);

      setDevices(devicesData);
      setVersions(versionsData);

      // Calculate stats
      const today = new Date().toDateString();
      const updatedToday = versionsData.filter(
        (v) => new Date(v.created_at).toDateString() === today
      ).length;

      const deviceChangeCounts = {};
      versionsData.forEach((v) => {
        if (!deviceChangeCounts[v.device_id]) {
          deviceChangeCounts[v.device_id] = 0;
        }
        deviceChangeCounts[v.device_id] += 1;
      });

      const devicesWithChanges = Object.keys(deviceChangeCounts).length;

      setStats({
        totalDevices: devicesData.length,
        updatedToday,
        devicesWithChanges,
      });

      // Get top devices by changes
      const top = Object.entries(deviceChangeCounts)
        .map(([deviceId, count]) => {
          const device = devicesData.find((d) => d.id === parseInt(deviceId));
          const latestVersion = versionsData
            .filter((v) => v.device_id === parseInt(deviceId))
            .sort((a, b) => new Date(b.created_at) - new Date(a.created_at))[0];
          return {
            deviceId: parseInt(deviceId),
            hostname: device?.hostname || `Device ${deviceId}`,
            changeCount: count,
            lastChange: latestVersion?.created_at,
          };
        })
        .sort((a, b) => b.changeCount - a.changeCount)
        .slice(0, 5);

      setTopDevices(top);
    } catch (error) {
      console.error('Failed to load dashboard data:', error);
    } finally {
      setLoading(false);
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
