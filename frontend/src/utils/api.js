import axios from 'axios';

// Определяем URL API
// В браузере используем localhost или значение из переменной окружения
// так как браузер работает на хосте пользователя, а не внутри Docker сети
const getApiUrl = () => {
  if (process.env.REACT_APP_API_URL) {
    return process.env.REACT_APP_API_URL;
  }
  return 'http://localhost:8080/api';
};

const API_URL = getApiUrl();

// Логируем используемый URL для отладки
if (process.env.NODE_ENV === 'development') {
  console.log('API URL:', API_URL);
}

const api = axios.create({
  baseURL: API_URL,
  headers: {
    'Content-Type': 'application/json',
  },
  timeout: 10000, // 10 секунд таймаут
});

// Добавляем обработчик ошибок
api.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.code === 'ECONNABORTED') {
      error.message = 'Превышено время ожидания ответа от сервера';
    } else if (error.message === 'Network Error') {
      error.message = `Не удалось подключиться к API серверу. Проверьте, что сервер запущен по адресу: ${API_URL}`;
    }
    return Promise.reject(error);
  }
);

export const getDevices = async () => {
  const response = await api.get('/devices');
  return response.data.devices || [];
};

export const getVersions = async () => {
  const response = await api.get('/versions');
  return response.data.versions || [];
};

export const getVersionContent = async (versionId) => {
  const response = await api.get(`/versions/${versionId}/content`, {
    responseType: 'text',
    timeout: 60000, // большие конфиги (800KB+) — даём до 60 с
  });
  return response.data;
};

export const getVersionDiff = async (versionId1, versionId2) => {
  const response = await api.get(`/versions/diff/${versionId1}/${versionId2}`, {
    timeout: 90000, // сравнение двух больших конфигов
  });
  return response.data;
};

export const searchPattern = async (params) => {
  const response = await api.post('/search/count', params);
  return response.data.results || [];
};

export const getDeviceVersions = async (deviceId) => {
  const response = await api.get(`/devices/${deviceId}/versions`);
  return response.data;
};

export const getDevicesDiff = async (deviceId1, deviceId2, date) => {
  const response = await api.post('/diff/devices', {
    device_id_1: deviceId1,
    device_id_2: deviceId2,
    date,
  });
  return response.data;
};

// UC-2: сравнение конфигурации устройства между датами (ТЗ 2.3)
export const getDiffByDate = async (deviceId, date1, date2) => {
  const response = await api.get(
    `/diff/date?deviceId=${deviceId}&date1=${date1}&date2=${date2}`
  );
  return response.data;
};

// UC-4: выгрузка конфига за выбранную дату (ТЗ 2.3)
export const exportConfigByDate = async (deviceId, date) => {
  const response = await api.get(
    `/export/config?deviceId=${deviceId}&date=${date}`,
    { responseType: 'blob' }
  );
  return response;
};

// UC-1: поиск устройств по изменениям (добавились/удалились строки по шаблонам)
export const searchChanges = async (body) => {
  const response = await api.post('/search/changes', body);
  return response.data;
};

export default api;

