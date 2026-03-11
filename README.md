# REST API Template with Gin Framework

A production-ready REST API template built with Go and Gin framework. This template supports both MySQL and Firebase as database backends, includes JWT authentication, rate limiting, request logging, CORS middleware, and comes with Swagger documentation.

## Features

- ✅ **Clean Architecture** - Separation of concerns with handlers, services, and repositories
- ✅ **Database Agnostic** - Support for both MySQL and Firebase
- ✅ **JWT Authentication** - Secure authentication with access and refresh tokens
- ✅ **Rate Limiting** - IP-based rate limiting to prevent abuse
- ✅ **Request Logging** - Comprehensive request/response logging
- ✅ **CORS Support** - Cross-Origin Resource Sharing middleware
- ✅ **Swagger Documentation** - Auto-generated API documentation
- ✅ **Docker Support** - Containerized deployment with Docker Compose
- ✅ **Database Migrations** - Version-controlled database schema
- ✅ **Graceful Shutdown** - Proper server shutdown handling
- ✅ **Environment Configuration** - Easy configuration via environment variables
- ✅ **Pagination** - Built-in pagination support for list endpoints
- ✅ **Error Handling** - Consistent error responses
- ✅ **Input Validation** - Request validation using binding tags

## Project Structure

```
gin-rest-template/
├── cmd/
│   └── api/
│       └── main.go              # Application entry point
├── internal/
│   ├── config/
│   │   └── config.go            # Configuration management
│   ├── handlers/
│   │   └── handlers.go          # HTTP request handlers
│   ├── middleware/
│   │   ├── auth.go              # JWT authentication middleware
│   │   ├── cors.go              # CORS middleware
│   │   ├── logger.go            # Request logging middleware
│   │   └── rate_limiter.go      # Rate limiting middleware
│   ├── models/
│   │   └── models.go            # Data models and DTOs
│   ├── repository/
│   │   ├── repository.go        # Repository interface
│   │   ├── mysql_repository.go  # MySQL implementation
│   │   └── firebase_repository.go # Firebase implementation
│   └── service/
│       └── service.go           # Business logic layer
├── pkg/
│   └── logger/
│       └── logger.go            # Logger package
├── migrations/                   # Database migrations
├── docs/                        # Swagger documentation
├── .env.example                 # Example environment variables
├── .gitignore
├── docker-compose.yml
├── Dockerfile
├── Makefile
├── go.mod
└── README.md
```

## Getting Started

> **Quick Start**: For detailed local development setup instructions, see [LOCAL_SETUP.md](docs/LOCAL_SETUP.md)

### Prerequisites

- Go 1.21 or higher
- Docker and Docker Compose (optional)
- MySQL 8.0 or higher (if using MySQL)
- Firebase project and credentials (if using Firebase)

### Installation

1. **Clone the repository**

```bash
git clone https://github.com/yourusername/gin-rest-template.git
cd gin-rest-template
```

2. **Install dependencies**

```bash
make install
# or
go mod download
```

3. **Set up environment variables**

```bash
cp .env.example .env
```

Edit `.env` with your configuration:

```env
# For MySQL
DATABASE_TYPE=mysql
MYSQL_HOST=localhost
MYSQL_PORT=3306
MYSQL_USER=root
MYSQL_PASSWORD=yourpassword
MYSQL_DATABASE=gin_rest_db

# For Firebase
DATABASE_TYPE=firebase
FIREBASE_PROJECT_ID=your-project-id
FIREBASE_CREDENTIALS=./serviceAccountKey.json
```

### Running the Application

#### Option 1: Run Locally

```bash
make run
# or
go run cmd/api/main.go
```

#### Option 2: Run with Docker

```bash
# Build and start containers
make docker-up

# View logs
make docker-logs

# Stop containers
make docker-down
```

The API will be available at `http://localhost:8080`

## API Endpoints

### Authentication

| Method | Endpoint                | Description          | Auth Required |
| ------ | ----------------------- | -------------------- | ------------- |
| POST   | `/api/v1/auth/register` | Register new user    | No            |
| POST   | `/api/v1/auth/login`    | Login user           | No            |
| POST   | `/api/v1/auth/refresh`  | Refresh access token | No            |

### Users

| Method | Endpoint           | Description         | Auth Required |
| ------ | ------------------ | ------------------- | ------------- |
| GET    | `/api/v1/users/me` | Get current user    | Yes           |
| PUT    | `/api/v1/users/me` | Update current user | Yes           |

### Albums (Example Resource)

| Method | Endpoint             | Description      | Auth Required |
| ------ | -------------------- | ---------------- | ------------- |
| GET    | `/api/v1/albums`     | Get all albums   | Yes           |
| GET    | `/api/v1/albums/:id` | Get album by ID  | Yes           |
| POST   | `/api/v1/albums`     | Create new album | Yes           |
| PUT    | `/api/v1/albums/:id` | Update album     | Yes           |
| DELETE | `/api/v1/albums/:id` | Delete album     | Yes           |

### Health Check

| Method | Endpoint  | Description  | Auth Required |
| ------ | --------- | ------------ | ------------- |
| GET    | `/health` | Health check | No            |

