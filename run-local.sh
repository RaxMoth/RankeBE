#!/bin/bash

# Quick Start Script for Gin REST API
# This script sets up the environment and starts the API in one command

set -e

export PATH="/opt/homebrew/opt/mysql@8.0/bin:$PATH"

cd "$(dirname "$0")"

echo "🚀 Gin REST API - Quick Start"
echo ""

# Check if MySQL is running
echo "🔍 Checking MySQL service..."
if ! pgrep -x "mysqld" > /dev/null; then
    echo "Starting MySQL..."
    brew services start mysql@8.0
    sleep 2
fi

# Build if needed
if [ ! -f "bin/gin-rest-api" ]; then
    echo "📦 Building application..."
    make build
fi

# Load environment variables
echo "🔧 Loading configuration..."
export $(cat .env.local | grep -v "^\#" | xargs)

# Start the application
echo ""
echo "✅ Starting API server on http://localhost:${PORT:-8080}"
echo ""
echo "Available endpoints:"
echo "  GET    /health                  - Health check"
echo "  POST   /api/v1/auth/register    - Register user"
echo "  POST   /api/v1/auth/login       - Login"
echo "  GET    /api/v1/albums           - List albums (requires auth)"
echo "  POST   /api/v1/albums           - Create album (requires auth)"
echo ""
echo "Press Ctrl+C to stop the server"
echo ""

./bin/gin-rest-api
