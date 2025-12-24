import React from 'react';
import './ChangesTab.css';

const ChangesTab = () => {
  return (
    <div className="changes-container">
      <div className="changes-header">
        <h2>Сравнение версий конфигов</h2>
        <p className="changes-description">
          Здесь будет реализовано side-by-side сравнение версий конфигов
        </p>
      </div>
      <div className="changes-content">
        <div className="comparison-placeholder">
          <div className="comparison-side left-side">
            <h3>Версия 1</h3>
            <div className="placeholder-content">
              <p>Выберите версию для сравнения</p>
            </div>
          </div>
          <div className="comparison-divider"></div>
          <div className="comparison-side right-side">
            <h3>Версия 2</h3>
            <div className="placeholder-content">
              <p>Выберите версию для сравнения</p>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};

export default ChangesTab;

