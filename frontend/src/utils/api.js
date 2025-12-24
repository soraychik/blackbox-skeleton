import axios from 'axios';

// Определяем URL API
// В браузере всегда используем localhost или значение из переменной окружения
// так как браузер работает на хосте пользователя, а не внутри Docker сети
const getApiUrl = () => {
  // Если переменная окружения установлена, используем её
  if (process.env.REACT_APP_API_URL) {
    return process.env.REACT_APP_API_URL;
  }
  
  // По умолчанию используем localhost:8080
  // В Docker это будет работать, если порт 8080 проброшен на хост
  return 'http://localhost:8080';
};

const API_URL = getApiUrl();

// Логируем используемый URL для отладки (только в development)
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

export default api;

