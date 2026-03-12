import React, { useState, useEffect, useCallback, useMemo, useRef } from 'react';
import { FixedSizeList } from 'react-window';
import { useTheme } from '@mui/material/styles';
import {
  Box,
  Button,
  Typography,
  Chip,
  FormControl,
  InputLabel,
  Select,
  MenuItem,
  CircularProgress,
} from '@mui/material';
import { getVersions, getVersionDiff } from '../utils/api';
import { formatDateTime } from '../utils/dateFormatter';

const ROW_HEIGHT = 20;
const DIFF_LIST_HEIGHT_VH = 55;

const DiffRow = ({ line, theme, lineNumWidth }) => {
  const isDark = theme.palette.mode === 'dark';

  const getBackgroundColor = () => {
    if (line.type === 'added') {
      return isDark ? 'rgba(46, 160, 67, 0.15)' : '#e6ffed';
    }
    if (line.type === 'removed') {
      return isDark ? 'rgba(248, 81, 73, 0.15)' : '#ffeef0';
    }
    return 'background.paper';
  };

  const getTextColor = () => {
    if (line.type === 'added') return 'success.main';
    if (line.type === 'removed') return 'error.main';
    return 'text.primary';
  };

  return (
    <Box
      sx={{
        display: 'grid',
        gridTemplateColumns: '1fr 1fr',
        bgcolor: getBackgroundColor(),
      }}
    >
      <Box
        sx={{
          display: 'flex',
          borderRight: 1,
          borderColor: 'divider',
          minHeight: 20,
          minWidth: 0,
        }}
      >
        <Box
          sx={{
            width: lineNumWidth,
            minWidth: lineNumWidth,
            px: 1,
            textAlign: 'right',
            color: 'text.secondary',
            bgcolor: line.type === 'removed' ? getBackgroundColor() : 'rgba(0,0,0,0.02)',
            userSelect: 'none',
            borderRight: 1,
            borderColor: 'divider',
            flexShrink: 0,
            fontFamily: 'monospace',
            fontSize: '0.75rem',
            lineHeight: '20px',
          }}
        >
          {(line.type === 'removed' || line.type === 'unchanged') ? line.leftLineNum : ''}
        </Box>
        <Box
          sx={{
            flex: 1,
            minWidth: 0,
            px: 1,
            overflow: 'hidden',
            textOverflow: 'ellipsis',
            whiteSpace: 'nowrap',
            color: getTextColor(),
            fontFamily: 'monospace',
            fontSize: '0.75rem',
            lineHeight: '20px',
          }}
        >
          {(line.type === 'removed' || line.type === 'unchanged')
            ? (line.type === 'removed' ? '− ' : '  ') + (line.content || '')
            : ''}
        </Box>
      </Box>
      <Box
        sx={{
          display: 'flex',
          minHeight: 20,
          minWidth: 0,
        }}
      >
        <Box
          sx={{
            width: lineNumWidth,
            minWidth: lineNumWidth,
            px: 1,
            textAlign: 'right',
            color: 'text.secondary',
            bgcolor: line.type === 'added' ? getBackgroundColor() : 'rgba(0,0,0,0.02)',
            userSelect: 'none',
            borderRight: 1,
            borderColor: 'divider',
            flexShrink: 0,
            fontFamily: 'monospace',
            fontSize: '0.75rem',
            lineHeight: '20px',
          }}
        >
          {(line.type === 'added' || line.type === 'unchanged') ? line.rightLineNum : ''}
        </Box>
        <Box
          sx={{
            flex: 1,
            minWidth: 0,
            px: 1,
            overflow: 'hidden',
            textOverflow: 'ellipsis',
            whiteSpace: 'nowrap',
            color: getTextColor(),
            fontFamily: 'monospace',
            fontSize: '0.75rem',
            lineHeight: '20px',
          }}
        >
          {(line.type === 'added' || line.type === 'unchanged')
            ? (line.type === 'added' ? '+ ' : '  ') + (line.content || '')
            : ''}
        </Box>
      </Box>
    </Box>
  );
};

const DiffStats = ({ stats, totalLines, theme }) => (
  <Box
    sx={{
      display: 'flex',
      alignItems: 'center',
      gap: 1.5,
      px: 2,
      py: 1,
      bgcolor: 'action.hover',
      borderBottom: 1,
      borderColor: 'divider',
      fontFamily: 'monospace',
      fontSize: '0.8125rem',
    }}
  >
    <Typography sx={{ color: 'success.main', fontWeight: 600 }}>
      +{stats.added}
    </Typography>
    <Typography sx={{ color: 'error.main', fontWeight: 600 }}>
      -{stats.removed}
    </Typography>
    <Typography sx={{ color: 'text.secondary', fontSize: '0.75rem', ml: 'auto' }}>
      Всего строк: {totalLines}
    </Typography>
  </Box>
);

