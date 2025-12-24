import React, { useState } from 'react';
import './App.css';
import DevicesTab from './components/DevicesTab';
import VersionsTab from './components/VersionsTab';
import ChangesTab from './components/ChangesTab';

function App() {
  const [activeTab, setActiveTab] = useState('devices');

  return (
    <div className="App">
      <header className="App-header">
        <h1>BlackBox</h1>
      </header>
      <div className="tabs-container">
        <button
          className={`tab-button ${activeTab === 'devices' ? 'active' : ''}`}
          onClick={() => setActiveTab('devices')}
        >
          Девайсы
        </button>
        <button
          className={`tab-button ${activeTab === 'versions' ? 'active' : ''}`}
          onClick={() => setActiveTab('versions')}
        >
          Версии
        </button>
        <button
          className={`tab-button ${activeTab === 'changes' ? 'active' : ''}`}
          onClick={() => setActiveTab('changes')}
        >
          Изменения
        </button>
      </div>
      <div className="tab-content">
        {activeTab === 'devices' && <DevicesTab />}
        {activeTab === 'versions' && <VersionsTab />}
        {activeTab === 'changes' && <ChangesTab />}
      </div>
    </div>
  );
}

export default App;

