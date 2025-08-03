#!/bin/bash

echo "🚀 Setting up Ripple Messenger Desktop App..."

# Check if Node.js is installed
if ! command -v node &> /dev/null; then
    echo "❌ Node.js is not installed. Please install Node.js 16+ first."
    exit 1
fi

# Check Node.js version
NODE_VERSION=$(node -v | cut -d'v' -f2 | cut -d'.' -f1)
if [ "$NODE_VERSION" -lt 16 ]; then
    echo "❌ Node.js version 16+ is required. Current version: $(node -v)"
    exit 1
fi

echo "✅ Node.js $(node -v) detected"

# Install dependencies
echo "📦 Installing dependencies..."
npm install

if [ $? -eq 0 ]; then
    echo "✅ Dependencies installed successfully"
else
    echo "❌ Failed to install dependencies"
    exit 1
fi

# Create basic icon files (placeholder)
echo "🎨 Creating placeholder icons..."
mkdir -p assets

# Check if backend is running
echo "🔍 Checking if backend server is running..."
if curl -s http://localhost:8080/health > /dev/null 2>&1; then
    echo "✅ Backend server is running"
else
    echo "⚠️  Backend server is not running. Please start it with:"
    echo "   cd ../backend && go run server.go"
fi

# Check if frontend is running
echo "🔍 Checking if frontend server is running..."
if curl -s http://localhost:3000 > /dev/null 2>&1; then
    echo "✅ Frontend server is running"
else
    echo "⚠️  Frontend server is not running. Please start it with:"
    echo "   cd ../frontend && npm run dev"
fi

echo ""
echo "🎉 Setup complete! You can now run the app with:"
echo "   npm start      # Production mode"
echo "   npm run dev    # Development mode"
echo ""
echo "📝 Make sure both backend and frontend servers are running before using the app."