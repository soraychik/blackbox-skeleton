import React from 'react';
import {
  Box,
  Button,
  CircularProgress,
  Dialog,
  DialogContent,
  IconButton,
  Typography,
} from '@mui/material';
import {
  Close as CloseIcon,
  Download as DownloadIcon,
} from '@mui/icons-material';

/**
 * Универсальный диалог просмотра конфигурации.
 *
 * Props:
 *   open          — boolean
 *   onClose       — () => void
 *   title         — string, заголовок диалога
 *   content       — string | null, полный текст конфига (если передан — показывается как есть)
 *   snippets      — array | null, массив { line, text, match } (если передан — показываются сниппеты)
 *   onDownload    — () => void | null, колбэк скачивания
 *   downloadLoading — boolean
 */
const ConfigViewDialog = ({
  open,
  onClose,
  title = 'Просмотр конфигурации',
  content = null,
  snippets = null,
  onDownload = null,
  downloadLoading = false,
}) => {
  return (
    <Dialog open={open} onClose={onClose} maxWidth="lg" fullWidth>
      <DialogContent sx={{ p: 0 }}>
        {/* Шапка */}
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
          <Typography variant="h6">{title}</Typography>
          <Box sx={{ display: 'flex', gap: 1, alignItems: 'center' }}>
            {onDownload && (
              <Button
                size="small"
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
            <IconButton onClick={onClose}>
              <CloseIcon />
            </IconButton>
          </Box>
        </Box>

        {/* Тело */}
        <Box
          sx={{
            maxHeight: '70vh',
            overflow: 'auto',
            bgcolor: 'background.default',
          }}
        >
          {/* Режим полного текста */}
          {content !== null && (
            <Box
              component="pre"
              sx={{
                p: 2,
                m: 0,
                fontFamily: 'monospace',
                fontSize: '0.875rem',
                whiteSpace: 'pre-wrap',
                wordBreak: 'break-all',
              }}
            >
              {content}
            </Box>
          )}

          {/* Режим сниппетов */}
          {snippets !== null && (
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
                      bgcolor: snippet.match ? 'action.selected' : 'transparent',
                      borderRadius: 0.5,
                      px: 1,
                      py: 0.25,
                    }}
                  >
                    <Typography
                      component="span"
                      sx={{
                        minWidth: 40,
                        color: 'text.disabled',
                        userSelect: 'none',
                        fontFamily: 'monospace',
                        fontSize: '0.875rem',
                        flexShrink: 0,
                      }}
                    >
                      {snippet.line}
                    </Typography>
                    <Typography
                      component="span"
                      sx={{
                        fontFamily: 'monospace',
                        fontSize: '0.875rem',
                        whiteSpace: 'pre-wrap',
                        wordBreak: 'break-all',
                        fontWeight: snippet.match ? 600 : 400,
                        color: snippet.match ? 'primary.main' : 'text.secondary',
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