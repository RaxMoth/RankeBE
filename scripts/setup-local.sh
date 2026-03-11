#!/bin/bash
set -e

echo "🚀 Setting up Gin REST API for local development..."
echo ""

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Step 1: Install MySQL if not already installed
echo "📦 Checking MySQL installation..."
if ! command -v mysql &> /dev/null; then
    echo "${YELLOW}MySQL not found. Installing...${NC}"
    
    # Try different installation methods
    if command -v brew &> /dev/null; then
        echo "Using Homebrew to install MySQL@8.0..."
        brew install mysql@8.0
        # Link mysql executable
        brew link mysql@8.0 --force
    elif command -v apt-get &> /dev/null; then
        echo "Using apt to install MySQL..."
        sudo apt-get update
        sudo apt-get install -y mysql-server
    else
        echo "${RED}Error: Neither Homebrew nor apt found. Please install MySQL manually.${NC}"
        exit 1
    fi
fi

echo "${GREEN}✓ MySQL is installed${NC}"
echo ""

# Step 2: Start MySQL service
echo "🔧 Starting MySQL service..."
if command -v brew &> /dev/null && [[ "$OSTYPE" == "darwin"* ]]; then
    echo "macOS detected. Using Homebrew services..."
    brew services start mysql@8.0 || brew services start mysql || true
    sleep 3
elif command -v systemctl &> /dev/null; then
    echo "Linux detected. Using systemctl..."
    sudo systemctl start mysql || sudo systemctl start mariadb || true
    sleep 3
fi

# Step 3: Verify MySQL is running
echo "🔍 Verifying MySQL is running..."
if mysql -u root -e "select 1" > /dev/null 2>&1; then
    echo "${GREEN}✓ MySQL is running${NC}"
else
    echo "${YELLOW}Warning: Could not connect to MySQL as root. Attempting alternative methods...${NC}"
    
    # Try with socket auth on Linux
    if [[ "$OSTYPE" == "linux-gnu"* ]]; then
        if sudo mysql -e "select 1" > /dev/null 2>&1; then
            echo "${GREEN}✓ MySQL is running (accessible via sudo)${NC}"
        else
            echo "${RED}✗ MySQL is not accessible${NC}"
            echo "Please ensure MySQL is running and accessible"
            exit 1
        fi
    fi
fi

echo ""

# Step 4: Create database and user
echo "🗄️  Creating database and user..."
SETUP_SQL="
CREATE DATABASE IF NOT EXISTS gin_rest_db;
CREATE USER IF NOT EXISTS 'gin_user'@'127.0.0.1' IDENTIFIED BY 'secure_password_123';
GRANT ALL PRIVILEGES ON gin_rest_db.* TO 'gin_user'@'127.0.0.1';
CREATE USER IF NOT EXISTS 'gin_user'@'localhost' IDENTIFIED BY 'secure_password_123';
GRANT ALL PRIVILEGES ON gin_rest_db.* TO 'gin_user'@'localhost';
FLUSH PRIVILEGES;
"

if mysql -u root -e "$SETUP_SQL" > /dev/null 2>&1; then
    echo "${GREEN}✓ Database and user created${NC}"
else
    echo "${YELLOW}Note: Database/user may already exist or require different permissions${NC}"
fi

echo ""

# Step 5: Verify database connection
echo "✓ Verifying database connection..."
if mysql -u gin_user -p'secure_password_123' -h 127.0.0.1 gin_rest_db -e "SELECT 1" > /dev/null 2>&1; then
    echo "${GREEN}✓ Database is ready${NC}"
else
    echo "${YELLOW}⚠️  Could not connect with gin_user. Some setup steps may have failed.${NC}"
fi

echo ""
echo "${GREEN}✅ Setup complete!${NC}"
echo ""
echo "Next steps:"
echo "1. Build the application:"
echo "   make build"
echo ""
echo "2. Run database migrations (if needed):"
echo "   make local-migrate"
echo ""
echo "3. Start the application:"
echo "   make local-run"
echo ""
echo "The API will be available at: http://localhost:8080"
