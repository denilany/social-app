# Ripple Social Media Platform

Ripple is a modern social media platform built with Go (backend) and Next.js (frontend).

## Features

- User authentication and profile management
- Post creation with privacy settings
- Image uploads for posts and comments
- Like/unlike functionality
- Commenting system
- Follow/unfollow users
- Feed with pagination
- **Cross-platform desktop messenger app**
  - Real-time messaging with WebSocket
  - Offline message viewing
  - Desktop notifications
  - System tray integration
  - Emoji picker and typing indicators

## Project Structure

### Backend (Go)

- `backend/server.go` - Main server entry point
- `backend/pkg/api/` - API handlers, middleware, and routes
- `backend/pkg/models/` - Data models and repositories
- `backend/pkg/db/migrations/` - Database migrations

### Frontend (Next.js)

- `frontend/src/pages/` - Next.js pages and API routes
- `frontend/src/components/` - React components
- `frontend/src/contexts/` - React context providers
- `frontend/src/services/` - API service layer

### Desktop Messenger (Electron)

- `desktop-messenger/src/main.js` - Main Electron process
- `desktop-messenger/src/preload.js` - Secure IPC bridge
- `desktop-messenger/src/renderer/` - Desktop app UI and logic

## Getting Started

### Prerequisites

- Go 1.16+
- Node.js 16+
- SQLite3

### Backend Setup

```bash
cd backend
go mod download
go run server.go
```

### Frontend Setup

```bash
cd frontend
npm install
npm run dev
```

### Desktop Messenger Setup

```bash
cd desktop-messenger
./setup.sh  # Linux/macOS
# OR
npm install
npm start
```

See [desktop-messenger/INSTALL.md](desktop-messenger/INSTALL.md) for detailed setup instructions.

## Committing Changes

A script has been provided to help commit changes with appropriate messages:

```bash
./commit_script.sh
```

This will stage and commit all files with descriptive commit messages organized by feature.

## License

This project is licensed under the MIT License - see the LICENSE file for details.