/**
 * Преобразует дату из формата YYYY-MM-DD в DD/MM/YYYY
 * @param {string} isoDate - Дата в формате YYYY-MM-DD
 * @returns {string} Дата в формате DD/MM/YYYY
 */
export const toDDMMYYYY = (isoDate) => {
  if (!isoDate || typeof isoDate !== 'string') return isoDate || '';
  const parts = isoDate.trim().split(/[-/]/);
  if (parts.length !== 3) return isoDate;
  const [y, m, d] = parts[0].length === 4 ? [parts[0], parts[1], parts[2]] : [parts[2], parts[1], parts[0]];
  return `${d.padStart(2, '0')}/${m.padStart(2, '0')}/${y}`;
};

/**
 * Форматирует дату в формат дд/мм/гггг чч:мм:сс
 * @param {string} dateString - Дата в формате ISO или MySQL datetime
 * @returns {string} Отформатированная дата
 */
export const formatDateTime = (dateString) => {
  if (!dateString) return '';
  
  const date = new Date(dateString);
  
  if (isNaN(date.getTime())) {
    return dateString; // Возвращаем исходную строку, если не удалось распарсить
  }
  
  const day = String(date.getDate()).padStart(2, '0');
  const month = String(date.getMonth() + 1).padStart(2, '0');
  const year = date.getFullYear();
  const hours = String(date.getHours()).padStart(2, '0');
  const minutes = String(date.getMinutes()).padStart(2, '0');
  const seconds = String(date.getSeconds()).padStart(2, '0');
  
  return `${day}/${month}/${year} ${hours}:${minutes}:${seconds}`;
};

