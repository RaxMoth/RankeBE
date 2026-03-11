package models

import (
	"time"

	"golang.org/x/crypto/bcrypt"
)

// User represents a user in the system
type User struct {
	ID        string    `json:"id" gorm:"primaryKey" firestore:"id"`
	Email     string    `json:"email" gorm:"type:varchar(255);uniqueIndex;not null" firestore:"email" binding:"required,email"`
	Password  string    `json:"-" gorm:"type:longtext;not null" firestore:"password"`
	Name      string    `json:"name" gorm:"type:varchar(255);not null" firestore:"name" binding:"required"`
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime" firestore:"created_at"`
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime" firestore:"updated_at"`
}

// UserResponse is the user model without sensitive data
type UserResponse struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ToResponse converts User to UserResponse
func (u *User) ToResponse() *UserResponse {
	return &UserResponse{
		ID:        u.ID,
		Email:     u.Email,
		Name:      u.Name,
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
	}
}

// HashPassword hashes the user's password
func (u *User) HashPassword() error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.Password = string(hashedPassword)
	return nil
}

// CheckPassword checks if the provided password is correct
func (u *User) CheckPassword(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password))
	return err == nil
}

// Album represents an album in the system
type Album struct {
	ID        string    `json:"id" gorm:"primaryKey" firestore:"id"`
	Title     string    `json:"title" gorm:"type:varchar(255);not null" firestore:"title" binding:"required"`
	Artist    string    `json:"artist" gorm:"type:varchar(255);not null" firestore:"artist" binding:"required"`
	Price     float64   `json:"price" gorm:"not null" firestore:"price" binding:"required,gt=0"`
	UserID    string    `json:"user_id" gorm:"type:varchar(255);not null;index" firestore:"user_id"`
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime" firestore:"created_at"`
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime" firestore:"updated_at"`
}

// LoginRequest represents login credentials
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

// RegisterRequest represents registration data
type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
	Name     string `json:"name" binding:"required,min=2"`
}

// UpdateUserRequest represents user update data
type UpdateUserRequest struct {
	Name  string `json:"name" binding:"omitempty,min=2"`
	Email string `json:"email" binding:"omitempty,email"`
}

// TokenResponse represents authentication tokens
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"`
}

// RefreshTokenRequest represents refresh token request
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string                 `json:"error"`
	Message string                 `json:"message,omitempty"`
	Details map[string]interface{} `json:"details,omitempty"`
}

// SuccessResponse represents a success response
type SuccessResponse struct {
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// PaginationQuery represents pagination parameters
type PaginationQuery struct {
	Page     int    `form:"page,default=1" binding:"omitempty,min=1"`
	PageSize int    `form:"page_size,default=10" binding:"omitempty,min=1,max=100"`
	SortBy   string `form:"sort_by,default=created_at"`
	Order    string `form:"order,default=desc" binding:"omitempty,oneof=asc desc"`
}

// PaginatedResponse represents a paginated response
type PaginatedResponse struct {
	Data       interface{} `json:"data"`
	Page       int         `json:"page"`
	PageSize   int         `json:"page_size"`
	TotalItems int64       `json:"total_items"`
	TotalPages int         `json:"total_pages"`
}
