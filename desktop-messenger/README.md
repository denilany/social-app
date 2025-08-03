# Ripple Messenger - Desktop App

A cross-platform desktop messenger application built with Electron for the Ripple Social Network.

## Features

- **Cross-Platform**: Works on Windows, macOS, and Linux
- **Real-time Messaging**: WebSocket-based instant messaging
- **Persistent Sessions**: Stay logged in across app restarts
- **Offline Mode**: View previous messages when offline
- **Desktop Notifications**: Native system notifications for new messages
- **System Tray Integration**: Minimize to tray and quick access
- **Search**: Search through message history
- **Emoji Support**: Built-in emoji picker
- **Typing Indicators**: See when contacts are typing
- **Online Status**: Real-time online/offline presence

## Prerequisites

- Node.js 16+ 
- Running Ripple backend server (localhost:8080)
- Running Ripple frontend (localhost:3000) for registration

## Installation

1. Navigate to the desktop messenger directory:
```bash
cd desktop-messenger
```

2. Install dependencies:
```bash
npm install
```

## Development

Run in development mode:
```bash
npm run dev
```

## Building

Build for all platforms:
```bash
npm run build
```

Build for specific platforms:
```bash
npm run build-win    # Windows
npm run build-mac    # macOS  
npm run build-linux  # Linux
```

## Architecture

### Main Process (`src/main.js`)
- Window management
- System tray integration
- IPC communication
- Native notifications

### Renderer Process (`src/renderer/`)
- User interface (HTML/CSS/JS)
- WebSocket communication
- Local data storage
- Chat functionality

### Key Components

- **Storage**: Persistent local storage using electron-store
- **WebSocketManager**: Real-time communication with auto-reconnect
- **AuthManager**: Authentication and session management
- **ChatManager**: Message handling and UI updates
- **App**: Main application orchestrator

## Storage

The app uses IndexedDB-like storage via electron-store for:
- Authentication tokens
- User data
- Message history
- App preferences

Data is stored locally and persists across app restarts.

## WebSocket Communication

Connects to the backend WebSocket server at `ws://localhost:8080/ws` with token-based authentication.

Message types handled:
- `message`: Chat messages
- `user_status`: Online/offline status updates
- `typing`: Typing indicators
- `notification`: System notifications

## Security

- Context isolation enabled
- Node integration disabled
- Secure IPC communication
- Token-based authentication

## Offline Functionality

- Shows offline indicator when disconnected
- Disables message sending
- Allows viewing cached messages
- Automatic reconnection when online

## Platform-Specific Features

- **macOS**: Dock badge for unread messages
- **Windows/Linux**: System tray notifications
- **All**: Native desktop notifications