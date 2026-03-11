package repository

import (
	"context"
	"gin-rest-template/internal/models"
)

// Repository defines the interface for data access
type Repository interface {
	// User operations
	CreateUser(ctx context.Context, user *models.User) error
	GetUserByID(ctx context.Context, id string) (*models.User, error)
	GetUserByEmail(ctx context.Context, email string) (*models.User, error)
	UpdateUser(ctx context.Context, user *models.User) error
	DeleteUser(ctx context.Context, id string) error

	// Album operations
	CreateAlbum(ctx context.Context, album *models.Album) error
	GetAlbumByID(ctx context.Context, id string) (*models.Album, error)
	GetAlbums(ctx context.Context, userID string, pagination *models.PaginationQuery) ([]*models.Album, int64, error)
	UpdateAlbum(ctx context.Context, album *models.Album) error
	DeleteAlbum(ctx context.Context, id string) error

	// Utility
	Close() error
	HealthCheck(ctx context.Context) error
}
