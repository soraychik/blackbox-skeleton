import React, { useState, useEffect } from 'react';
import {
  Box,
  Button,
  Card,
  CardContent,
  Checkbox,
  CircularProgress,
  Dialog,
  DialogContent,
  FormControlLabel,
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
  Autocomplete,
  Chip,
} from '@mui/material';
import {
  Search as SearchIcon,
  Visibility as VisibilityIcon,
  Close as CloseIcon,
  FileDownload as FileDownloadIcon,
} from '@mui/icons-material';
import { getDevices, searchPattern } from '../utils/api';

const Search = () => {
  const [loading, setLoading] = useState(false);
  const [devices, setDevices] = useState([]);
  const [searchParams, setSearchParams] = useState({
    pattern: '',
    caseSensitive: false,
    scope: 'all',
    selectedDevice: null,
  });
  const [results, setResults] = useState(null);
  const [snippetsDialog, setSnippetsDialog] = useState({ open: false, device: null, snippets: [] });

  useEffect(() => {
    loadDevices();
  }, []);

  const loadDevices = async () => {
    try {
      const data = await getDevices();
      setDevices(data);
    } catch (error) {
      console.error('Failed to load devices:', error);
    }
  };

  const handleSearch = async () => {
    if (!searchParams.pattern.trim()) return;

    try {
      setLoading(true);
      const requestBody = {
        pattern: searchParams.pattern,
        caseSensitive: searchParams.caseSensitive,
        scope: searchParams.scope,
        deviceId: searchParams.scope === 'device' ? searchParams.selectedDevice?.id : null,
      };

      const data = await searchPattern(requestBody);
      setResults(data);
    } catch (error) {
      console.error('Failed to search:', error);
    } finally {
      setLoading(false);
    }
  };

  const handleExport = () => {
    if (!results) return;

    const csvContent = [
      ['Hostname', 'IP', 'Matches'],
      ...results.map((r) => [r.hostname, r.mgmt_ip || '', r.match_count]),
    ]
      .map((row) => row.join(','))
      .join('\n');

    const blob = new Blob([csvContent], { type: 'text/csv' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `search_results_${Date.now()}.csv`;
    a.click();
    URL.revokeObjectURL(url);
  };

  const handleShowSnippets = (device, snippets) => {
    setSnippetsDialog({ open: true, device, snippets });
  };

  return (
    <Box>
      <Typography variant="h4" fontWeight={600} gutterBottom>
        Поиск и подсчет
      </Typography>
      <Typography variant="body1" color="text.secondary" sx={{ mb: 4 }}>
        Найдите устройства с определенным паттерном в конфигурации
      </Typography>

      <Card sx={{ mb: 4 }}>
        <CardContent>
          <Grid container spacing={3}>
            <Grid item xs={12}>
              <TextField
                fullWidth
                label="Паттерн поиска"
                placeholder="Введите regexp или текст..."
                value={searchParams.pattern}
                onChange={(e) =>
                  setSearchParams((prev) => ({ ...prev, pattern: e.target.value }))
                }
                onKeyDown={(e) => e.key === 'Enter' && handleSearch()}
              />
            </Grid>
            <Grid item xs={12}>
              <FormControlLabel
                control={
                  <Checkbox
                    checked={searchParams.caseSensitive}
                    onChange={(e) =>
                      setSearchParams((prev) => ({
                        ...prev,
                        caseSensitive: e.target.checked,
                      }))
                    }
                  />
                }
                label="Учитывать регистр"
              />
            </Grid>
            <Grid item xs={12} md={4}>
              <TextField
                select
                fullWidth
                label="Область поиска"
                value={searchParams.scope}
                onChange={(e) =>
                  setSearchParams((prev) => ({ ...prev, scope: e.target.value }))
                }
              >
                <option value="all">Все устройства</option>
                <option value="device">Конкретное устройство</option>
              </TextField>
            </Grid>
            {searchParams.scope === 'device' && (
              <Grid item xs={12} md={8}>
                <Autocomplete
                  options={devices}
                  getOptionLabel={(option) => `${option.hostname} (${option.mgmt_ip || 'N/A'})`}
                  value={searchParams.selectedDevice}
                  onChange={(e, value) =>
                    setSearchParams((prev) => ({ ...prev, selectedDevice: value }))
                  }
                  renderInput={(params) => (
                    <TextField {...params} label="Выберите устройство" />
                  )}
                />
              </Grid>
            )}
            <Grid item xs={12}>
              <Button
                variant="contained"
                startIcon={loading ? <CircularProgress size={20} color="inherit" /> : <SearchIcon />}
                onClick={handleSearch}
                disabled={loading || !searchParams.pattern.trim()}
              >
                {loading ? 'Поиск...' : 'Найти и подсчитать'}
              </Button>
            </Grid>
          </Grid>
        </CardContent>
      </Card>

      {results && (
        <Card>
          <CardContent>
            <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 2 }}>
              <Typography variant="h6" fontWeight={600}>
                Результаты поиска
              </Typography>
              <Box sx={{ display: 'flex', gap: 1, alignItems: 'center' }}>
                <Typography variant="body2" color="text.secondary">
                  Всего совпадений: {results.reduce((sum, r) => sum + r.match_count, 0)}
                </Typography>
                <Button
                  size="small"
                  startIcon={<FileDownloadIcon />}
                  onClick={handleExport}
                >
                  CSV
                </Button>
              </Box>
            </Box>

            <TableContainer>
              <Table>
                <TableHead>
                  <TableRow>
                    <TableCell>Устройство</TableCell>
                    <TableCell>IP адрес</TableCell>
                    <TableCell align="center">Совпадений</TableCell>
                    <TableCell align="right">Действия</TableCell>
                  </TableRow>
                </TableHead>
                <TableBody>
                  {results.map((result) => (
                    <TableRow key={result.device_id} hover>
                      <TableCell>
                        <Typography fontWeight={500}>{result.hostname}</Typography>
                      </TableCell>
                      <TableCell>{result.mgmt_ip || '-'}</TableCell>
                      <TableCell align="center">
                        <Chip
                          label={result.match_count}
                          color="primary"
                          size="small"
                        />
                      </TableCell>
                      <TableCell align="right">
                        <IconButton
                          size="small"
                          onClick={() => handleShowSnippets(result.hostname, result.snippets || [])}
                        >
                          <VisibilityIcon fontSize="small" />
                        </IconButton>
                      </TableCell>
                    </TableRow>
                  ))}
                  {results.length === 0 && (
                    <TableRow>
                      <TableCell colSpan={4} align="center" sx={{ py: 4 }}>
                        <Typography color="text.secondary">Ничего не найдено</Typography>
                      </TableCell>
                    </TableRow>
                  )}
                </TableBody>
              </Table>
            </TableContainer>
          </CardContent>
        </Card>
      )}

      {/* Snippets Dialog */}
      <Dialog
        open={snippetsDialog.open}
        onClose={() => setSnippetsDialog({ open: false, device: null, snippets: [] })}
        maxWidth="md"
        fullWidth
      >
        <DialogContent sx={{ p: 0 }}>
          <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', p: 2, borderBottom: 1, borderColor: 'divider' }}>
            <Typography variant="h6">{snippetsDialog.device}</Typography>
            <IconButton onClick={() => setSnippetsDialog({ open: false, device: null, snippets: [] })}>
              <CloseIcon />
            </IconButton>
          </Box>
          <Box
            component="pre"
            sx={{
              p: 2,
              m: 0,
              maxHeight: '60vh',
              overflow: 'auto',
              bgcolor: 'background.default',
              fontFamily: 'monospace',
              fontSize: '0.875rem',
              whiteSpace: 'pre-wrap',
            }}
          >
            {snippetsDialog.snippets.join('\n')}
          </Box>
        </DialogContent>
      </Dialog>
    </Box>
  );
};

export default Search;
