import React, { useState, useEffect } from 'react';
import { uploadFile, downloadFile, listFiles, getPresignedUrl, deleteFile } from '../utils/api';

const FileManager = () => {
  const [files, setFiles] = useState([]);
  const [uploading, setUploading] = useState(false);
  const [selectedFile, setSelectedFile] = useState(null);

  useEffect(() => {
    loadFiles();
  }, []);

  const loadFiles = async () => {
    try {
      const response = await listFiles();
      setFiles(response.files || []);
    } catch (error) {
      console.error('Failed to load files:', error);
    }
  };

  const handleFileUpload = async (event) => {
    const file = event.target.files[0];
    if (!file) return;

    setUploading(true);
    try {
      await uploadFile(file);
      await loadFiles();
      alert('File uploaded successfully!');
    } catch (error) {
      console.error('Upload failed:', error);
      alert('Failed to upload file');
    } finally {
      setUploading(false);
    }
  };

  const handleDownload = async (objectName) => {
    try {
      const blob = await downloadFile(objectName);
      const url = window.URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = objectName;
      document.body.appendChild(a);
      a.click();
      window.URL.revokeObjectURL(url);
      document.body.removeChild(a);
    } catch (error) {
      console.error('Download failed:', error);
      alert('Failed to download file');
    }
  };

  const handleGetUrl = async (objectName) => {
    try {
      const response = await getPresignedUrl(objectName);
      alert(`Presigned URL: ${response.url}`);
    } catch (error) {
      console.error('Failed to get URL:', error);
      alert('Failed to generate URL');
    }
  };

  const handleDelete = async (objectName) => {
    if (!window.confirm(`Are you sure you want to delete ${objectName}?`)) {
      return;
    }

    try {
      await deleteFile(objectName);
      await loadFiles();
      alert('File deleted successfully!');
    } catch (error) {
      console.error('Delete failed:', error);
      alert('Failed to delete file');
    }
  };

  return (
    <div style={{ padding: '20px' }}>
      <h2>File Manager (MinIO)</h2>
      
      <div style={{ marginBottom: '20px' }}>
        <h3>Upload File</h3>
        <input
          type="file"
          onChange={handleFileUpload}
          disabled={uploading}
        />
        {uploading && <span style={{ marginLeft: '10px' }}>Uploading...</span>}
      </div>

      <div>
        <h3>Files</h3>
        {files.length === 0 ? (
          <p>No files found</p>
        ) : (
          <div style={{ display: 'grid', gap: '10px' }}>
            {files.map((file, index) => (
              <div
                key={index}
                style={{
                  border: '1px solid #ccc',
                  padding: '10px',
                  borderRadius: '5px',
                  display: 'flex',
                  justifyContent: 'space-between',
                  alignItems: 'center'
                }}
              >
                <span>{file}</span>
                <div>
                  <button
                    onClick={() => handleDownload(file)}
                    style={{ marginRight: '5px' }}
                  >
                    Download
                  </button>
                  <button
                    onClick={() => handleGetUrl(file)}
                    style={{ marginRight: '5px' }}
                  >
                    Get URL
                  </button>
                  <button
                    onClick={() => handleDelete(file)}
                    style={{ backgroundColor: '#ff4444', color: 'white' }}
                  >
                    Delete
                  </button>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
};

export default FileManager;