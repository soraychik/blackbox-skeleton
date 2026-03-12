import React, { useState, useEffect, useMemo } from 'react';
import {
  Box,
  Button,
  CircularProgress,
  Dialog,
  DialogContent,
  IconButton,
  Typography,
  useTheme,
} from '@mui/material';
import {
  Close as CloseIcon,
  Download as DownloadIcon,
  Visibility as VisibilityIcon,
} from '@mui/icons-material';

/**
 * Универсальный диалог просмотра конфигурации.
 *
 * Props:
 *   open            — boolean
 *   onClose         — () => void
 *   title           — string, заголовок диалога
 *   content         — string | null, полный текст конфига (если передан — показывается как есть)
 *   snippets        — array | null, массив { line, text, match } (если передан — показываются сниппеты)
 *   onDownload      — () => void | null, колбэк скачивания
 *   downloadLoading — boolean
 *   onViewFullFile  — () => Promise<string> | null, загрузка полного файла (кнопка «Посмотреть весь файл»)
 */
const ConfigViewDialog = ({
  open,
  onClose,
  title = 'Просмотр конфигурации',
  content = null,
  snippets = null,
  onDownload = null,
  downloadLoading = false,
  onViewFullFile = null,
}) => {
  const theme = useTheme();
  const isDark = theme.palette.mode === 'dark';

  const [fullContent, setFullContent] = useState(null);
  const [loadingFull, setLoadingFull] = useState(false);

  const displayContent = content !== null ? content : fullContent;
  const isFullFileMode = displayContent !== null;
  const showViewFullButton = snippets !== null && !isFullFileMode && onViewFullFile != null;

  // Стили в зависимости от темы: светлая — белый фон и чёрный текст, тёмная — как в приложении
  const codeBlockSx = useMemo(
    () => ({
      fontFamily: 'monospace',
      fontSize: '0.875rem',
      ...(isDark
        ? { bgcolor: 'background.default', color: 'text.primary' }
        : { bgcolor: '#fff', color: '#000' }),
    }),
    [isDark]
  );

  // Нумерация по правому краю, фиксированная ширина — строки текста начинаются друг под другом
  const lineNumberSx = useMemo(
    () => ({
      width: 56,
      minWidth: 56,
      color: 'text.disabled',
      userSelect: 'none',
      fontFamily: 'monospace',
      fontSize: '0.875rem',
      flexShrink: 0,
      textAlign: 'right',
      pr: 2,
    }),
    []
  );

  // Разбиваем контент по строкам для построчного рендера: у каждой логической строки один номер слева
  const fullContentLines = useMemo(() => {
    if (!displayContent) return [];
    return (displayContent || '').split('\n');
  }, [displayContent]);

  useEffect(() => {
    if (!open) {
      setFullContent(null);
      setLoadingFull(false);
    }
  }, [open]);

  const handleViewFullFile = async () => {
    if (!onViewFullFile) return;
    setLoadingFull(true);
    try {
      const text = await onViewFullFile();
      setFullContent(text);
    } finally {
      setLoadingFull(false);
    }
  };

  return (
    <Dialog
      open={open}
      onClose={onClose}
      maxWidth="lg"
      fullWidth
      PaperProps={{
        sx: {
          bgcolor: isDark ? 'background.paper' : undefined,
          color: isDark ? 'text.primary' : undefined,
        },
      }}
    >
      <DialogContent sx={{ p: 0 }}>
        {/* Шапка — в тёмной теме в тон приложению */}
        <Box
          sx={{
            display: 'flex',
            justifyContent: 'space-between',
            alignItems: 'center',
            p: 2,
            borderBottom: 1,
            borderColor: 'divider',
            ...(isDark ? { bgcolor: 'background.paper', color: 'text.primary' } : {}),
          }}
        >
          <Typography variant="h6">{title}</Typography>
          <Box sx={{ display: 'flex', gap: 1, alignItems: 'center' }}>
            {onDownload && (
              <Button
                size="medium"
                startIcon={
                  downloadLoading ? (
                    <CircularProgress size={16} color="inherit" />
                  ) : (
                    <DownloadIcon />
                  )
                }
                onClick={onDownload}
                disabled={downloadLoading}
              >
                Скачать
              </Button>
            )}
            {showViewFullButton && (
              <Button
                size="medium"
                startIcon={
                  loadingFull ? (
                    <CircularProgress size={16} color="inherit" />
                  ) : (
                    <VisibilityIcon />
                  )
                }
                onClick={handleViewFullFile}
                disabled={loadingFull}
              >
                Посмотреть весь файл
              </Button>
            )}
            <IconButton onClick={onClose}>
              <CloseIcon />
            </IconButton>
          </Box>
        </Box>

        {/* Тело: светлая тема — белый фон и чёрный текст, тёмная — как в приложении */}
        <Box
          sx={{
            maxHeight: '70vh',
            overflow: 'auto',
            ...codeBlockSx,
          }}
        >
          {/* Режим полного текста: каждая логическая строка — один номер слева и контент справа (перенос не дублирует номер) */}
          {isFullFileMode && (
            <Box sx={{ p: 2 }}>
              {fullContentLines.map((lineText, idx) => (
                <Box
                  key={idx}
                  sx={{
                    display: 'flex',
                    alignItems: 'flex-start',
                    gap: 0,
                    lineHeight: 1.5,
                  }}
                >
                  <Box
                    component="span"
                    sx={{
                      ...codeBlockSx,
                      ...lineNumberSx,
                      flexShrink: 0,
                      py: '2px',
                    }}
                  >
                    {idx + 1}
                  </Box>
                  <Box
                    component="pre"
                    sx={{
                      m: 0,
                      flex: 1,
                      minWidth: 0,
                      ...codeBlockSx,
                      whiteSpace: 'pre-wrap',
                      wordBreak: 'break-all',
                      lineHeight: 1.5,
                      py: '2px',
                    }}
                  >
                    {lineText === '' ? '\u00A0' : lineText}
                  </Box>
                </Box>
              ))}
            </Box>
          )}

          {/* Режим сниппетов — тот же стиль + синяя подсветка совпадений */}
          {snippets !== null && !isFullFileMode && (
            <Box sx={{ p: 2 }}>
              {snippets.length === 0 ? (
                <Typography color="text.secondary">Нет сниппетов</Typography>
              ) : (
                snippets.map((snippet, idx) => (
                  <Box
                    key={idx}
                    sx={{
                      display: 'flex',
                      gap: 2,
                      mb: 0.5,
                      // Светлая тема: вся строка серая, текст синий. Тёмная: та же гамма — серая строка + синий текст
                      bgcolor: snippet.match
                        ? isDark
                          ? '#475569'   // серый фон строки (на тёмном фоне)
                          : '#e5e7eb'   // серый фон строки (светлая тема)
                        : 'transparent',
                      borderRadius: 0.5,
                      px: 1,
                      py: 0.25,
                    }}
                  >
                    <Typography
                      component="span"
                      sx={{
                        ...lineNumberSx,
                        ...(snippet.match
                          ? { color: isDark ? 'primary.light' : 'primary.dark', fontWeight: 600 }
                          : {}),
                      }}
                    >
                      {snippet.line}
                    </Typography>
                    <Typography
                      component="span"
                      sx={{
                        ...codeBlockSx,
                        flex: 1,
                        whiteSpace: 'pre-wrap',
                        wordBreak: 'break-all',
                        ...(snippet.match
                          ? { color: isDark ? 'primary.light' : 'primary.dark', fontWeight: 600 }
                          : {}),
                      }}
                    >
                      {snippet.text}
                    </Typography>
                  </Box>
                ))
              )}
            </Box>
          )}
        </Box>
      </DialogContent>
    </Dialog>
  );
};

export default ConfigViewDialog;