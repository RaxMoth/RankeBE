package repository

import (
	"context"
	"fmt"
	"time"

	"gin-rest-template/internal/config"
	"gin-rest-template/internal/models"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go"
	"github.com/google/uuid"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

type FirebaseRepository struct {
	client *firestore.Client
}

func NewFirebaseRepository(cfg *config.Config) (*FirebaseRepository, error) {
	ctx := context.Background()

	opt := option.WithCredentialsFile(cfg.FirebaseCredentials)
	app, err := firebase.NewApp(ctx, &firebase.Config{
		ProjectID: cfg.FirebaseProjectID,
	}, opt)
	if err != nil {
		return nil, fmt.Errorf("error initializing firebase app: %w", err)
	}

	client, err := app.Firestore(ctx)
	if err != nil {
		return nil, fmt.Errorf("error initializing firestore client: %w", err)
	}

	return &FirebaseRepository{client: client}, nil
}

// User operations
func (r *FirebaseRepository) CreateUser(ctx context.Context, user *models.User) error {
	user.ID = uuid.New().String()
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()

	_, err := r.client.Collection("users").Doc(user.ID).Set(ctx, user)
	return err
}

func (r *FirebaseRepository) GetUserByID(ctx context.Context, id string) (*models.User, error) {
	doc, err := r.client.Collection("users").Doc(id).Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	var user models.User
	if err := doc.DataTo(&user); err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *FirebaseRepository) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	iter := r.client.Collection("users").Where("email", "==", email).Limit(1).Documents(ctx)
	doc, err := iter.Next()
	if err == iterator.Done {
		return nil, fmt.Errorf("user not found")
	}
	if err != nil {
		return nil, err
	}

	var user models.User
	if err := doc.DataTo(&user); err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *FirebaseRepository) UpdateUser(ctx context.Context, user *models.User) error {
	user.UpdatedAt = time.Now()
	_, err := r.client.Collection("users").Doc(user.ID).Set(ctx, user)
	return err
}

func (r *FirebaseRepository) DeleteUser(ctx context.Context, id string) error {
	_, err := r.client.Collection("users").Doc(id).Delete(ctx)
	return err
}

// Album operations
func (r *FirebaseRepository) CreateAlbum(ctx context.Context, album *models.Album) error {
	album.ID = uuid.New().String()
	album.CreatedAt = time.Now()
	album.UpdatedAt = time.Now()

	_, err := r.client.Collection("albums").Doc(album.ID).Set(ctx, album)
	return err
}

func (r *FirebaseRepository) GetAlbumByID(ctx context.Context, id string) (*models.Album, error) {
	doc, err := r.client.Collection("albums").Doc(id).Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("album not found: %w", err)
	}

	var album models.Album
	if err := doc.DataTo(&album); err != nil {
		return nil, err
	}

	return &album, nil
}

func (r *FirebaseRepository) GetAlbums(ctx context.Context, userID string, pagination *models.PaginationQuery) ([]*models.Album, int64, error) {
	query := r.client.Collection("albums").Query

	if userID != "" {
		query = query.Where("user_id", "==", userID)
	}

	// Get total count
	countSnap, err := query.Documents(ctx).GetAll()
	if err != nil {
		return nil, 0, err
	}
	total := int64(len(countSnap))

	// Apply sorting
	var direction firestore.Direction
	if pagination.Order == "desc" {
		direction = firestore.Desc
	} else {
		direction = firestore.Asc
	}
	query = query.OrderBy(pagination.SortBy, direction)

	// Apply pagination
	offset := (pagination.Page - 1) * pagination.PageSize
	query = query.Offset(offset).Limit(pagination.PageSize)

	iter := query.Documents(ctx)
	docs, err := iter.GetAll()
	if err != nil {
		return nil, 0, err
	}

	albums := make([]*models.Album, 0, len(docs))
	for _, doc := range docs {
		var album models.Album
		if err := doc.DataTo(&album); err != nil {
			return nil, 0, err
		}
		albums = append(albums, &album)
	}

	return albums, total, nil
}

func (r *FirebaseRepository) UpdateAlbum(ctx context.Context, album *models.Album) error {
	album.UpdatedAt = time.Now()
	_, err := r.client.Collection("albums").Doc(album.ID).Set(ctx, album)
	return err
}

func (r *FirebaseRepository) DeleteAlbum(ctx context.Context, id string) error {
	_, err := r.client.Collection("albums").Doc(id).Delete(ctx)
	return err
}

// Utility
func (r *FirebaseRepository) Close() error {
	return r.client.Close()
}

func (r *FirebaseRepository) HealthCheck(ctx context.Context) error {
	// Try to read from a collection to verify connection
	_, err := r.client.Collection("health_check").Limit(1).Documents(ctx).Next()
	if err != nil && err != iterator.Done {
		return err
	}
	return nil
}
