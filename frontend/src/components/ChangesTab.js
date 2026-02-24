import React, { useState, useEffect, useCallback, useMemo } from 'react';
import { getVersions, getVersionDiff } from '../utils/api';
import { formatDateTime } from '../utils/dateFormatter';
import './ChangesTab.css';

const ChangesTab = ({ embedded = false, initialDiffData = null }) => {
  const [versions, setVersions] = useState([]);
  const [loading, setLoading] = useState(!embedded);
  const [error, setError] = useState(null);
  const [selectedVersion1, setSelectedVersion1] = useState(null);
  const [selectedVersion2, setSelectedVersion2] = useState(null);
  const [diffData, setDiffData] = useState(initialDiffData);
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
    if (embedded) {
      setLoading(false);
      if (initialDiffData) {
        setDiffData(initialDiffData);
      }
    } else {
      loadVersions();
    }
  }, [loadVersions, embedded, initialDiffData]);

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

  const processedDiff = useMemo(() => {
    if (!diffData || !diffData.lines || diffData.lines.length === 0) {
      return null;
    }

    const lines = diffData.lines;
    
    const stats = {
      added: 0,
      removed: 0,
    };

    lines.forEach(line => {
      if (line.type === 'added') stats.added++;
      if (line.type === 'removed') stats.removed++;
    });

    if (stats.added === 0 && stats.removed === 0) {
      return { identical: true };
    }

    const processedLines = [];
    let leftLineNum = 1;
    let rightLineNum = 1;

    lines.forEach((line) => {
      processedLines.push({
        ...line,
        leftLineNum: line.type === 'removed' || line.type === 'unchanged' ? leftLineNum++ : null,
        rightLineNum: line.type === 'added' || line.type === 'unchanged' ? rightLineNum++ : null,
      });
    });

    return {
      identical: false,
      stats,
      lines: processedLines,
      totalLines: lines.length,
    };
  }, [diffData]);

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
      {!embedded && (
        <>
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
                    {version.device_hostname} - {formatDateTime(version.created_at)}
                  </option>
                ))}
              </select>
              {version1Info && (
                <div className="version-info">
                  <small>
                    {version1Info.device_hostname} | 
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
                {version.device_hostname} - {formatDateTime(version.created_at)}
              </option>
            ))}
          </select>
          {version2Info && (
            <div className="version-info">
              <small>
                {version2Info.device_hostname} | 
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
        </>
      )}

      {diffError && (
        <div className="error" style={{ marginTop: '20px' }}>
          {diffError}
        </div>
      )}

      {diffData && processedDiff && (
        <div className="diff-container">
          {processedDiff.identical ? (
            <div className="diff-identical">
              Версии идентичны — изменений нет
            </div>
          ) : (
            <>
              <div className="diff-stats">
                <span className="diff-stat-added">+{processedDiff.stats.added}</span>
                <span className="diff-stat-removed">-{processedDiff.stats.removed}</span>
                <span className="diff-total-lines">
                  Всего строк: {processedDiff.totalLines}
                </span>
              </div>

              <div className="diff-table-header">
                <div className="diff-header-cell">
                  {version1Info ? `${version1Info.device_hostname} (${formatDateTime(version1Info.created_at)})` : `Версия ${diffData.left_version_id}`}
                </div>
                <div className="diff-header-cell">
                  {version2Info ? `${version2Info.device_hostname} (${formatDateTime(version2Info.created_at)})` : `Версия ${diffData.right_version_id}`}
                </div>
              </div>

              <div className="diff-table">
                {processedDiff.lines.map((line, idx) => (
                    <div key={`line-${idx}`} className={`diff-row diff-row-${line.type}`}>
                      <div className="diff-cell diff-cell-left">
                        <span className="diff-line-number">
                          {line.leftLineNum || ''}
                        </span>
                        <span className="diff-line-content">
                          {line.leftLineNum 
                            ? (line.type === 'removed' ? '−' : '') + (line.content || '') 
                            : ''}
                        </span>
                      </div>
                      <div className="diff-cell diff-cell-right">
                        <span className="diff-line-number">
                          {line.rightLineNum || ''}
                        </span>
                        <span className="diff-line-content">
                          {line.rightLineNum 
                            ? (line.type === 'added' ? '+' : '') + (line.content || '')
                            : ''}
                        </span>
                      </div>
                    </div>
                ))}
              </div>
            </>
          )}
        </div>
      )}
    </div>
  );
};

export default ChangesTab;
