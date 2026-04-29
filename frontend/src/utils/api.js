import axios from 'axios';

const getApiUrl = () => {
  if (import.meta.env.VITE_API_URL) {
    return import.meta.env.VITE_API_URL;
  }
  return '/api';
};

const API_URL = getApiUrl();

const api = axios.create({
  baseURL: API_URL,
  headers: {
    'Content-Type': 'application/json',
  },
  timeout: 10000,
});

let navigateFn = null;

export const setNavigate = (fn) => {
  navigateFn = fn;
};

api.interceptors.request.use(
  (config) => {
    const token = localStorage.getItem('token');
    if (token) {
      config.headers.Authorization = `Bearer ${token}`;
    }
    return config;
  },
  (error) => Promise.reject(error)
);

api.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response?.status === 401) {
      localStorage.removeItem('token');
      localStorage.removeItem('user');
      if (navigateFn) {
        navigateFn('/login');
      } else {
        window.location.href = '/login';
      }
    }
    if (error.code === 'ECONNABORTED') {
      error.message = 'Превышено время ожидания ответа от сервера';
    } else if (error.message === 'Network Error') {
      error.message = `Не удалось подключиться к API серверу. Проверьте, что сервер запущен по адресу: ${API_URL}`;
    }
    return Promise.reject(error);
  }
);

export const login = async (loginValue, password, remember = false) => {
  const response = await api.post('/auth/login', { login: loginValue, password, remember });
  return response.data;
};

export const getDevices = async () => {
  const response = await api.get('/devices');
  return response.data.devices || [];
};

export const getVersions = async () => {
  const response = await api.get('/versions');
  return response.data.versions || [];
};

export const getDashboardStats = async () => {
  const response = await api.get('/dashboard/stats');
  return response.data;
};

export const getVersionContent = async (versionId) => {
  const response = await api.get(`/versions/${versionId}/content`, {
    responseType: 'text',
    timeout: 60000,
  });
  return response.data;
};

export const getVersionDiff = async (versionId1, versionId2) => {
  const response = await api.get(`/versions/diff/${versionId1}/${versionId2}`, {
    timeout: 90000,
  });
  return response.data;
};

export const searchPattern = async (params) => {
  const response = await api.post('/search/count', params, {
    timeout: 120000, // 2 минуты
  });
  return response.data.results || [];
};

export const getDeviceVersions = async (deviceId) => {
  const response = await api.get(`/devices/${deviceId}/versions`);
  return response.data;
};

export const getLatestVersionsForDevices = async (leftDeviceId, rightDeviceId) => {
  const response = await api.get(
    `/devices/compare/latest?leftDeviceId=${leftDeviceId}&rightDeviceId=${rightDeviceId}`
  );
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

export const getDiffByDate = async (deviceId, date1, date2) => {
  const response = await api.get(
    `/diff/date?deviceId=${deviceId}&date1=${date1}&date2=${date2}`
  );
  return response.data;
};

export const exportConfigByDate = async (deviceId, date) => {
  const response = await api.get(
    `/export/config?deviceId=${deviceId}&date=${date}`,
    { responseType: 'blob' }
  );
  return response;
};

export const searchChanges = async (body) => {
  const response = await api.post('/search/changes', body, {
    timeout: 120000, // УДАЛИТЬ ПОТОМ. Костыль
  });
  return response.data;
};

export const triggerScan = async () => {
  const response = await api.post('/scan');
  return response.data;
};

export const getScanStatus = async () => {
  const response = await api.get('/scan/status');
  return response.data;
};

export const getSettings = async () => {
  const response = await api.get('/settings');
  return response.data;
};

export const updateSettings = async (settings) => {
  const response = await api.put('/settings', settings);
  return response.data;
};

export default api;
