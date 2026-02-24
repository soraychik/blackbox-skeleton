import React, { useState } from 'react';
import { Link } from 'react-router-dom';
import {
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
  TableRow,
  TextField,
  Typography,
} from '@mui/material';
import {
  Search as SearchIcon,
  Close as CloseIcon,
  KeyboardArrowDown as KeyboardArrowDownIcon,
  KeyboardArrowUp as KeyboardArrowUpIcon,
} from '@mui/icons-material';
import { searchChanges, getVersionDiff } from '../utils/api';
import ChangesTab from '../components/ChangesTab';

/**
 * Поиск устройств, у которых конфиг изменился по множественному условию
 * (добавились/удалились строки по шаблонам). ТЗ 2.1, 2.3.
 */
const SearchChanges = () => {
  const [loading, setLoading] = useState(false);
  const [params, setParams] = useState({
    added_patterns: '',
    removed_patterns: '',
    from_date: '',
    to_date: '',
  });
  const [results, setResults] = useState(null);
  const [diffDialog, setDiffDialog] = useState({
    open: false,
    loading: false,
    data: null,
    deviceId: null,
  });
  const [expandedRows, setExpandedRows] = useState({});

  const toggleRow = (deviceId) => {
    setExpandedRows((prev) => ({
      ...prev,
      [deviceId]: !prev[deviceId],
    }));
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
      const data = await searchChanges({
        added_patterns: added,
        removed_patterns: removed,
        from_date: params.from_date || undefined,
        to_date: params.to_date || undefined,
      });
      setResults(data.devices || []);
    } catch (error) {
      console.error('Search changes failed:', error);
      setResults([]);
    } finally {
      setLoading(false);
    }
  };

  const handleShowDiff = async (leftVersionId, rightVersionId) => {
    setDiffDialog({ open: true, loading: true, data: null, deviceId: null });
    try {
      const diffData = await getVersionDiff(leftVersionId, rightVersionId);
      setDiffDialog({ open: true, loading: false, data: diffData, deviceId: null });
    } catch (error) {
      console.error('Failed to load diff:', error);
      setDiffDialog({ open: false, loading: false, data: null, deviceId: null });
    }
  };

  const closeDiffDialog = () => {
    setDiffDialog({ open: false, loading: false, data: null, deviceId: null });
  };

  return (
    <Box>
      <Typography variant="h4" fontWeight={600} gutterBottom>
        Поиск по изменениям
      </Typography>
      <Typography variant="body1" color="text.secondary" sx={{ mb: 4 }}>
        Найти устройства, у которых конфиг изменился по условию: добавились и/или удалились строки по шаблонам (regex)
      </Typography>

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
            <Typography variant="h6" fontWeight={600} gutterBottom>
              Результаты: устройств с подходящими изменениями — {results.length}
            </Typography>
            <TableContainer>
              <Table>
                <TableHead>
                  <TableRow>
                    <TableCell sx={{ width: 50 }} />
                    <TableCell>Устройство</TableCell>
                    <TableCell>IP</TableCell>
                    <TableCell>Вендор</TableCell>
                    <TableCell>Модель</TableCell>
                    <TableCell align="right">Изменений</TableCell>
                  </TableRow>
                </TableHead>
                <TableBody>
                  {results.map((row) => (
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
                        <TableCell>{row.mgmt_ip || '-'}</TableCell>
                        <TableCell>{row.vendor || '-'}</TableCell>
                        <TableCell>{row.model || '-'}</TableCell>
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
                                  onClick={() => handleShowDiff(ch.left_version_id, ch.right_version_id)}
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
                                    Нажмите для просмотра diff →
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
          </CardContent>
        </Card>
      )}

      <Dialog open={diffDialog.open} onClose={closeDiffDialog} maxWidth="xl" fullWidth>
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
              <ChangesTab embedded initialDiffData={diffDialog.data} />
            ) : null}
          </Box>
        </DialogContent>
      </Dialog>
    </Box>
  );
};

export default SearchChanges;
