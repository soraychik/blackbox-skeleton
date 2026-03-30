import React, { useState, useEffect, useMemo } from 'react';
import { useNavigate } from 'react-router-dom';
import {
  Alert,
  Box,
  Card,
  CardContent,
  CircularProgress,
  Grid,
  TextField,
  Typography,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  TablePagination,
  InputAdornment,
  Chip,
  Button,
} from '@mui/material';
import {
  Search as SearchIcon,
  Clear as ClearIcon,
  FilterList as FilterListIcon,
} from '@mui/icons-material';
import { getDevices } from '../utils/api';

const Devices = () => {
  const navigate = useNavigate();
  const [loading, setLoading] = useState(true);
  const [devices, setDevices] = useState([]);
  const [error, setError] = useState(null);
  const [page, setPage] = useState(0);
  const [rowsPerPage, setRowsPerPage] = useState(25);
  const [filters, setFilters] = useState({
    search: '',
    vendor: '',
  });

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

  const filteredDevices = useMemo(() => {
    return devices.filter((device) => {
      const matchesSearch =
        !filters.search ||
        device.hostname?.toLowerCase().includes(filters.search.toLowerCase()) ||
        device.mgmt_ip?.toLowerCase().includes(filters.search.toLowerCase());

      const matchesVendor =
        !filters.vendor ||
        device.vendor?.toLowerCase().includes(filters.vendor.toLowerCase());

      return matchesSearch && matchesVendor;
    });
  }, [devices, filters]);

  const paginatedDevices = useMemo(() => {
    const start = page * rowsPerPage;
    return filteredDevices.slice(start, start + rowsPerPage);
  }, [filteredDevices, page, rowsPerPage]);

  const handleFilterChange = (field, value) => {
    setFilters((prev) => ({ ...prev, [field]: value }));
    setPage(0);
  };

  const clearFilters = () => {
    setFilters({ search: '', vendor: '' });
    setPage(0);
  };

  const handleRowClick = (deviceId) => {
    navigate(`/devices/${deviceId}`);
  };

  const hasActiveFilters = filters.search || filters.vendor;

  return (
    <Box>
      <Typography variant="h4" fontWeight={600} gutterBottom>
        Каталог устройств
      </Typography>
      <Typography variant="body1" color="text.secondary" sx={{ mb: 3 }}>
        {filteredDevices.length} устройств в системе
      </Typography>

      {error && (
        <Alert severity="error" onClose={() => setError(null)} sx={{ mb: 2 }}>
          {error}
        </Alert>
      )}

      <Card sx={{ mb: 3 }}>
        <CardContent sx={{ pb: '16px !important' }}>
          <Grid container spacing={2} alignItems="center">
            <Grid item xs={12} md={4}>
              <TextField
                fullWidth
                size="small"
                placeholder="Поиск по имени или IP..."
                value={filters.search}
                onChange={(e) => handleFilterChange('search', e.target.value)}
                InputProps={{
                  startAdornment: (
                    <InputAdornment position="start">
                      <SearchIcon color="action" />
                    </InputAdornment>
                  ),
                }}
              />
            </Grid>
            <Grid item xs={12} md={3}>
              <TextField
                fullWidth
                size="small"
                placeholder="Вендор (Cisco, Juniper...)"
                value={filters.vendor}
                onChange={(e) => handleFilterChange('vendor', e.target.value)}
              />
            </Grid>
            <Grid item xs={12} md={3}>
              <Button
                variant="outlined"
                startIcon={hasActiveFilters ? <ClearIcon /> : <FilterListIcon />}
                onClick={hasActiveFilters ? clearFilters : undefined}
              >
                {hasActiveFilters ? 'Очистить' : 'Фильтры'}
              </Button>
            </Grid>
          </Grid>
        </CardContent>
      </Card>

      <Card>
        {loading ? (
          <Box sx={{ display: 'flex', justifyContent: 'center', py: 8 }}>
            <CircularProgress />
          </Box>
        ) : (
          <>
            <TableContainer>
              <Table>
                <TableHead>
                  <TableRow>
                    <TableCell>Имя устройства</TableCell>
                    <TableCell>IP адрес</TableCell>
                    <TableCell>Вендор</TableCell>
                    <TableCell>Модель</TableCell>
                    <TableCell>Площадка</TableCell>
                    <TableCell>Теги</TableCell>
                  </TableRow>
                </TableHead>
                <TableBody>
                  {paginatedDevices.map((device) => (
                    <TableRow
                      key={device.id}
                      hover
                      sx={{ cursor: 'pointer' }}
                      onClick={() => handleRowClick(device.id)}
                    >
                      <TableCell>
                        <Typography fontWeight={500}>{device.hostname}</Typography>
                      </TableCell>
                      <TableCell>{device.mgmt_ip || '-'}</TableCell>
                      <TableCell>{device.vendor || '-'}</TableCell>
                      <TableCell>{device.model || '-'}</TableCell>
                      <TableCell>{device.location || '-'}</TableCell>
                      <TableCell>
                        {device.tags ? (
                          <Box sx={{ display: 'flex', gap: 0.5, flexWrap: 'wrap' }}>
                            {device.tags.split(',').map((tag, i) => (
                              <Chip key={i} label={tag.trim()} size="small" variant="outlined" />
                            ))}
                          </Box>
                        ) : (
                          '-'
                        )}
                      </TableCell>
                    </TableRow>
                  ))}
                  {paginatedDevices.length === 0 && (
                    <TableRow>
                      <TableCell colSpan={6} align="center" sx={{ py: 4 }}>
                        <Typography color="text.secondary">
                          Устройства не найдены
                        </Typography>
                      </TableCell>
                    </TableRow>
                  )}
                </TableBody>
              </Table>
            </TableContainer>
            <TablePagination
              component="div"
              count={filteredDevices.length}
              page={page}
              onPageChange={(e, newPage) => setPage(newPage)}
              rowsPerPage={rowsPerPage}
              onRowsPerPageChange={(e) => {
                setRowsPerPage(parseInt(e.target.value, 10));
                setPage(0);
              }}
              rowsPerPageOptions={[10, 25, 50, 100]}
              labelRowsPerPage="Строк на странице:"
              labelDisplayedRows={({ from, to, count }) => `${from}–${to} из ${count}`}
            />
          </>
        )}
      </Card>
    </Box>
  );
};

export default Devices;
