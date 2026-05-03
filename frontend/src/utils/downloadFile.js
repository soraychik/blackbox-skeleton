export const downloadBlob = (content, filename, mimeType = 'text/plain') => {
  const blob = new Blob([content], { type: mimeType });
  const url = URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  a.download = filename;
  a.click();
  URL.revokeObjectURL(url);
};

export const downloadText = (content, filename) =>
  downloadBlob(content, filename, 'text/plain');

export const downloadCsv = (rows, filename) => {
  const csv = rows
    .map(row => row.map(v => `"${String(v ?? '').replace(/"/g, '""')}"`).join(','))
    .join('\n');
  // BOM для корректного открытия в Excel
  downloadBlob('﻿' + csv, filename, 'text/csv;charset=utf-8');
};

export const downloadJson = (data, filename) =>
  downloadBlob(JSON.stringify(data, null, 2), filename, 'application/json');
