import React, { useState, useEffect, useCallback } from 'react';
import { getVersions, getVersionDiff } from '../utils/api';
import { formatDateTime } from '../utils/dateFormatter';
import './ChangesTab.css';

const ChangesTab = () => {
  const [versions, setVersions] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [selectedVersion1, setSelectedVersion1] = useState(null);
  const [selectedVersion2, setSelectedVersion2] = useState(null);
  const [diffData, setDiffData] = useState(null);
  const [diffLoading, setDiffLoading] = useState(false);
  const [diffError, setDiffError] = useState(null);

  const loadVersions = useCallback(async () => {
    try {
      setLoading(true);
      setError(null);
      const data = await getVersions();
      setVersions(data);
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
    loadVersions();
  }, [loadVersions]);

  const handleCompare = async () => {
    if (!selectedVersion1 || !selectedVersion2) {
      setDiffError('Выберите обе версии для сравнения');
      return;
    }

    if (selectedVersion1 === selectedVersion2) {
      setDiffError('Выберите разные версии для сравнения');
      return;
    }

    try {
      setDiffLoading(true);
      setDiffError(null);
      const diff = await getVersionDiff(selectedVersion1, selectedVersion2);
      setDiffData(diff);
    } catch (err) {
      const errorMessage = err.response 
        ? `Ошибка ${err.response.status}: ${err.response.data?.error || err.message}`
        : err.message || 'Не удалось получить diff';
      setDiffError('Ошибка при получении diff: ' + errorMessage);
      setDiffData(null);
    } finally {
      setDiffLoading(false);
    }
  };

  const getVersionInfo = (versionId) => {
    return versions.find(v => v.id === parseInt(versionId));
  };

  const renderDiffLine = (line, index, side) => {
    const className = `diff-line diff-line-${line.type}`;
    const lineNum = line.line_num || '';
    
    return (
      <div key={index} className={className}>
        <span className="diff-line-number">{lineNum}</span>
        <span className="diff-line-content">{line.content || ' '}</span>
      </div>
    );
  };

  if (loading) {
    return <div className="loading">Загрузка версий...</div>;
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

  const version1Info = selectedVersion1 ? getVersionInfo(selectedVersion1) : null;
  const version2Info = selectedVersion2 ? getVersionInfo(selectedVersion2) : null;

  return (
    <div className="changes-container">
      <div className="changes-header">
        <h2>Сравнение версий конфигов</h2>
      </div>

      <div className="version-selectors">
        <div className="version-selector">
          <label htmlFor="version1">Версия 1 (левая):</label>
          <select
            id="version1"
            value={selectedVersion1 || ''}
            onChange={(e) => setSelectedVersion1(e.target.value)}
            className="version-select"
          >
            <option value="">Выберите версию...</option>
            {versions.map((version) => (
              <option key={version.id} value={version.id}>
                {version.device_name} - {formatDateTime(version.version_date)}
              </option>
            ))}
          </select>
          {version1Info && (
            <div className="version-info">
              <small>
                {version1Info.device_name} | 
                Изменено: {formatDateTime(version1Info.version_date)} | 
                Создано: {formatDateTime(version1Info.created_at)}
              </small>
            </div>
          )}
        </div>

        <div className="version-selector">
          <label htmlFor="version2">Версия 2 (правая):</label>
          <select
            id="version2"
            value={selectedVersion2 || ''}
            onChange={(e) => setSelectedVersion2(e.target.value)}
            className="version-select"
          >
            <option value="">Выберите версию...</option>
            {versions.map((version) => (
              <option key={version.id} value={version.id}>
                {version.device_name} - {formatDateTime(version.version_date)}
              </option>
            ))}
          </select>
          {version2Info && (
            <div className="version-info">
              <small>
                {version2Info.device_name} | 
                Изменено: {formatDateTime(version2Info.version_date)} | 
                Создано: {formatDateTime(version2Info.created_at)}
              </small>
            </div>
          )}
        </div>
      </div>

      <div className="compare-button-container">
        <button
          onClick={handleCompare}
          disabled={!selectedVersion1 || !selectedVersion2 || diffLoading}
          className="compare-button"
        >
          {diffLoading ? 'Сравнение...' : 'Сравнить версии'}
        </button>
      </div>

      {diffError && (
        <div className="error" style={{ marginTop: '20px' }}>
          {diffError}
        </div>
      )}

      {diffData && (
        <div className="diff-container">
          <div className="diff-header">
            <div className="diff-header-left">
              <h3>Версия {diffData.left_version_id}</h3>
              {version1Info && (
                <small>{version1Info.device_name} - {formatDateTime(version1Info.version_date)}</small>
              )}
            </div>
            <div className="diff-header-right">
              <h3>Версия {diffData.right_version_id}</h3>
              {version2Info && (
                <small>{version2Info.device_name} - {formatDateTime(version2Info.version_date)}</small>
              )}
            </div>
          </div>

          <div className="diff-content">
            <div className="diff-side diff-side-left">
              <div className="diff-lines">
                {diffData.lines.map((line, index) => {
                  if (line.type === 'removed' || line.type === 'unchanged') {
                    return renderDiffLine(line, index, 'left');
                  } else if (line.type === 'added') {
                    // Показываем пустую строку для добавленных в левой стороне
                    return (
                      <div key={index} className="diff-line diff-line-empty">
                        <span className="diff-line-number"></span>
                        <span className="diff-line-content"></span>
                      </div>
                    );
                  }
                  return null;
                })}
              </div>
            </div>

            <div className="diff-side diff-side-right">
              <div className="diff-lines">
                {diffData.lines.map((line, index) => {
                  if (line.type === 'added' || line.type === 'unchanged') {
                    return renderDiffLine(line, index, 'right');
                  } else if (line.type === 'removed') {
                    // Показываем пустую строку для удаленных в правой стороне
                    return (
                      <div key={index} className="diff-line diff-line-empty">
                        <span className="diff-line-number"></span>
                        <span className="diff-line-content"></span>
                      </div>
                    );
                  }
                  return null;
                })}
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

export default ChangesTab;
