# Local Development Setup Guide

This guide will help you run the Gin REST API locally on your machine.

## Prerequisites

- Go 1.21 or higher
- One of the following:
    - **Option A**: MySQL 8.0+ installed locally, OR
    - **Option B**: Docker & Docker Compose installed

## Quick Start (Option A - MySQL Local)

### 1. Install MySQL Locally (macOS)

If you're on macOS, use Homebrew:

```bash
# Install MySQL
brew install mysql

# Start MySQL service
brew services start mysql

# Create database and user
mysql -u root -e "CREATE DATABASE gin_rest_db;"
mysql -u root -e "CREATE USER 'gin_user'@'localhost' IDENTIFIED BY 'secure_password_123';"
mysql -u root -e "GRANT ALL PRIVILEGES ON gin_rest_db.* TO 'gin_user'@'localhost';"
mysql -u root -e "FLUSH PRIVILEGES;"
```

For Linux or other systems, follow the [official MySQL installation guide](https://dev.mysql.com/doc/mysql-installation-excerpt/en/).

### 2. Run the Application

```bash
# 1. Install Go dependencies
make install

# 2. Copy environment configuration
cp .env.local .env

# 3. Run database migrations (optional but recommended)
# Make sure golang-migrate is installed: go install -tags 'mysql' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
make local-migrate

# 4. Build and run the application
make local-run
```

The API will be available at: `http://localhost:8080`

## Quick Start (Option B - Docker)

### 1. Ensure Docker is Running

Make sure Docker daemon is running:

```bash
docker ps  # Should show running containers without errors
```

### 2. Start MySQL in Docker

```bash
make local-db
```

MySQL will start on `localhost:3306` with these credentials:

- User: `gin_user`
- Password: `secure_password_123`
- Database: `gin_rest_db`

### 3. Run the Application

```bash
# 1. Install Go dependencies
make install

# 2. Run database migrations
make local-migrate

# 3. Build and run the application
make local-run
```

The API will be available at: `http://localhost:8080`

## Useful Commands

```bash
# Build the application
make build

# Run with hot reload (requires 'air' - install with: go install github.com/cosmtrek/air@latest)
make dev

# Run tests
make test

# Run tests with coverage
make test-coverage

# View MySQL logs (Docker only)
make local-db-logs

# Stop MySQL (Docker only)
make local-db-stop

# View help
make help
```

## Testing the API

Once running, you can test the API:

### 1. Health Check

```bash
curl http://localhost:8080/health
```

### 2. Register a User

```bash
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "password": "password123",
    "name": "John Doe"
  }'
```

### 3. Login

```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "password": "password123"
  }'
```

The response will include `access_token` and `refresh_token`. Use the `access_token` for subsequent requests.

### 4. Create an Album (Protected Route)

```bash
# Replace TOKEN with the access_token from login
curl -X POST http://localhost:8080/api/v1/albums \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer TOKEN" \
  -d '{
    "title": "My Album",
    "description": "Album description"
  }'
```

## Environment Configuration

The application uses environment variables for configuration. The `.env.local` file contains the recommended settings for local development.

**Key variables:**

- `DATABASE_TYPE`: `mysql` or `firebase`
- `MYSQL_HOST`: Database host (default: `localhost`)
- `MYSQL_PORT`: Database port (default: `3306`)
- `MYSQL_USER`: Database user
- `MYSQL_PASSWORD`: Database password
- `MYSQL_DATABASE`: Database name
- `JWT_SECRET`: Secret key for JWT tokens (change in production!)
- `PORT`: API server port (default: `8080`)
- `LOG_LEVEL`: Logging level (debug, info, warn, error)

## Troubleshooting

### MySQL Connection Refused

**Problem**: `dial tcp [::1]:3306: connect: connection refused`

**Solution**:

- Make sure MySQL is running: `mysql -u root` should connect without error
- Check if port 3306 is in use: `lsof -i :3306`
- If using Docker, ensure the container is running: `docker ps | grep mysql`

### Port Already in Use

**Problem**: `listen tcp :8080: bind: address already in use`

**Solution**:

- Change the port in `.env.local`: `PORT=8081`
- Or kill the process using the port: `lsof -i :8080` and `kill -9 PID`

### Build Errors

**Problem**: Go build fails with missing dependencies

**Solution**:

```bash
go clean -modcache
go mod download
go mod tidy
make build
```

## Using Postman

A Postman collection is included in the repository:

1. Open Postman
2. Click "Import" → "File"
3. Select `postman_collection.json`
4. The collection includes example requests for all endpoints
5. After login, manually update the `Bearer TOKEN` in request headers with your JWT token

## Next Steps

- Read the [API Documentation](API.md)
- Check [ADDING_RESOURCES.md](ADDING_RESOURCES.md) to add new API endpoints
- Review [DEPLOYMENT.md](DEPLOYMENT.md) for production deployment
