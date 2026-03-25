import React, { useState, useEffect, useMemo, useRef } from 'react';
import { FixedSizeList } from 'react-window';
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

const LINE_HEIGHT = 21;

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
  const listContainerRef = useRef(null);
  const [listHeight, setListHeight] = useState(() =>
  typeof window !== 'undefined' ? Math.floor(window.innerHeight * 0.7) : 400
);

  // ── Сначала все вычисляемые переменные ──────────────────────────────────────
  const displayContent = content !== null ? content : fullContent;
  const isFullFileMode = displayContent !== null;
  const showViewFullButton = snippets !== null && !isFullFileMode && onViewFullFile != null;

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

  const fullContentLines = useMemo(() => {
    if (!displayContent) return [];
    return displayContent.split('\n');
  }, [displayContent]);

  // ── Потом все useEffect (уже после объявления переменных выше) ───────────────
  useEffect(() => {
    if (!open) {
      setFullContent(null);
      setLoadingFull(false);
    }
  }, [open]);

  useEffect(() => {
  if (!open || !isFullFileMode || !listContainerRef.current) return;
  const el = listContainerRef.current;
  setListHeight(el.clientHeight);
  const observer = new ResizeObserver((entries) => {
    for (const entry of entries) {
      setListHeight(entry.contentRect.height);
    }
  });
  observer.observe(el);
  return () => observer.disconnect();
}, [open, isFullFileMode, fullContentLines.length]);

  // ────────────────────────────────────────────────────────────────────────────
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

        {isFullFileMode && (
          <Box sx={{ ...codeBlockSx }}>
            <Box ref={listContainerRef} sx={{ height: '70vh' }}>
              <FixedSizeList
                height={listHeight}
                width="100%"
                itemCount={fullContentLines.length}
                itemSize={LINE_HEIGHT}
                overscanCount={10}
              >
                {({ index, style }) => (
                  <div style={style}>
                    <Box sx={{ display: 'flex', alignItems: 'flex-start', pl: 2, pr: 2 }}>
                      <Box
                        component="span"
                        sx={{
                          ...codeBlockSx,
                          ...lineNumberSx,
                          flexShrink: 0,
                          lineHeight: LINE_HEIGHT + 'px',
                        }}
                      >
                        {index + 1}
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
                          lineHeight: LINE_HEIGHT + 'px',
                        }}
                      >
                        {fullContentLines[index] === '' ? '\u00A0' : fullContentLines[index]}
                      </Box>
                    </Box>
                  </div>
                )}
              </FixedSizeList>
            </Box>
          </Box>
        )}

        {snippets !== null && !isFullFileMode && (
          <Box sx={{ maxHeight: '70vh', overflow: 'auto', ...codeBlockSx }}>
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
                      bgcolor: snippet.match
                        ? isDark
                          ? '#475569'
                          : '#e5e7eb'
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
          </Box>
        )}
      </DialogContent>
    </Dialog>
  );
};

export default ConfigViewDialog;