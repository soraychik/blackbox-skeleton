import React, { useState, useEffect, useCallback } from 'react';
import {
  Box,
  Card,
  CardContent,
  Chip,
  CircularProgress,
  FormControl,
  InputLabel,
  MenuItem,
  Pagination,
  Select,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  TextField,
  Typography,
} from '@mui/material';
import { getAuditLog } from '../utils/api';

const ACTION_LABELS = {
  login:           'Вход в систему',
  view_config:     'Просмотр конфига',
  export_config:   'Выгрузка конфига',
  view_diff:       'Сравнение версий',
  view_diff_date:  'Сравнение по датам',
  compare_devices: 'Сравнение устройств',
  search:          'Поиск',
  search_changes:  'Поиск по изменениям',
  trigger_scan:    'Запуск сканирования',
  update_settings: 'Изменение настроек',
};

const ROLE_LABELS = {
  admin:    'Администратор',
  engineer: 'Инженер',
  operator: 'Оператор',
};

const ACTION_COLORS = {
  login:           'default',
  view_config:     'info',
  export_config:   'success',
  view_diff:       'info',
  view_diff_date:  'info',
  compare_devices: 'info',
  search:          'warning',
  search_changes:  'warning',
  trigger_scan:    'error',
  update_settings: 'error',
};

const formatDetails = (details) => {
  if (!details || Object.keys(details).length === 0) return '—';
  return Object.entries(details)
    .map(([k, v]) => {
      if (Array.isArray(v)) return `${k}: [${v.join(', ')}]`;
      return `${k}: ${v}`;
    })
    .join(' · ');
};

const LIMIT = 50;

const AuditLog = () => {
  const [entries, setEntries] = useState([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [loading, setLoading] = useState(false);

  const [filterUsername, setFilterUsername] = useState('');
  const [filterAction, setFilterAction] = useState('');
  const [filterFrom, setFilterFrom] = useState('');
  const [filterTo, setFilterTo] = useState('');

  const load = useCallback(() => {
    setLoading(true);
    const params = { page, limit: LIMIT };
    if (filterUsername) params.username = filterUsername;
    if (filterAction)   params.action   = filterAction;
    if (filterFrom)     params.from     = filterFrom;
    if (filterTo)       params.to       = filterTo;

    getAuditLog(params)
      .then((data) => {
        setEntries(data.entries || []);
        setTotal(data.total || 0);
      })
      .finally(() => setLoading(false));
  }, [page, filterUsername, filterAction, filterFrom, filterTo]);

  useEffect(() => { load(); }, [load]);

  const handleFilterChange = (setter) => (e) => {
    setter(e.target.value);
    setPage(1);
  };

  return (
    <Box>
      <Typography variant="h5" fontWeight={700} mb={3}>
        Журнал аудита
      </Typography>

      <Card sx={{ mb: 3 }}>
        <CardContent>
          <Box sx={{ display: 'flex', gap: 2, flexWrap: 'wrap', alignItems: 'flex-end' }}>
            <TextField
              label="Пользователь"
              size="small"
              value={filterUsername}
              onChange={handleFilterChange(setFilterUsername)}
              sx={{ width: 180 }}
            />
            <FormControl size="small" sx={{ width: 220 }}>
              <InputLabel>Действие</InputLabel>
              <Select
                value={filterAction}
                label="Действие"
                onChange={handleFilterChange(setFilterAction)}
              >
                <MenuItem value="">Все</MenuItem>
                {Object.entries(ACTION_LABELS).map(([val, label]) => (
                  <MenuItem key={val} value={val}>{label}</MenuItem>
                ))}
              </Select>
            </FormControl>
            <TextField
              label="С"
              type="date"
              size="small"
              value={filterFrom}
              onChange={handleFilterChange(setFilterFrom)}
              InputLabelProps={{ shrink: true }}
              sx={{ width: 160 }}
            />
            <TextField
              label="По"
              type="date"
              size="small"
              value={filterTo}
              onChange={handleFilterChange(setFilterTo)}
              InputLabelProps={{ shrink: true }}
              sx={{ width: 160 }}
            />
          </Box>
        </CardContent>
      </Card>

      <Card>
        <TableContainer sx={{ overflowX: 'auto' }}>
          <Table size="small" sx={{ minWidth: 360 }}>
            <TableHead>
              <TableRow>
                <TableCell sx={{ fontWeight: 600, whiteSpace: 'nowrap' }}>Дата и время</TableCell>
                <TableCell sx={{ fontWeight: 600 }}>Пользователь</TableCell>
                <TableCell sx={{ fontWeight: 600, display: { xs: 'none', sm: 'table-cell' } }}>Роль</TableCell>
                <TableCell sx={{ fontWeight: 600 }}>Действие</TableCell>
                <TableCell sx={{ fontWeight: 600, display: { xs: 'none', md: 'table-cell' } }}>Детали</TableCell>
                <TableCell sx={{ fontWeight: 600, display: { xs: 'none', sm: 'table-cell' } }}>IP</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {loading ? (
                <TableRow>
                  <TableCell colSpan={6} align="center" sx={{ py: 4 }}>
                    <CircularProgress size={28} />
                  </TableCell>
                </TableRow>
              ) : entries.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={6} align="center" sx={{ py: 4, color: 'text.secondary' }}>
                    Записей не найдено
                  </TableCell>
                </TableRow>
              ) : entries.map((e) => (
                <TableRow key={e.id} hover>
                  <TableCell sx={{ whiteSpace: 'nowrap', fontFamily: 'monospace', fontSize: 12 }}>
                    {new Date(e.created_at).toLocaleString('ru-RU')}
                  </TableCell>
                  <TableCell>{e.username}</TableCell>
                  <TableCell sx={{ display: { xs: 'none', sm: 'table-cell' } }}>
                    <Chip label={ROLE_LABELS[e.role] || e.role} size="small" variant="outlined" sx={{ fontSize: 11 }} />
                  </TableCell>
                  <TableCell>
                    <Chip label={ACTION_LABELS[e.action] || e.action} size="small" color={ACTION_COLORS[e.action] || 'default'} />
                  </TableCell>
                  <TableCell sx={{ fontSize: 12, color: 'text.secondary', maxWidth: 320, display: { xs: 'none', md: 'table-cell' } }}>
                    {formatDetails(e.details)}
                  </TableCell>
                  <TableCell sx={{ fontFamily: 'monospace', fontSize: 12, display: { xs: 'none', sm: 'table-cell' } }}>{e.ip_address}</TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </TableContainer>
        {total > LIMIT && (
          <Box sx={{ display: 'flex', justifyContent: 'center', p: 2 }}>
            <Pagination
              count={Math.ceil(total / LIMIT)}
              page={page}
              onChange={(_, v) => setPage(v)}
              color="primary"
            />
          </Box>
        )}
      </Card>
    </Box>
  );
};

export default AuditLog;
