# Quick Start Guide

Get your REST API up and running in 5 minutes!

## Prerequisites

- Go 1.21+ installed
- Docker & Docker Compose (for containerized setup)
- MySQL 8.0+ (for local MySQL setup)

## Option 1: Quick Start with Docker (Recommended)

This is the fastest way to get started with everything pre-configured.

```bash
# 1. Clone the repository
git clone https://github.com/yourusername/gin-rest-template.git
cd gin-rest-template

# 2. Start the application with Docker Compose
make docker-up

# 3. Wait for containers to be ready (about 30 seconds)
# The API will be available at http://localhost:8080
```

That's it! Skip to [Testing Your API](#testing-your-api).

## Option 2: Local Development Setup

### With MySQL

```bash
# 1. Clone and install dependencies
git clone https://github.com/yourusername/gin-rest-template.git
cd gin-rest-template
make install

# 2. Start MySQL (if not running)
# Using Docker:
docker run --name mysql-dev -e MYSQL_ROOT_PASSWORD=rootpassword -e MYSQL_DATABASE=gin_rest_db -p 3306:3306 -d mysql:8.0

# OR install MySQL locally on your system

# 3. Configure environment
cp .env.example .env

# Edit .env:
DATABASE_TYPE=mysql
MYSQL_HOST=localhost
MYSQL_PORT=3306
MYSQL_USER=root
MYSQL_PASSWORD=rootpassword
MYSQL_DATABASE=gin_rest_db

# 4. Run the application
make run

# The API will be available at http://localhost:8080
```

### With Firebase

```bash
# 1. Clone and install dependencies
git clone https://github.com/yourusername/gin-rest-template.git
cd gin-rest-template
make install

# 2. Set up Firebase
# - Create a project at https://console.firebase.google.com
# - Enable Firestore Database
# - Generate service account key (Settings > Service Accounts > Generate New Private Key)
# - Save as serviceAccountKey.json in project root

# 3. Configure environment
cp .env.example .env

# Edit .env:
DATABASE_TYPE=firebase
FIREBASE_PROJECT_ID=your-project-id
FIREBASE_CREDENTIALS=./serviceAccountKey.json

# 4. Run the application
make run

# The API will be available at http://localhost:8080
```

## Testing Your API

### 1. Check Health

```bash
curl http://localhost:8080/health
```

Expected response:
```json
{
  "message": "Service is healthy",
  "data": {
    "status": "ok",
    "version": "1.0.0"
  }
}
```

### 2. Register a User

```bash
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "test@example.com",
    "password": "password123",
    "name": "Test User"
  }'
```

### 3. Login

```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "test@example.com",
    "password": "password123"
  }'
```

Save the `access_token` from the response.

### 4. Create an Album

```bash
curl -X POST http://localhost:8080/api/v1/albums \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN" \
  -d '{
    "title": "Abbey Road",
    "artist": "The Beatles",
    "price": 24.99
  }'
```

### 5. Get All Albums

```bash
curl http://localhost:8080/api/v1/albums \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN"
```

## Using Postman

1. Import `postman_collection.json` into Postman
2. Run the "Register" request
3. Run the "Login" request (tokens will be saved automatically)
4. Try other requests with authenticated access

## View API Documentation

Open Swagger UI:
```
http://localhost:8080/swagger/index.html
```

(Available in development mode only)

## Next Steps

1. **Customize the template** - Add your own resources and business logic
2. **Configure security** - Change JWT secret, set up HTTPS
3. **Add features** - Implement your specific requirements
4. **Deploy** - Check out the deployment guide

## Common Issues

### Port Already in Use

If port 8080 is already in use:
```bash
# Change port in .env
PORT=8081

# Or for Docker:
docker-compose down
# Edit docker-compose.yml to change port
docker-compose up -d
```

### Database Connection Failed

**MySQL:**
- Check if MySQL is running
- Verify credentials in `.env`
- Ensure database exists

**Firebase:**
- Check if credentials file path is correct
- Verify Firebase project ID
- Ensure Firestore is enabled

### "Command not found" Errors

```bash
# Make sure Go is installed
go version

# Make sure dependencies are installed
make install
```

## Getting Help

- Check the [main README](README.md) for detailed documentation
- Review [API documentation](docs/API.md)
- Open an issue on GitHub

## What's Next?

- Read the [full documentation](README.md)
- Learn how to [add new resources](README.md#adding-new-resources)
- Check out the [deployment guide](docs/DEPLOYMENT.md)
- Explore [best practices](CONTRIBUTING.md)

Happy coding! 🚀
