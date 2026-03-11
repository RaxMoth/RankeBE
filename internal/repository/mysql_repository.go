package repository

import (
	"context"
	"fmt"
	"time"

	"gin-rest-template/internal/config"
	"gin-rest-template/internal/models"

	"github.com/google/uuid"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type MySQLRepository struct {
	db *gorm.DB
}

func NewMySQLRepository(cfg *config.Config) (*MySQLRepository, error) {
	// Configure GORM logger
	gormLogger := logger.Default
	if cfg.Environment == "production" {
		gormLogger = logger.Default.LogMode(logger.Silent)
	}

	db, err := gorm.Open(mysql.Open(cfg.GetMySQLDSN()), &gorm.Config{
		Logger: gormLogger,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Configure connection pool
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database instance: %w", err)
	}

	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	// Auto-migrate models
	if err := db.AutoMigrate(&models.User{}, &models.Album{}); err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	return &MySQLRepository{db: db}, nil
}

// User operations
func (r *MySQLRepository) CreateUser(ctx context.Context, user *models.User) error {
	user.ID = uuid.New().String()
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()
	return r.db.WithContext(ctx).Create(user).Error
}

func (r *MySQLRepository) GetUserByID(ctx context.Context, id string) (*models.User, error) {
	var user models.User
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&user).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("user not found")
		}
		return nil, err
	}
	return &user, nil
}

func (r *MySQLRepository) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	var user models.User
	err := r.db.WithContext(ctx).Where("email = ?", email).First(&user).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("user not found")
		}
		return nil, err
	}
	return &user, nil
}

func (r *MySQLRepository) UpdateUser(ctx context.Context, user *models.User) error {
	user.UpdatedAt = time.Now()
	return r.db.WithContext(ctx).Save(user).Error
}

func (r *MySQLRepository) DeleteUser(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&models.User{}, "id = ?", id).Error
}

// Album operations
func (r *MySQLRepository) CreateAlbum(ctx context.Context, album *models.Album) error {
	album.ID = uuid.New().String()
	album.CreatedAt = time.Now()
	album.UpdatedAt = time.Now()
	return r.db.WithContext(ctx).Create(album).Error
}

func (r *MySQLRepository) GetAlbumByID(ctx context.Context, id string) (*models.Album, error) {
	var album models.Album
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&album).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("album not found")
		}
		return nil, err
	}
	return &album, nil
}

func (r *MySQLRepository) GetAlbums(ctx context.Context, userID string, pagination *models.PaginationQuery) ([]*models.Album, int64, error) {
	var albums []*models.Album
	var total int64

	query := r.db.WithContext(ctx).Model(&models.Album{})
	
	if userID != "" {
		query = query.Where("user_id = ?", userID)
	}

	// Get total count
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Apply pagination
	offset := (pagination.Page - 1) * pagination.PageSize
	orderClause := fmt.Sprintf("%s %s", pagination.SortBy, pagination.Order)
	
	err := query.
		Order(orderClause).
		Limit(pagination.PageSize).
		Offset(offset).
		Find(&albums).Error

	return albums, total, err
}

func (r *MySQLRepository) UpdateAlbum(ctx context.Context, album *models.Album) error {
	album.UpdatedAt = time.Now()
	return r.db.WithContext(ctx).Save(album).Error
}

func (r *MySQLRepository) DeleteAlbum(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&models.Album{}, "id = ?", id).Error
}

// Utility
func (r *MySQLRepository) Close() error {
	sqlDB, err := r.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

func (r *MySQLRepository) HealthCheck(ctx context.Context) error {
	sqlDB, err := r.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.PingContext(ctx)
}
