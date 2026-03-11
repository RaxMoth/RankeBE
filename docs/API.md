# API Documentation

## Base URL

```
http://localhost:8080
```

## Authentication

This API uses JWT (JSON Web Tokens) for authentication. Include the token in the Authorization header:

```
Authorization: Bearer YOUR_ACCESS_TOKEN
```

## Response Format

All responses follow this structure:

**Success Response:**
```json
{
  "message": "Success message",
  "data": { ... }
}
```

**Error Response:**
```json
{
  "error": "error_code",
  "message": "Error description",
  "details": { ... }
}
```

**Paginated Response:**
```json
{
  "data": [...],
  "page": 1,
  "page_size": 10,
  "total_items": 100,
  "total_pages": 10
}
```

## Rate Limiting

- **Limit:** 100 requests per minute per IP
- **Headers:** Rate limit info in response headers

## Status Codes

- `200` - Success
- `201` - Created
- `400` - Bad Request
- `401` - Unauthorized
- `403` - Forbidden
- `404` - Not Found
- `409` - Conflict
- `429` - Too Many Requests
- `500` - Internal Server Error

## Endpoints

### Authentication

#### Register User

```http
POST /api/v1/auth/register
Content-Type: application/json

{
  "email": "user@example.com",
  "password": "password123",
  "name": "John Doe"
}
```

**Response (201):**
```json
{
  "message": "User registered successfully",
  "data": {
    "id": "uuid",
    "email": "user@example.com",
    "name": "John Doe",
    "created_at": "2024-01-01T00:00:00Z",
    "updated_at": "2024-01-01T00:00:00Z"
  }
}
```

#### Login

```http
POST /api/v1/auth/login
Content-Type: application/json

{
  "email": "user@example.com",
  "password": "password123"
}
```

**Response (200):**
```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "token_type": "Bearer",
  "expires_in": 900
}
```

#### Refresh Token

```http
POST /api/v1/auth/refresh
Content-Type: application/json

{
  "refresh_token": "your-refresh-token"
}
```

**Response (200):**
```json
{
  "access_token": "new-access-token",
  "refresh_token": "new-refresh-token",
  "token_type": "Bearer",
  "expires_in": 900
}
```

### User Operations

#### Get Current User

```http
GET /api/v1/users/me
Authorization: Bearer YOUR_ACCESS_TOKEN
```

**Response (200):**
```json
{
  "message": "User retrieved successfully",
  "data": {
    "id": "uuid",
    "email": "user@example.com",
    "name": "John Doe",
    "created_at": "2024-01-01T00:00:00Z",
    "updated_at": "2024-01-01T00:00:00Z"
  }
}
```

#### Update Current User

```http
PUT /api/v1/users/me
Authorization: Bearer YOUR_ACCESS_TOKEN
Content-Type: application/json

{
  "name": "Jane Doe",
  "email": "jane@example.com"
}
```

### Album Operations

#### Create Album

```http
POST /api/v1/albums
Authorization: Bearer YOUR_ACCESS_TOKEN
Content-Type: application/json

{
  "title": "Abbey Road",
  "artist": "The Beatles",
  "price": 24.99
}
```

**Response (201):**
```json
{
  "message": "Album created successfully",
  "data": {
    "id": "uuid",
    "title": "Abbey Road",
    "artist": "The Beatles",
    "price": 24.99,
    "user_id": "user-uuid",
    "created_at": "2024-01-01T00:00:00Z",
    "updated_at": "2024-01-01T00:00:00Z"
  }
}
```

#### Get All Albums

```http
GET /api/v1/albums?page=1&page_size=10&sort_by=created_at&order=desc
Authorization: Bearer YOUR_ACCESS_TOKEN
```

**Query Parameters:**
- `page` (int, optional) - Page number (default: 1)
- `page_size` (int, optional) - Items per page (default: 10, max: 100)
- `sort_by` (string, optional) - Sort field (default: created_at)
- `order` (string, optional) - Sort order: asc/desc (default: desc)

**Response (200):**
```json
{
  "data": [
    {
      "id": "uuid",
      "title": "Abbey Road",
      "artist": "The Beatles",
      "price": 24.99,
      "user_id": "user-uuid",
      "created_at": "2024-01-01T00:00:00Z",
      "updated_at": "2024-01-01T00:00:00Z"
    }
  ],
  "page": 1,
  "page_size": 10,
  "total_items": 1,
  "total_pages": 1
}
```

#### Get Album by ID

```http
GET /api/v1/albums/:id
Authorization: Bearer YOUR_ACCESS_TOKEN
```

**Response (200):**
```json
{
  "message": "Album retrieved successfully",
  "data": {
    "id": "uuid",
    "title": "Abbey Road",
    "artist": "The Beatles",
    "price": 24.99,
    "user_id": "user-uuid",
    "created_at": "2024-01-01T00:00:00Z",
    "updated_at": "2024-01-01T00:00:00Z"
  }
}
```

#### Update Album

```http
PUT /api/v1/albums/:id
Authorization: Bearer YOUR_ACCESS_TOKEN
Content-Type: application/json

{
  "title": "Abbey Road (Remastered)",
  "artist": "The Beatles",
  "price": 29.99
}
```

**Response (200):**
```json
{
  "message": "Album updated successfully",
  "data": {
    "id": "uuid",
    "title": "Abbey Road (Remastered)",
    "artist": "The Beatles",
    "price": 29.99,
    "user_id": "user-uuid",
    "created_at": "2024-01-01T00:00:00Z",
    "updated_at": "2024-01-01T12:00:00Z"
  }
}
```

#### Delete Album

```http
DELETE /api/v1/albums/:id
Authorization: Bearer YOUR_ACCESS_TOKEN
```

**Response (200):**
```json
{
  "message": "Album deleted successfully"
}
```

## Error Examples

### Validation Error
```json
{
  "error": "invalid_request",
  "message": "Key: 'RegisterRequest.Email' Error:Field validation for 'Email' failed on the 'email' tag"
}
```

### Authentication Error
```json
{
  "error": "unauthorized",
  "message": "Invalid or expired token"
}
```

### Not Found Error
```json
{
  "error": "retrieval_failed",
  "message": "album not found"
}
```

### Rate Limit Error
```json
{
  "error": "rate_limit_exceeded",
  "message": "Too many requests. Please try again later."
}
```

## Postman Collection

Import the `postman_collection.json` file into Postman for easy testing.

## WebSocket Support

Currently not implemented. Future feature.

## Versioning

API version is included in the URL path: `/api/v1/`

## Support

For issues or questions, please create an issue on GitHub.
