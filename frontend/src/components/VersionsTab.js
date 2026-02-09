import React, { useState, useEffect, useCallback } from 'react';
import { getVersions } from '../utils/api';
import { formatDateTime } from '../utils/dateFormatter';
import './VersionsTab.css';

const VersionsTab = () => {
  const [versions, setVersions] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [sortOrder, setSortOrder] = useState('desc'); // 'asc' или 'desc'

  const sortVersionsByChangeDate = useCallback((data, order) => {
    const sorted = [...data].sort((a, b) => {
      const dateA = new Date(a.version_date);
      const dateB = new Date(b.version_date);
      
      if (order === 'asc') {
        return dateA - dateB;
      } else {
        return dateB - dateA;
      }
    });
    
    setVersions(sorted);
  }, []);

  const loadVersions = useCallback(async () => {
    try {
      setLoading(true);
      setError(null);
      const data = await getVersions();
      // Сортируем по умолчанию по времени изменения (desc)
      sortVersionsByChangeDate(data, 'desc');
    } catch (err) {
      const errorMessage = err.response 
        ? `Ошибка ${err.response.status}: ${err.response.data?.error || err.message}`
        : err.message || 'Не удалось подключиться к серверу. Проверьте, что API сервер запущен.';
      setError('Ошибка при загрузке данных: ' + errorMessage);
    } finally {
      setLoading(false);
    }
  }, [sortVersionsByChangeDate]);

  useEffect(() => {
    loadVersions();
  }, [loadVersions]);

  const handleSort = () => {
    const newOrder = sortOrder === 'desc' ? 'asc' : 'desc';
    setSortOrder(newOrder);
    sortVersionsByChangeDate(versions, newOrder);
  };

  if (loading) {
    return <div className="loading">Загрузка...</div>;
  }

  if (error) {
    return (
      <div className="error">
        <p>{error}</p>
        <button onClick={loadVersions} style={{ marginTop: '10px', padding: '8px 16px', cursor: 'pointer' }}>
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
              Время изменения
              <button className="sort-button" onClick={handleSort} title="Сортировать">
                {sortOrder === 'desc' ? '↓' : '↑'}
              </button>
            </th>
            <th>Время создания</th>
          </tr>
        </thead>
        <tbody>
          {versions.length === 0 ? (
            <tr>
              <td colSpan="3" className="empty">Нет данных</td>
            </tr>
          ) : (
            versions.map((version) => (
              <tr key={version.id}>
                <td>{version.device_name}</td>
                <td>{formatDateTime(version.version_date)}</td>
                <td>{formatDateTime(version.created_at)}</td>
              </tr>
            ))
          )}
        </tbody>
      </table>
    </div>
  );
};

export default VersionsTab;

