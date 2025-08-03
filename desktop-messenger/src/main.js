const { app, BrowserWindow, ipcMain, Menu, Tray, nativeImage, Notification } = require('electron');
const path = require('path');
const Store = require('electron-store');

const store = new Store();
let mainWindow;
let tray;

function createWindow() {
  mainWindow = new BrowserWindow({
    width: 1200,
    height: 800,
    minWidth: 800,
    minHeight: 600,
    webPreferences: {
      nodeIntegration: false,
      contextIsolation: true,
      preload: path.join(__dirname, 'preload.js')
    },
    icon: path.join(__dirname, '../assets/icon.png'),
    show: false
  });

  mainWindow.loadFile(path.join(__dirname, 'renderer/index.html'));

  mainWindow.once('ready-to-show', () => {
    mainWindow.show();
    
    // Restore window state
    const windowState = store.get('windowState');
    if (windowState) {
      mainWindow.setBounds(windowState);
    }
  });

  mainWindow.on('close', (event) => {
    if (!app.isQuiting) {
      event.preventDefault();
      mainWindow.hide();
    }
  });

  mainWindow.on('resize', saveWindowState);
  mainWindow.on('move', saveWindowState);
}

function saveWindowState() {
  store.set('windowState', mainWindow.getBounds());
}

function createTray() {
  const iconPath = path.join(__dirname, '../assets/tray-icon.png');
  tray = new Tray(nativeImage.createFromPath(iconPath));
  
  const contextMenu = Menu.buildFromTemplate([
    {
      label: 'Show App',
      click: () => {
        mainWindow.show();
      }
    },
    {
      label: 'Quit',
      click: () => {
        app.isQuiting = true;
        app.quit();
      }
    }
  ]);

  tray.setToolTip('Ripple Messenger');
  tray.setContextMenu(contextMenu);
  
  tray.on('click', () => {
    mainWindow.isVisible() ? mainWindow.hide() : mainWindow.show();
  });
}

app.whenReady().then(() => {
  createWindow();
  createTray();

  app.on('activate', () => {
    if (BrowserWindow.getAllWindows().length === 0) {
      createWindow();
    }
  });
});

app.on('window-all-closed', () => {
  if (process.platform !== 'darwin') {
    app.quit();
  }
});

// IPC handlers
ipcMain.handle('get-stored-data', (event, key) => {
  return store.get(key);
});

ipcMain.handle('set-stored-data', (event, key, value) => {
  if (value === undefined || value === null) {
    store.delete(key);
  } else {
    store.set(key, value);
  }
});

ipcMain.handle('delete-stored-data', (event, key) => {
  return store.delete(key);
});

ipcMain.handle('show-notification', (event, options) => {
  if (Notification.isSupported()) {
    new Notification(options).show();
  }
});

ipcMain.handle('set-badge-count', (event, count) => {
  if (process.platform === 'darwin') {
    app.dock.setBadge(count.toString());
  }
});