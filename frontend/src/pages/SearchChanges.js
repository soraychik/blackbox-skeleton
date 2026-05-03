import React, { useState } from 'react';
import { Link } from 'react-router-dom';
import {
  Alert,
  Box,
  Button,
  Card,
  CardContent,
  Chip,
  CircularProgress,
  Collapse,
  Dialog,
  DialogContent,
  Grid,
  IconButton,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TablePagination,
  TableRow,
  TextField,
  Typography,
  useMediaQuery,
  useTheme,
} from '@mui/material';
import {
  Search as SearchIcon,
  Close as CloseIcon,
  KeyboardArrowDown as KeyboardArrowDownIcon,
  KeyboardArrowUp as KeyboardArrowUpIcon,
  FileDownload as FileDownloadIcon,
} from '@mui/icons-material';
import { searchChanges, getVersionDiff } from '../utils/api';
import ChangesTab from '../components/ChangesTab';

/**
 * Поиск устройств, у которых конфиг изменился по множественному условию
 * (добавились/удалились строки по шаблонам).
 */
const SearchChanges = () => {
  const theme = useTheme();
  const isMobile = useMediaQuery(theme.breakpoints.down('sm'));
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(null);
  const [params, setParams] = useState({
    added_patterns: '',
    removed_patterns: '',
    from_date: '',
    to_date: '',
  });
  const [results, setResults] = useState(null);
  const [page, setPage] = useState(0);
  const [rowsPerPage, setRowsPerPage] = useState(10);
  const [diffDialog, setDiffDialog] = useState({
    open: false,
    loading: false,
    data: null,
    deviceId: null,
    deviceName: null,
    version1Date: null,
    version2Date: null,
  });
  const [expandedRows, setExpandedRows] = useState({});

  const toggleRow = (deviceId) => {
    setExpandedRows((prev) => ({
      ...prev,
      [deviceId]: !prev[deviceId],
    }));
  };

  const handleChangePage = (event, newPage) => {
    setExpandedRows({});
    setPage(newPage);
  };

  const handleChangeRowsPerPage = (event) => {
    setExpandedRows({});
    setRowsPerPage(parseInt(event.target.value, 10));
    setPage(0);
  };

  const handleSearch = async () => {
    const added = params.added_patterns
      .split('\n')
      .map((s) => s.trim())
      .filter(Boolean);
    const removed = params.removed_patterns
      .split('\n')
      .map((s) => s.trim())
      .filter(Boolean);
    if (added.length === 0 && removed.length === 0) {
      return;
    }
    try {
      setLoading(true);
      setError(null);
      const data = await searchChanges({
        added_patterns: added,
        removed_patterns: removed,
        from_date: params.from_date || undefined,
        to_date: params.to_date || undefined,
      });
      setResults(data.devices || []);
    } catch {
      setError('Не удалось выполнить поиск по изменениям');
      setResults([]);
    } finally {
      setLoading(false);
    }
  };

  const handleShowDiff = async (leftVersionId, rightVersionId, deviceName, version1Date, version2Date) => {
    setDiffDialog({ open: true, loading: true, data: null, deviceId: null, deviceName, version1Date, version2Date });
    try {
      const diffData = await getVersionDiff(leftVersionId, rightVersionId);
      setDiffDialog({ open: true, loading: false, data: diffData, deviceId: null, deviceName, version1Date, version2Date });
    } catch {
      setError('Не удалось загрузить сравнение версий');
      setDiffDialog({ open: false, loading: false, data: null, deviceId: null, deviceName: null, version1Date: null, version2Date: null });
    }
  };

  const closeDiffDialog = () => {
    setDiffDialog({ open: false, loading: false, data: null, deviceId: null, deviceName: null, version1Date: null, version2Date: null });
  };

  const handleExportCSV = () => {
    if (!results || results.length === 0) return;
    const rows = [['hostname', 'mgmt_ip', 'vendor', 'model', 'date_from', 'date_to', 'added_lines', 'removed_lines']];
    for (const device of results) {
      for (const ch of (device.changes || [])) {
        rows.push([
          device.hostname, device.mgmt_ip || '', device.vendor || '', device.model || '',
          ch.left_date || '', ch.right_date || '', ch.added_count || 0, ch.removed_count || 0,
        ]);
      }
    }
    const csv = rows.map(r => r.map(v => `"${String(v).replace(/"/g, '""')}"`).join(',')).join('\n');
    const blob = new Blob(['﻿' + csv], { type: 'text/csv;charset=utf-8' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `changes-${new Date().toISOString().slice(0, 10)}.csv`;
    a.click();
    URL.revokeObjectURL(url);
  };

  const handleExportJSON = () => {
    if (!results || results.length === 0) return;
    const blob = new Blob([JSON.stringify(results, null, 2)], { type: 'application/json' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `changes-${new Date().toISOString().slice(0, 10)}.json`;
    a.click();
    URL.revokeObjectURL(url);
  };

  return (
    <Box>
      <Typography variant="h4" fontWeight={600} gutterBottom>
        Поиск по изменениям
      </Typography>
      <Typography variant="body1" color="text.secondary" sx={{ mb: 4 }}>
        Найти устройства, у которых конфиг изменился по условию: добавились и/или удалились строки по шаблонам (regex)
      </Typography>

      {error && (
        <Alert severity="error" onClose={() => setError(null)} sx={{ mb: 2 }}>
          {error}
        </Alert>
      )}

      <Card sx={{ mb: 4 }}>
        <CardContent>
          <Grid container spacing={3}>
            <Grid item xs={12}>
              <TextField
                fullWidth
                multiline
                minRows={2}
                label="Шаблоны добавленных строк (regex, по одному на строку)"
                placeholder="например:^ip route\n^interface"
                value={params.added_patterns}
                onChange={(e) =>
                  setParams((prev) => ({ ...prev, added_patterns: e.target.value }))
                }
              />
            </Grid>
            <Grid item xs={12}>
              <TextField
                fullWidth
                multiline
                minRows={2}
                label="Шаблоны удалённых строк (regex, по одному на строку)"
                placeholder="например:^no ip"
                value={params.removed_patterns}
                onChange={(e) =>
                  setParams((prev) => ({ ...prev, removed_patterns: e.target.value }))
                }
              />
            </Grid>
            <Grid item xs={12} sm={6}>
              <TextField
                fullWidth
                type="date"
                label="Дата от (необязательно)"
                InputLabelProps={{ shrink: true }}
                value={params.from_date}
                onChange={(e) =>
                  setParams((prev) => ({ ...prev, from_date: e.target.value }))
                }
              />
            </Grid>
            <Grid item xs={12} sm={6}>
              <TextField
                fullWidth
                type="date"
                label="Дата до (необязательно)"
                InputLabelProps={{ shrink: true }}
                value={params.to_date}
                onChange={(e) =>
                  setParams((prev) => ({ ...prev, to_date: e.target.value }))
                }
              />
            </Grid>
            <Grid item xs={12}>
              <Button
                variant="contained"
                startIcon={loading ? <CircularProgress size={20} color="inherit" /> : <SearchIcon />}
                onClick={handleSearch}
                disabled={loading || (!params.added_patterns.trim() && !params.removed_patterns.trim())}
              >
                {loading ? 'Поиск...' : 'Найти устройства'}
              </Button>
            </Grid>
          </Grid>
        </CardContent>
      </Card>

      {results && (
        <Card>
          <CardContent>
            <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', mb: 1 }}>
              <Typography variant="h6" fontWeight={600}>
                Результаты: устройств с подходящими изменениями — {results.length}
              </Typography>
              {results.length > 0 && (
                <Box sx={{ display: 'flex', gap: 1 }}>
                  <Button size="small" variant="outlined" startIcon={<FileDownloadIcon />} onClick={handleExportCSV}>
                    CSV
                  </Button>
                  <Button size="small" variant="outlined" startIcon={<FileDownloadIcon />} onClick={handleExportJSON}>
                    JSON
                  </Button>
                </Box>
              )}
            </Box>
            <TableContainer sx={{ overflowX: 'auto' }}>
              <Table sx={{ minWidth: 360 }}>
                <TableHead>
                  <TableRow>
                    <TableCell sx={{ width: 50 }} />
                    <TableCell>Устройство</TableCell>
                    <TableCell sx={{ display: { xs: 'none', sm: 'table-cell' } }}>IP</TableCell>
                    <TableCell sx={{ display: { xs: 'none', md: 'table-cell' } }}>Вендор</TableCell>
                    <TableCell sx={{ display: { xs: 'none', md: 'table-cell' } }}>Модель</TableCell>
                    <TableCell align="right">Изменений</TableCell>
                  </TableRow>
                </TableHead>
                <TableBody>
                  {results.slice(page * rowsPerPage, page * rowsPerPage + rowsPerPage).map((row) => (
                    <React.Fragment key={row.device_id}>
                      <TableRow
                        hover
                        onClick={() => toggleRow(row.device_id)}
                        sx={{ cursor: 'pointer' }}
                      >
                        <TableCell>
                          {expandedRows[row.device_id] ? (
                            <KeyboardArrowUpIcon />
                          ) : (
                            <KeyboardArrowDownIcon />
                          )}
                        </TableCell>
                        <TableCell>
                          <Typography
                            fontWeight={500}
                            component={Link}
                            to={`/devices/${row.device_id}`}
                            onClick={(e) => e.stopPropagation()}
                            sx={{
                              color: 'primary.main',
                              textDecoration: 'underline',
                              '&:hover': { color: 'primary.dark' },
                            }}
                          >
                            {row.hostname}
                          </Typography>
                        </TableCell>
                        <TableCell sx={{ display: { xs: 'none', sm: 'table-cell' } }}>{row.mgmt_ip || '-'}</TableCell>
                        <TableCell sx={{ display: { xs: 'none', md: 'table-cell' } }}>{row.vendor || '-'}</TableCell>
                        <TableCell sx={{ display: { xs: 'none', md: 'table-cell' } }}>{row.model || '-'}</TableCell>
                        <TableCell align="right">
                          <Chip
                            size="small"
                            label={row.changes?.length || 0}
                            color="primary"
                            variant="outlined"
                          />
                        </TableCell>
                      </TableRow>
                      <TableRow>
                        <TableCell colSpan={6} sx={{ py: 0, border: 0 }}>
                          <Collapse in={expandedRows[row.device_id]} timeout="auto" unmountOnExit>
                            <Box sx={{ bgcolor: 'action.hover', px: 2, py: 1 }}>
                              {row.changes?.map((ch, idx) => (
                                <Box
                                  key={idx}
                                  onClick={() => handleShowDiff(ch.left_version_id, ch.right_version_id, row.hostname, ch.left_date, ch.right_date)}
                                  sx={{
                                    display: 'flex',
                                    alignItems: 'center',
                                    gap: 2,
                                    py: 1,
                                    cursor: 'pointer',
                                    borderBottom: idx < row.changes.length - 1 ? '1px solid' : 'none',
                                    borderColor: 'divider',
                                    '&:hover': { bgcolor: 'action.selected' },
                                    borderRadius: 1,
                                    px: 1,
                                  }}
                                >
                                  <Typography variant="body2" sx={{ minWidth: 200 }}>
                                    {ch.left_date} → {ch.right_date}
                                  </Typography>
                                  <Chip size="small" label={`+${ch.added_count}`} color="success" variant="outlined" />
                                  <Chip size="small" label={`-${ch.removed_count}`} color="error" variant="outlined" />
                                  <Typography variant="body2" color="text.secondary" sx={{ ml: 'auto' }}>
                                    Нажмите для просмотра сравнения →
                                  </Typography>
                                </Box>
                              ))}
                            </Box>
                          </Collapse>
                        </TableCell>
                      </TableRow>
                    </React.Fragment>
                  ))}
                  {results.length === 0 && (
                    <TableRow>
                      <TableCell colSpan={6} align="center" sx={{ py: 4 }}>
                        <Typography color="text.secondary">
                          Нет устройств с изменениями по заданным шаблонам
                        </Typography>
                      </TableCell>
                    </TableRow>
                  )}
                </TableBody>
              </Table>
            </TableContainer>
            <TablePagination
              component="div"
              count={results.length}
              page={page}
              onPageChange={handleChangePage}
              rowsPerPage={rowsPerPage}
              onRowsPerPageChange={handleChangeRowsPerPage}
              rowsPerPageOptions={[10, 25, 50, 100]}
            />
          </CardContent>
        </Card>
      )}

      <Dialog open={diffDialog.open} onClose={closeDiffDialog} maxWidth="xl" fullWidth fullScreen={isMobile}>
        <DialogContent sx={{ p: 0 }}>
          <Box
            sx={{
              display: 'flex',
              justifyContent: 'space-between',
              alignItems: 'center',
              p: 2,
              borderBottom: 1,
              borderColor: 'divider',
            }}
          >
            <Typography variant="h6">Сравнение версий</Typography>
            <IconButton onClick={closeDiffDialog}>
              <CloseIcon />
            </IconButton>
          </Box>
          <Box sx={{ p: 2 }}>
            {diffDialog.loading ? (
              <Box sx={{ display: 'flex', justifyContent: 'center', py: 8 }}>
                <CircularProgress />
              </Box>
            ) : diffDialog.data ? (
              <ChangesTab 
                embedded 
                initialDiffData={diffDialog.data}
                deviceName={diffDialog.deviceName}
                version1Date={diffDialog.version1Date}
                version2Date={diffDialog.version2Date}
              />
            ) : null}
          </Box>
        </DialogContent>
      </Dialog>
    </Box>
  );
};

export default SearchChanges;
