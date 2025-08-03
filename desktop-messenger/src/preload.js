const { contextBridge, ipcRenderer } = require('electron');

contextBridge.exposeInMainWorld('electronAPI', {
  // Storage
  getStoredData: (key) => ipcRenderer.invoke('get-stored-data', key),
  setStoredData: (key, value) => ipcRenderer.invoke('set-stored-data', key, value),
  deleteStoredData: (key) => ipcRenderer.invoke('delete-stored-data', key),
  
  // Notifications
  showNotification: (options) => ipcRenderer.invoke('show-notification', options),
  setBadgeCount: (count) => ipcRenderer.invoke('set-badge-count', count),
  
  // Platform info
  platform: process.platform
});