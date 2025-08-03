# Installation Guide - Ripple Messenger Desktop

## Quick Start

1. **Prerequisites Check**
   ```bash
   node --version  # Should be 16+
   npm --version   # Should be 8+
   ```

2. **Install Dependencies**
   ```bash
   cd desktop-messenger
   ./setup.sh  # On Linux/macOS
   # OR manually:
   npm install
   ```

3. **Start Backend Server**
   ```bash
   cd ../backend
   go run server.go
   ```

4. **Start Frontend Server** (for registration)
   ```bash
   cd ../frontend
   npm run dev
   ```

5. **Run Desktop App**
   ```bash
   cd ../desktop-messenger
   npm start
   ```

## First Time Setup

1. **Register Account**: If you don't have an account, click "Register on our website" in the login screen
2. **Login**: Use your email and password to login
3. **Start Chatting**: Select contacts from the sidebar to start messaging

## Troubleshooting

### Connection Issues
- Ensure backend server is running on port 8080
- Check if WebSocket connection is blocked by firewall
- Verify session cookies are being set correctly

### Avatar Issues
- Avatars are loaded from the backend server
- Default avatar will show if user hasn't set one
- Check network connectivity if avatars don't load

### Message Sync Issues
- Messages are stored locally for offline viewing
- New messages sync when connection is restored
- Clear app data if sync issues persist

### Build Issues
- Ensure all dependencies are installed: `npm install`
- Check Node.js version compatibility
- Try clearing node_modules and reinstalling

## Development Mode

Run in development mode with hot reload:
```bash
npm run dev
```

## Building for Distribution

Build for all platforms:
```bash
npm run build
```

Build for specific platform:
```bash
npm run build-win    # Windows
npm run build-mac    # macOS
npm run build-linux  # Linux
```

## Data Storage

- **Messages**: Stored locally using electron-store
- **User Data**: Cached locally, synced with server
- **Settings**: Stored in user's app data directory

Location varies by OS:
- **Windows**: `%APPDATA%/ripple-messenger`
- **macOS**: `~/Library/Application Support/ripple-messenger`
- **Linux**: `~/.config/ripple-messenger`