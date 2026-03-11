package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gin-rest-template/internal/config"
	"gin-rest-template/internal/models"
	"gin-rest-template/internal/repository"
	"gin-rest-template/pkg/logger"

	"github.com/golang-jwt/jwt/v5"
)

type Service struct {
	repo   repository.Repository
	config *config.Config
}

func NewService(repo repository.Repository, config *config.Config) *Service {
	return &Service{
		repo:   repo,
		config: config,
	}
}

// JWT Claims
type Claims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
}

// Auth operations
func (s *Service) Register(ctx context.Context, req *models.RegisterRequest) (*models.UserResponse, error) {
	// Check if user already exists
	existingUser, err := s.repo.GetUserByEmail(ctx, req.Email)
	if err == nil && existingUser != nil {
		return nil, errors.New("user with this email already exists")
	}

	// Create user
	user := &models.User{
		Email: req.Email,
		Name:  req.Name,
		Password: req.Password,
	}

	if err := user.HashPassword(); err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	if err := s.repo.CreateUser(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	logger.Info("User registered successfully", "user_id", user.ID, "email", user.Email)
	return user.ToResponse(), nil
}

func (s *Service) Login(ctx context.Context, req *models.LoginRequest) (*models.TokenResponse, error) {
	user, err := s.repo.GetUserByEmail(ctx, req.Email)
	if err != nil {
		return nil, errors.New("invalid credentials")
	}

	if !user.CheckPassword(req.Password) {
		return nil, errors.New("invalid credentials")
	}

	accessToken, err := s.generateAccessToken(user)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshToken, err := s.generateRefreshToken(user)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	logger.Info("User logged in successfully", "user_id", user.ID, "email", user.Email)

	return &models.TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    int64(s.config.JWTExpiration.Seconds()),
	}, nil
}

func (s *Service) RefreshToken(ctx context.Context, refreshToken string) (*models.TokenResponse, error) {
	claims, err := s.ValidateToken(refreshToken, true)
	if err != nil {
		return nil, errors.New("invalid refresh token")
	}

	user, err := s.repo.GetUserByID(ctx, claims.UserID)
	if err != nil {
		return nil, errors.New("user not found")
	}

	accessToken, err := s.generateAccessToken(user)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	newRefreshToken, err := s.generateRefreshToken(user)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	return &models.TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    int64(s.config.JWTExpiration.Seconds()),
	}, nil
}

func (s *Service) generateAccessToken(user *models.User) (string, error) {
	claims := &Claims{
		UserID: user.ID,
		Email:  user.Email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.config.JWTExpiration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.config.JWTSecret))
}

func (s *Service) generateRefreshToken(user *models.User) (string, error) {
	claims := &Claims{
		UserID: user.ID,
		Email:  user.Email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.config.JWTRefreshExpiration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.config.JWTSecret))
}

func (s *Service) ValidateToken(tokenString string, isRefreshToken bool) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.config.JWTSecret), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}

// User operations
func (s *Service) GetUserByID(ctx context.Context, userID string) (*models.UserResponse, error) {
	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return user.ToResponse(), nil
}

func (s *Service) UpdateUser(ctx context.Context, userID string, req *models.UpdateUserRequest) (*models.UserResponse, error) {
	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	if req.Name != "" {
		user.Name = req.Name
	}
	if req.Email != "" {
		// Check if email is already taken
		existingUser, _ := s.repo.GetUserByEmail(ctx, req.Email)
		if existingUser != nil && existingUser.ID != userID {
			return nil, errors.New("email already taken")
		}
		user.Email = req.Email
	}

	if err := s.repo.UpdateUser(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	logger.Info("User updated successfully", "user_id", userID)
	return user.ToResponse(), nil
}

// Album operations
func (s *Service) CreateAlbum(ctx context.Context, userID string, album *models.Album) (*models.Album, error) {
	album.UserID = userID
	if err := s.repo.CreateAlbum(ctx, album); err != nil {
		return nil, fmt.Errorf("failed to create album: %w", err)
	}

	logger.Info("Album created successfully", "album_id", album.ID, "user_id", userID)
	return album, nil
}

func (s *Service) GetAlbumByID(ctx context.Context, albumID, userID string) (*models.Album, error) {
	album, err := s.repo.GetAlbumByID(ctx, albumID)
	if err != nil {
		return nil, err
	}

	// Check ownership
	if album.UserID != userID {
		return nil, errors.New("unauthorized access to album")
	}

	return album, nil
}

func (s *Service) GetAlbums(ctx context.Context, userID string, pagination *models.PaginationQuery) (*models.PaginatedResponse, error) {
	albums, total, err := s.repo.GetAlbums(ctx, userID, pagination)
	if err != nil {
		return nil, fmt.Errorf("failed to get albums: %w", err)
	}

	totalPages := int(total) / pagination.PageSize
	if int(total)%pagination.PageSize > 0 {
		totalPages++
	}

	return &models.PaginatedResponse{
		Data:       albums,
		Page:       pagination.Page,
		PageSize:   pagination.PageSize,
		TotalItems: total,
		TotalPages: totalPages,
	}, nil
}

func (s *Service) UpdateAlbum(ctx context.Context, albumID, userID string, updates *models.Album) (*models.Album, error) {
	album, err := s.repo.GetAlbumByID(ctx, albumID)
	if err != nil {
		return nil, err
	}

	// Check ownership
	if album.UserID != userID {
		return nil, errors.New("unauthorized access to album")
	}

	// Update fields
	if updates.Title != "" {
		album.Title = updates.Title
	}
	if updates.Artist != "" {
		album.Artist = updates.Artist
	}
	if updates.Price > 0 {
		album.Price = updates.Price
	}

	if err := s.repo.UpdateAlbum(ctx, album); err != nil {
		return nil, fmt.Errorf("failed to update album: %w", err)
	}

	logger.Info("Album updated successfully", "album_id", albumID, "user_id", userID)
	return album, nil
}

func (s *Service) DeleteAlbum(ctx context.Context, albumID, userID string) error {
	album, err := s.repo.GetAlbumByID(ctx, albumID)
	if err != nil {
		return err
	}

	// Check ownership
	if album.UserID != userID {
		return errors.New("unauthorized access to album")
	}

	if err := s.repo.DeleteAlbum(ctx, albumID); err != nil {
		return fmt.Errorf("failed to delete album: %w", err)
	}

	logger.Info("Album deleted successfully", "album_id", albumID, "user_id", userID)
	return nil
}

// Health check
func (s *Service) HealthCheck(ctx context.Context) error {
	return s.repo.HealthCheck(ctx)
}
