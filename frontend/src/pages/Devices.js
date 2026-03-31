import React, { useState, useEffect, useMemo } from 'react';
import { useNavigate } from 'react-router-dom';
import {
  Accordion,
  AccordionDetails,
  AccordionSummary,
  Alert,
  Box,
  Button,
  Card,
  CardContent,
  Checkbox,
  Chip,
  CircularProgress,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  FormControlLabel,
  Grid,
  InputAdornment,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TablePagination,
  TableRow,
  TextField,
  Typography,
} from '@mui/material';
import {
  ExpandMore as ExpandMoreIcon,
  FilterList as FilterListIcon,
  Search as SearchIcon,
} from '@mui/icons-material';
import { getDevices } from '../utils/api';

const Devices = () => {
  const navigate = useNavigate();
  const [loading, setLoading] = useState(true);
  const [devices, setDevices] = useState([]);
  const [error, setError] = useState(null);
  const [page, setPage] = useState(0);
  const [rowsPerPage, setRowsPerPage] = useState(25);
  const [search, setSearch] = useState('');
  const [filtersDialogOpen, setFiltersDialogOpen] = useState(false);
  const [appliedFilters, setAppliedFilters] = useState({ types: [], locations: [] });
  const [draftFilters, setDraftFilters] = useState({ types: [], locations: [] });
  const [expanded, setExpanded] = useState({
    type: true,
    vendor: false,
    model: false,
    location: true,
    tags: false,
  });
  const [showAll, setShowAll] = useState({ type: false, location: false });

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

  const typeOptions = useMemo(
    () => [...new Set(devices.map((d) => d.device_type).filter(Boolean))].sort((a, b) => a.localeCompare(b, 'ru')),
    [devices]
  );

  const locationOptions = useMemo(
    () => [...new Set(devices.map((d) => d.location).filter(Boolean))].sort((a, b) => a.localeCompare(b, 'ru')),
    [devices]
  );

  const filteredDevices = useMemo(() => {
    return devices.filter((device) => {
      const matchesSearch =
        !search ||
        device.hostname?.toLowerCase().includes(search.toLowerCase()) ||
        device.mgmt_ip?.toLowerCase().includes(search.toLowerCase());

      const matchesType =
        appliedFilters.types.length === 0 ||
        appliedFilters.types.includes(device.device_type);

      const matchesLocation =
        appliedFilters.locations.length === 0 ||
        appliedFilters.locations.includes(device.location);

      return matchesSearch && matchesType && matchesLocation;
    });
  }, [devices, search, appliedFilters]);

  const paginatedDevices = useMemo(() => {
    const start = page * rowsPerPage;
    return filteredDevices.slice(start, start + rowsPerPage);
  }, [filteredDevices, page, rowsPerPage]);

  const selectedFilterChips = useMemo(
    () => [
      ...appliedFilters.types.map((value) => ({ key: `type:${value}`, label: `Тип: ${value}`, group: 'types', value })),
      ...appliedFilters.locations.map((value) => ({ key: `location:${value}`, label: `Площадка: ${value}`, group: 'locations', value })),
    ],
    [appliedFilters]
  );

  const openFiltersDialog = () => {
    setDraftFilters(appliedFilters);
    setFiltersDialogOpen(true);
  };

  const applyFilters = () => {
    setAppliedFilters(draftFilters);
    setFiltersDialogOpen(false);
    setPage(0);
  };

  const clearDraftFilters = () => {
    setDraftFilters({ types: [], locations: [] });
  };

  const clearAllFilters = () => {
    setSearch('');
    setAppliedFilters({ types: [], locations: [] });
    setDraftFilters({ types: [], locations: [] });
    setPage(0);
  };

  const toggleDraftItem = (group, value) => {
    setDraftFilters((prev) => {
      const hasValue = prev[group].includes(value);
      return {
        ...prev,
        [group]: hasValue ? prev[group].filter((item) => item !== value) : [...prev[group], value],
      };
    });
  };

  const removeAppliedFilter = (group, value) => {
    setAppliedFilters((prev) => ({
      ...prev,
      [group]: prev[group].filter((item) => item !== value),
    }));
    setPage(0);
  };

  const handleRowClick = (deviceId) => {
    navigate(`/devices/${deviceId}`);
  };

  const renderFilterOptions = (group, options, key) => {
    const visibleOptions = showAll[key] ? options : options.slice(0, 5);
    return (
      <Box>
        <Box sx={showAll[key] ? { maxHeight: 180, overflowY: 'auto', pr: 1 } : {}}>
          {visibleOptions.map((option) => (
            <FormControlLabel
              key={option}
              control={<Checkbox checked={draftFilters[group].includes(option)} onChange={() => toggleDraftItem(group, option)} />}
              label={option}
            />
          ))}
        </Box>
        {options.length > 5 && !showAll[key] && (
          <Button size="small" onClick={() => setShowAll((prev) => ({ ...prev, [key]: true }))}>
            Показать все
          </Button>
        )}
      </Box>
    );
  };

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
            <Grid item xs={12} md={6}>
              <TextField
                fullWidth
                size="small"
                placeholder="Поиск по имени или IP..."
                value={search}
                onChange={(e) => {
                  setSearch(e.target.value);
                  setPage(0);
                }}
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
              <Button variant="outlined" startIcon={<FilterListIcon />} onClick={openFiltersDialog}>
                Фильтры
              </Button>
            </Grid>
          </Grid>
          {selectedFilterChips.length > 0 && (
            <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 1, mt: 2 }}>
              {selectedFilterChips.map((chip) => (
                <Chip
                  key={chip.key}
                  label={chip.label}
                  color="warning"
                  onDelete={() => removeAppliedFilter(chip.group, chip.value)}
                />
              ))}
              <Chip label="Сбросить фильтры" variant="outlined" onClick={clearAllFilters} />
            </Box>
          )}
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
                    <TableCell>Тип</TableCell>
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
                      <TableCell>{device.device_type || '-'}</TableCell>
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
                      <TableCell colSpan={7} align="center" sx={{ py: 4 }}>
                        <Typography color="text.secondary">Устройства не найдены</Typography>
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

      <Dialog open={filtersDialogOpen} onClose={() => setFiltersDialogOpen(false)} fullWidth maxWidth="sm">
        <DialogTitle>Фильтры</DialogTitle>
        <DialogContent dividers sx={{ maxHeight: '60vh' }}>
          <Accordion expanded={expanded.type} onChange={(_, x) => setExpanded((p) => ({ ...p, type: x }))}>
            <AccordionSummary expandIcon={<ExpandMoreIcon />}>
              <Typography fontWeight={500}>Тип</Typography>
            </AccordionSummary>
            <AccordionDetails>{renderFilterOptions('types', typeOptions, 'type')}</AccordionDetails>
          </Accordion>

          <Accordion expanded={expanded.vendor} onChange={(_, x) => setExpanded((p) => ({ ...p, vendor: x }))}>
            <AccordionSummary expandIcon={<ExpandMoreIcon />}>
              <Typography fontWeight={500}>Вендор</Typography>
            </AccordionSummary>
            <AccordionDetails>
              <Typography color="text.secondary">Скоро будет доступно</Typography>
            </AccordionDetails>
          </Accordion>

          <Accordion expanded={expanded.model} onChange={(_, x) => setExpanded((p) => ({ ...p, model: x }))}>
            <AccordionSummary expandIcon={<ExpandMoreIcon />}>
              <Typography fontWeight={500}>Модель</Typography>
            </AccordionSummary>
            <AccordionDetails>
              <Typography color="text.secondary">Скоро будет доступно</Typography>
            </AccordionDetails>
          </Accordion>

          <Accordion expanded={expanded.location} onChange={(_, x) => setExpanded((p) => ({ ...p, location: x }))}>
            <AccordionSummary expandIcon={<ExpandMoreIcon />}>
              <Typography fontWeight={500}>Площадка</Typography>
            </AccordionSummary>
            <AccordionDetails>{renderFilterOptions('locations', locationOptions, 'location')}</AccordionDetails>
          </Accordion>

          <Accordion expanded={expanded.tags} onChange={(_, x) => setExpanded((p) => ({ ...p, tags: x }))}>
            <AccordionSummary expandIcon={<ExpandMoreIcon />}>
              <Typography fontWeight={500}>Теги</Typography>
            </AccordionSummary>
            <AccordionDetails>
              <Typography color="text.secondary">Скоро будет доступно</Typography>
            </AccordionDetails>
          </Accordion>
        </DialogContent>
        <DialogActions sx={{ px: 3, py: 2, flexDirection: 'column', alignItems: 'stretch', gap: 1 }}>
          <Button variant="contained" onClick={applyFilters}>
            Применить
          </Button>
          <Button variant="text" onClick={clearDraftFilters}>
            Сбросить фильтры
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
};

export default Devices;
