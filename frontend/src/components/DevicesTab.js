import React, { useState, useEffect, useCallback } from 'react';
import { getDevices } from '../utils/api';
import { formatDateTime } from '../utils/dateFormatter';
import './DevicesTab.css';

const DevicesTab = () => {
  const [devices, setDevices] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [sortOrder, setSortOrder] = useState('desc'); // 'asc' или 'desc'

  const loadDevices = useCallback(async () => {
    try {
      setLoading(true);
      setError(null);
      const data = await getDevices();
      setDevices(data);
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
    loadDevices();
  }, [loadDevices]);

  const handleSort = () => {
    const newOrder = sortOrder === 'desc' ? 'asc' : 'desc';
    setSortOrder(newOrder);
    
    const sorted = [...devices].sort((a, b) => {
      const dateA = new Date(a.created_at);
      const dateB = new Date(b.created_at);
      
      if (newOrder === 'asc') {
        return dateA - dateB;
      } else {
        return dateB - dateA;
      }
    });
    
    setDevices(sorted);
  };

  if (loading) {
    return <div className="loading">Загрузка...</div>;
  }

  if (error) {
    return (
      <div className="error">
        <p>{error}</p>
        <button onClick={loadDevices} style={{ marginTop: '10px', padding: '8px 16px', cursor: 'pointer' }}>
          Повторить
        </button>
      </div>
    );
  }

  return (
    <div className="table-container">
      <table>
        <thead>
          <tr>
            <th>Название</th>
            <th>
              Время создания
              <button className="sort-button" onClick={handleSort} title="Сортировать">
                {sortOrder === 'desc' ? '↓' : '↑'}
              </button>
            </th>
          </tr>
        </thead>
        <tbody>
          {devices.length === 0 ? (
            <tr>
              <td colSpan="2" className="empty">Нет данных</td>
            </tr>
          ) : (
            devices.map((device) => (
              <tr key={device.id}>
                <td>{device.name}</td>
                <td>{formatDateTime(device.created_at)}</td>
              </tr>
            ))
          )}
        </tbody>
      </table>
    </div>
  );
};

export default DevicesTab;