const ChangesTab = ({ embedded = false, initialDiffData = null, deviceName = null, version1Date = null, version2Date = null }) => {
  const theme = useTheme();
  const listContainerRef = useRef(null);
  const [listSize, setListSize] = useState({ width: 0, height: 400 });
  const [versions, setVersions] = useState([]);
  const [loading, setLoading] = useState(!embedded);
  const [error, setError] = useState(null);
  const [selectedVersion1, setSelectedVersion1] = useState(null);
  const [selectedVersion2, setSelectedVersion2] = useState(null);
  const [diffData, setDiffData] = useState(initialDiffData);
  const [diffLoading, setDiffLoading] = useState(false);
  const [diffError, setDiffError] = useState(null);

  useEffect(() => {
    const el = listContainerRef.current;
    if (!el) return;
    const updateSize = () => {
      setListSize({ width: el.clientWidth, height: el.clientHeight });
    };
    updateSize();
    const ro = new ResizeObserver(updateSize);
    ro.observe(el);
    return () => ro.disconnect();
  }, [diffData]);

  const loadVersions = useCallback(async () => {
    try {
      setLoading(true);
      setError(null);
      const data = await getVersions();
      setVersions(data);
    } catch (err) {
      const errorMessage = err.response
        ? `Ошибка ${err.response.status}: ${err.response.data?.error || err.message}`
        : err.message || 'Не удалось подключиться к серверу. Проверьте, что API сервер запущен.';
      setError('Ошибка при загрузке данных: ' + errorMessage);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    if (embedded) {
      setLoading(false);
      if (initialDiffData) {
        setDiffData(initialDiffData);
      }
    } else {
      loadVersions();
    }
  }, [loadVersions, embedded, initialDiffData]);

  const handleCompare = async () => {
    if (!selectedVersion1 || !selectedVersion2) {
      setDiffError('Выберите обе версии для сравнения');
      return;
    }

    if (selectedVersion1 === selectedVersion2) {
      setDiffError('Выберите разные версии для сравнения');
      return;
    }

    try {
      setDiffLoading(true);
      setDiffError(null);
      const diff = await getVersionDiff(selectedVersion1, selectedVersion2);
      setDiffData(diff);
    } catch (err) {
      const errorMessage = err.response
        ? `Ошибка ${err.response.status}: ${err.response.data?.error || err.message}`
        : err.message || 'Не удалось получить данные сравнения';
      setDiffError('Ошибка при получении данных сравнения: ' + errorMessage);
      setDiffData(null);
    } finally {
      setDiffLoading(false);
    }
  };

  const getVersionInfo = (versionId) => {
    return versions.find(v => v.id === parseInt(versionId));
  };

  const processedDiff = useMemo(() => {
    if (!diffData || !diffData.lines || diffData.lines.length === 0) {
      return null;
    }

    const lines = diffData.lines;

    const stats = { added: 0, removed: 0 };
    lines.forEach(line => {
      if (line.type === 'added') stats.added++;
      if (line.type === 'removed') stats.removed++;
    });

    if (stats.added === 0 && stats.removed === 0) {
      return { identical: true };
    }

    const processedLines = [];
    let leftLineNum = 1;
    let rightLineNum = 1;

    lines.forEach((line) => {
      processedLines.push({
        ...line,
        leftLineNum: line.left_num !== undefined ? line.left_num : (line.type === 'removed' || line.type === 'unchanged' ? leftLineNum++ : null),
        rightLineNum: line.right_num !== undefined ? line.right_num : (line.type === 'added' || line.type === 'unchanged' ? rightLineNum++ : null),
      });
    });

    const maxLineNum = Math.max(
      ...processedLines.map(l => l.leftLineNum || 0),
      ...processedLines.map(l => l.rightLineNum || 0),
    );
    const lineNumWidth = Math.max(40, String(maxLineNum).length * 9 + 16);

    return {
      identical: false,
      stats,
      lines: processedLines,
      totalLines: lines.length,
      lineNumWidth,
    };
  }, [diffData]);

  if (loading) {
    return (
      <Box sx={{ p: 2, textAlign: 'center' }}>
        <Typography>Загрузка версий...</Typography>
      </Box>
    );
  }

  if (error) {
    return (
      <Box sx={{ p: 2 }}>
        <Typography color="error">{error}</Typography>
        <Button onClick={loadVersions} sx={{ mt: 1 }} variant="outlined">
          Повторить
        </Button>
      </Box>
    );
  }

  const version1Info = selectedVersion1 ? getVersionInfo(selectedVersion1) : null;
  const version2Info = selectedVersion2 ? getVersionInfo(selectedVersion2) : null;

  return (
    <Box
      sx={{
        bgcolor: 'background.paper',
        borderRadius: 2,
        boxShadow: 1,
        p: 3,
      }}
    >
      {!embedded && (
        <>
          <Box sx={{ mb: 3, textAlign: 'center' }}>
            <Typography variant="h5" sx={{ color: 'text.primary', mb: 1 }}>
              Сравнение версий конфигураций
            </Typography>
          </Box>

          <Box
            sx={{
              display: 'grid',
              gridTemplateColumns: { xs: '1fr', md: '1fr 1fr' },
              gap: 2.5,
              mb: 2.5,
            }}
          >
            <FormControl fullWidth>
              <InputLabel id="version1-label">Версия 1 (левая)</InputLabel>
              <Select
                labelId="version1-label"
                value={selectedVersion1 || ''}
                label="Версия 1 (левая)"
                onChange={(e) => setSelectedVersion1(e.target.value)}
              >
                <MenuItem value="">Выберите версию...</MenuItem>
                {versions.map((version) => (
                  <MenuItem key={version.id} value={version.id}>
                    {version.device_hostname} - {formatDateTime(version.created_at)}
                  </MenuItem>
                ))}
              </Select>
              {version1Info && (
                <Typography variant="body2" sx={{ color: 'text.secondary', mt: 1, fontSize: '0.75rem' }}>
                  {version1Info.device_hostname} | Создано: {formatDateTime(version1Info.created_at)}
                </Typography>
              )}
            </FormControl>

            <FormControl fullWidth>
              <InputLabel id="version2-label">Версия 2 (правая)</InputLabel>
              <Select
                labelId="version2-label"
                value={selectedVersion2 || ''}
                label="Версия 2 (правая)"
                onChange={(e) => setSelectedVersion2(e.target.value)}
              >
                <MenuItem value="">Выберите версию...</MenuItem>
                {versions.map((version) => (
                  <MenuItem key={version.id} value={version.id}>
                    {version.device_hostname} - {formatDateTime(version.created_at)}
                  </MenuItem>
                ))}
              </Select>
              {version2Info && (
                <Typography variant="body2" sx={{ color: 'text.secondary', mt: 1, fontSize: '0.75rem' }}>
                  {version2Info.device_hostname} | Создано: {formatDateTime(version2Info.created_at)}
                </Typography>
              )}
            </FormControl>
          </Box>

          <Box sx={{ display: 'flex', justifyContent: 'center', my: 2.5 }}>
            <Button
              variant="contained"
              onClick={handleCompare}
              disabled={!selectedVersion1 || !selectedVersion2 || diffLoading}
              sx={{
                px: 3,
                py: 1.5,
                bgcolor: 'text.primary',
                '&:hover': { bgcolor: 'text.secondary' },
                '&:disabled': { bgcolor: 'action.disabledBackground' },
              }}
            >
              {diffLoading ? 'Сравнение...' : 'Сравнить версии'}
            </Button>
          </Box>
        </>
      )}

      {diffError && (
        <Box sx={{ mt: 2.5 }}>
          <Typography color="error">{diffError}</Typography>
        </Box>
      )}

      {diffData && processedDiff && (
        <Box
          sx={{
            mt: 3.75,
            border: 1,
            borderColor: 'divider',
            borderRadius: 1.5,
            overflow: 'hidden',
          }}
        >
          {processedDiff.identical ? (
            <Box
              sx={{
                p: 5,
                textAlign: 'center',
                color: 'text.secondary',
                bgcolor: 'action.hover',
              }}
            >
              Версии идентичны — изменений нет
            </Box>
          ) : (
            <>
              <DiffStats stats={processedDiff.stats} totalLines={processedDiff.totalLines} theme={theme} />

              <Box
                sx={{
                  display: 'grid',
                  gridTemplateColumns: { xs: '1fr', md: '1fr 1fr' },
                  bgcolor: 'action.hover',
                  borderBottom: 1,
                  borderColor: 'divider',
                }}
              >
                <Box sx={{ p: 1, fontSize: '0.875rem', fontWeight: 600, color: 'text.secondary' }}>
                  {version1Info ? `${version1Info.device_hostname} (${formatDateTime(version1Info.created_at)})` : (deviceName && version1Date ? `${deviceName} (${version1Date})` : `Версия ${diffData.left_version_id}`)}
                </Box>
                <Box sx={{ p: 1, fontSize: '0.875rem', fontWeight: 600, color: 'text.secondary', borderLeft: { md: 1 }, borderColor: 'divider' }}>
                  {version2Info ? `${version2Info.device_hostname} (${formatDateTime(version2Info.created_at)})` : (deviceName && version2Date ? `${deviceName} (${version2Date})` : `Версия ${diffData.right_version_id}`)}
                </Box>
              </Box>

              <Box
                ref={listContainerRef}
                sx={{
                  fontFamily: 'monospace',
                  fontSize: '0.75rem',
                  lineHeight: '20px',
                  height: `max(${ROW_HEIGHT}px, min(${DIFF_LIST_HEIGHT_VH}vh, ${processedDiff.lines.length * ROW_HEIGHT}px))`,
                }}
              >
                {listSize.width > 0 && listSize.height > 0 && (
                  <FixedSizeList
                    height={listSize.height}
                    width={listSize.width}
                    itemCount={processedDiff.lines.length}
                    itemSize={ROW_HEIGHT}
                    overscanCount={10}
                  >
                    {({ index, style }) => (
                      <div style={style}>
                        <DiffRow
                          line={processedDiff.lines[index]}
                          theme={theme}
                          lineNumWidth={processedDiff.lineNumWidth}
                        />
                      </div>
                    )}
                  </FixedSizeList>
                )}
              </Box>
            </>
          )}
        </Box>
      )}
    </Box>
  );
};

export default ChangesTab;