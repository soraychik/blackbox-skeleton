import React, { useState } from 'react';
import { useNavigate } from 'react-router-dom';
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
  TextField,
  Typography,
} from '@mui/material';
import {
  Search as SearchIcon,
  Compare as CompareIcon,
  Close as CloseIcon,
} from '@mui/icons-material';
import { searchChanges, getVersionDiff } from '../utils/api';
import ChangesTab from '../components/ChangesTab';

/**
 * UC-1: Поиск устройств, у которых конфиг изменился по множественному условию
 * (добавились/удалились строки по шаблонам). ТЗ 2.1, 2.3.
 */
const SearchChanges = () => {
  const navigate = useNavigate();
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
        Поиск по изменениям (UC-1)
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
                    <TableCell>Устройство</TableCell>
                    <TableCell>IP</TableCell>
                    <TableCell>Изменения</TableCell>
                    <TableCell align="right">Действия</TableCell>
                  </TableRow>
                </TableHead>
                <TableBody>
                  {results.map((row) => (
                    <TableRow key={row.device_id} hover>
                      <TableCell>
                        <Typography
                          fontWeight={500}
                          component="button"
                          onClick={() => navigate(`/devices/${row.device_id}`)}
                          sx={{
                            background: 'none',
                            border: 'none',
                            cursor: 'pointer',
                            color: 'primary.main',
                            textDecoration: 'underline',
                            '&:hover': { color: 'primary.dark' },
                          }}
                        >
                          {row.hostname}
                        </Typography>
                      </TableCell>
                      <TableCell>{row.mgmt_ip || '-'}</TableCell>
                      <TableCell>
                        {row.changes && row.changes.length > 0 ? (
                          <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 0.5 }}>
                            {row.changes.slice(0, 3).map((ch, idx) => (
                              <Chip
                                key={idx}
                                size="small"
                                label={`${ch.left_date}→${ch.right_date}: +${ch.added_count}/-${ch.removed_count}`}
                                variant="outlined"
                              />
                            ))}
                            {row.changes.length > 3 && (
                              <Chip size="small" label={`+${row.changes.length - 3} ещё`} />
                            )}
                          </Box>
                        ) : (
                          '-'
                        )}
                      </TableCell>
                      <TableCell align="right">
                        {row.changes && row.changes[0] && (
                          <Button
                            size="small"
                            startIcon={<CompareIcon />}
                            onClick={() =>
                              handleShowDiff(
                                row.changes[0].left_version_id,
                                row.changes[0].right_version_id
                              )
                            }
                          >
                            Дифф
                          </Button>
                        )}
                      </TableCell>
                    </TableRow>
                  ))}
                  {results.length === 0 && (
                    <TableRow>
                      <TableCell colSpan={4} align="center" sx={{ py: 4 }}>
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
