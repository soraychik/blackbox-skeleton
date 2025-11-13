import React, { useState, useEffect } from 'react';
import axios from 'axios';

//const API_BASE = 'http://localhost:8080';
const API_BASE = process.env.REACT_APP_API_URL || 'http://localhost:8080';
console.log('API URL:', API_BASE);

function App() {
  const [devices, setDevices] = useState([]);
  const [versions, setVersions] = useState([]);
  const [activeTab, setActiveTab] = useState('devices');

  useEffect(() => {
    fetchDevices();
    fetchVersions();
  }, []);

  const fetchDevices = async () => {
    try {
      const response = await axios.get(`${API_BASE}/devices`);
      setDevices(response.data.devices || []);
    } catch (error) {
      console.error('Error fetching devices:', error);
    }
  };

  const fetchVersions = async () => {
    try {
      const response = await axios.get(`${API_BASE}/versions`);
      setVersions(response.data.versions || []);
    } catch (error) {
      console.error('Error fetching versions:', error);
    }
  };

  return (
    <div style={{ padding: '20px', fontFamily: 'Arial, sans-serif' }}>
      <h1>BlackBox Config Manager</h1>
      
      <div style={{ marginBottom: '20px' }}>
        <button 
          onClick={() => setActiveTab('devices')}
          style={{ marginRight: '10px', padding: '10px' }}
        >
          Devices ({devices.length})
        </button>
        <button 
          onClick={() => setActiveTab('versions')}
          style={{ padding: '10px' }}
        >
          Config Versions ({versions.length})
        </button>
      </div>

      {activeTab === 'devices' && (
        <div>
          <h2>Devices</h2>
          <table border="1" style={{ borderCollapse: 'collapse', width: '100%' }}>
            <thead>
              <tr>
                <th>ID</th>
                <th>Name</th>
                <th>Created At</th>
              </tr>
            </thead>
            <tbody>
              {devices.map(device => (
                <tr key={device.id}>
                  <td>{device.id}</td>
                  <td>{device.name}</td>
                  <td>{new Date(device.created_at).toLocaleString()}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {activeTab === 'versions' && (
        <div>
          <h2>Config Versions</h2>
          <table border="1" style={{ borderCollapse: 'collapse', width: '100%' }}>
            <thead>
              <tr>
                <th>ID</th>
                <th>Device</th>
                <th>Version Date</th>
                <th>File Hash</th>
                <th>Created At</th>
              </tr>
            </thead>
            <tbody>
              {versions.map(version => (
                <tr key={version.id}>
                  <td>{version.id}</td>
                  <td>{version.device_name} (ID: {version.device_id})</td>
                  <td>{new Date(version.version_date).toLocaleString()}</td>
                  <td title={version.file_hash}>
                    {version.file_hash?.substring(0, 8)}...
                  </td>
                  <td>{new Date(version.created_at).toLocaleString()}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}

export default App;