import React, { useState } from 'react';
import {
  Alert,
  Box,
  Button,
  Card,
  CardContent,
  Checkbox,
  CircularProgress,
  FormControlLabel,
  Grid,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  TextField,
  Typography,
  Chip,
} from '@mui/material';
import {
  Search as SearchIcon,
  FileDownload as FileDownloadIcon,
} from '@mui/icons-material';
import { searchPattern, getVersionContent } from '../utils/api';
import ConfigViewDialog from '../components/ConfigViewDialog';

const Search = () => {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(null);
  const [searchParams, setSearchParams] = useState({
    pattern: '',
    caseSensitive: false,
  });
  const [results, setResults] = useState(null);
  const [snippetsDialog, setSnippetsDialog] = useState({
    open: false,
    device: null,
    snippets: [],
    versionId: null,
    downloadLoading: false,
  });

  const handleSearch = async () => {
    if (!searchParams.pattern.trim()) return;

    try {
      setLoading(true);
      setError(null);
      const requestBody = {
        pattern: searchParams.pattern,
        caseSensitive: searchParams.caseSensitive,
        scope: 'all',
      };

      const data = await searchPattern(requestBody);
      setResults(data);
    } catch (error) {
      setError('Не удалось выполнить поиск. Проверьте соединение с сервером');
    } finally {
      setLoading(false);
    }
  };

  const handleExportCsv = () => {
    if (!results) return;

    const csvContent = [
      ['Устройство', 'IP адрес', 'Совпадений'],
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

  const handleRowClick = (result) => {
    setSnippetsDialog({
      open: true,
      device: result.hostname,
      snippets: result.snippets || [],
      versionId: result.version_id || null,
      downloadLoading: false,
    });
  };

  const handleDownloadConfig = async () => {
    if (!snippetsDialog.versionId) return;

    setSnippetsDialog((prev) => ({ ...prev, downloadLoading: true }));
    try {
      const content = await getVersionContent(snippetsDialog.versionId);
      const blob = new Blob([content], { type: 'text/plain' });
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `config_${snippetsDialog.device}.txt`;
      a.click();
      URL.revokeObjectURL(url);
    } catch (error) {
      setError('Не удалось скачать конфигурацию');
    } finally {
      setSnippetsDialog((prev) => ({ ...prev, downloadLoading: false }));
    }
  };

  const closeDialog = () => {
    setSnippetsDialog({ open: false, device: null, snippets: [], versionId: null, downloadLoading: false });
  };

  return (
    <Box>
      <Typography variant="h4" fontWeight={600} gutterBottom>
        Поиск и подсчёт
      </Typography>
      <Typography variant="body1" color="text.secondary" sx={{ mb: 4 }}>
        Найдите устройства с определённым паттерном в конфигурации
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
                      setSearchParams((prev) => ({ ...prev, caseSensitive: e.target.checked }))
                    }
                  />
                }
                label="Учитывать регистр"
              />
            </Grid>
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
                <Button size="small" startIcon={<FileDownloadIcon />} onClick={handleExportCsv}>
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
                  </TableRow>
                </TableHead>
                <TableBody>
                  {results.map((result) => (
                    <TableRow
                      key={result.device_id}
                      hover
                      onClick={() => handleRowClick(result)}
                      sx={{ cursor: 'pointer' }}
                    >
                      <TableCell>
                        <Typography fontWeight={500}>{result.hostname}</Typography>
                      </TableCell>
                      <TableCell>{result.mgmt_ip || '-'}</TableCell>
                      <TableCell align="center">
                        <Chip label={result.match_count} color="primary" size="small" />
                      </TableCell>
                    </TableRow>
                  ))}
                  {results.length === 0 && (
                    <TableRow>
                      <TableCell colSpan={3} align="center" sx={{ py: 4 }}>
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

      <ConfigViewDialog
        open={snippetsDialog.open}
        onClose={closeDialog}
        title={snippetsDialog.device || 'Просмотр конфигурации'}
        snippets={snippetsDialog.snippets}
        onDownload={snippetsDialog.versionId ? handleDownloadConfig : null}
        downloadLoading={snippetsDialog.downloadLoading}
      />
    </Box>
  );
};

export default Search;