## API Usage Examples

### Register a new user

```bash
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "password": "password123",
    "name": "John Doe"
  }'
```

### Login

```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "password": "password123"
  }'
```

Response:

```json
{
    "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "token_type": "Bearer",
    "expires_in": 900
}
```

### Create an album (with authentication)

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

### Get all albums with pagination

```bash
curl -X GET "http://localhost:8080/api/v1/albums?page=1&page_size=10&sort_by=created_at&order=desc" \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN"
```

## Swagger Documentation

Generate and view Swagger documentation:

```bash
# Generate Swagger docs
make swagger

# Run the API and visit
http://localhost:8080/swagger/index.html
```

## Database Configuration

### MySQL Setup

1. Create a database:

```sql
CREATE DATABASE gin_rest_db CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
```

2. Update `.env`:

```env
DATABASE_TYPE=mysql
MYSQL_HOST=localhost
MYSQL_PORT=3306
MYSQL_USER=root
MYSQL_PASSWORD=yourpassword
MYSQL_DATABASE=gin_rest_db
```

### Firebase Setup

1. Create a Firebase project at https://console.firebase.google.com

2. Enable Firestore Database

3. Generate a service account key:
    - Go to Project Settings > Service Accounts
    - Click "Generate New Private Key"
    - Save as `serviceAccountKey.json` in the project root

4. Update `.env`:

```env
DATABASE_TYPE=firebase
FIREBASE_PROJECT_ID=your-project-id
FIREBASE_CREDENTIALS=./serviceAccountKey.json
```

## Adding New Resources

To add a new resource (e.g., "Books"):

1. **Add model** in `internal/models/models.go`:

```go
type Book struct {
    ID        string    `json:"id" gorm:"primaryKey" firestore:"id"`
    Title     string    `json:"title" binding:"required"`
    Author    string    `json:"author" binding:"required"`
    ISBN      string    `json:"isbn" binding:"required"`
    UserID    string    `json:"user_id"`
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
}
```

2. **Add repository methods** in `internal/repository/repository.go`:

```go
CreateBook(ctx context.Context, book *models.Book) error
GetBookByID(ctx context.Context, id string) (*models.Book, error)
// ... other CRUD methods
```

3. **Implement repository** in both MySQL and Firebase repositories

4. **Add service methods** in `internal/service/service.go`

5. **Add handlers** in `internal/handlers/handlers.go`

6. **Register routes** in `cmd/api/main.go`:

```go
books := protected.Group("/books")
{
    books.GET("", h.GetBooks)
    books.GET("/:id", h.GetBookByID)
    books.POST("", h.CreateBook)
    books.PUT("/:id", h.UpdateBook)
    books.DELETE("/:id", h.DeleteBook)
}
```

## Testing

Run tests:

```bash
make test

# With coverage
make test-coverage
```

## Available Make Commands

```bash
make help              # Display all available commands
make install           # Install dependencies
make build             # Build the application
make run               # Run the application
make test              # Run tests
make clean             # Clean build artifacts
make docker-build      # Build Docker image
make docker-up         # Start Docker containers
make docker-down       # Stop Docker containers
make docker-logs       # View Docker logs
make swagger           # Generate Swagger documentation
make lint              # Run linter
make fmt               # Format code
```

## Configuration

All configuration is done via environment variables. See `.env.example` for all available options.

### Key Configuration Options

- `ENVIRONMENT` - Environment mode (development/production)
- `PORT` - Server port
- `DATABASE_TYPE` - Database type (mysql/firebase)
- `JWT_SECRET` - Secret key for JWT tokens (change in production!)
- `JWT_EXPIRATION` - Access token expiration time
- `JWT_REFRESH_EXPIRATION` - Refresh token expiration time
- `RATE_LIMIT_REQUESTS` - Number of requests per window
- `RATE_LIMIT_DURATION` - Rate limit time window
- `LOG_LEVEL` - Logging level (debug/info/warn/error)

## Security Best Practices

1. **Change the JWT secret** in production
2. **Use HTTPS** in production
3. **Set strong passwords** for database
4. **Enable rate limiting** to prevent abuse
5. **Keep dependencies updated**
6. **Don't commit** `.env` or Firebase credentials
7. **Use environment-specific** configurations
8. **Implement proper** CORS settings for your frontend domain

## Deployment

### Deploy with Docker

```bash
# Build image
docker build -t gin-rest-api .

# Run container
docker run -p 8080:8080 --env-file .env gin-rest-api
```

### Deploy to Cloud Platforms

The template is ready to deploy to:

- AWS (ECS, EC2, Elastic Beanstalk)
- Google Cloud Run
- Azure Container Instances
- Heroku
- DigitalOcean App Platform

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Support

For questions or issues, please open an issue on GitHub.

## Acknowledgments

- [Gin Web Framework](https://gin-gonic.com/)
- [GORM](https://gorm.io/)
- [Firebase Admin SDK](https://firebase.google.com/docs/admin/setup)
- [golang-jwt](https://github.com/golang-jwt/jwt)

---

**Happy coding! 🚀**
