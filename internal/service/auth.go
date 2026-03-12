package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

	"ranke-be/internal/apple"
	"ranke-be/internal/config"
	db "ranke-be/internal/db/sqlc"
)

type AuthService struct {
	queries       *db.Queries
	pool          *pgxpool.Pool
	cfg           *config.Config
	appleVerifier *apple.Verifier
}

func NewAuthService(queries *db.Queries, pool *pgxpool.Pool, cfg *config.Config, av *apple.Verifier) *AuthService {
	return &AuthService{queries: queries, pool: pool, cfg: cfg, appleVerifier: av}
}

type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
}

func (s *AuthService) Register(ctx context.Context, email, displayName, password string) (*db.User, *TokenPair, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return nil, nil, fmt.Errorf("hash password: %w", err)
	}

	user, err := s.queries.CreateUser(ctx, db.CreateUserParams{
		Email:        email,
		DisplayName:  displayName,
		PasswordHash: pgtype.Text{String: string(hash), Valid: true},
	})
	if err != nil {
		return nil, nil, fmt.Errorf("create user: %w", err)
	}

	pair, err := s.issueTokenPair(ctx, user.ID)
	if err != nil {
		return nil, nil, err
	}

	return &user, pair, nil
}

func (s *AuthService) Login(ctx context.Context, email, password string) (*db.User, *TokenPair, error) {
	user, err := s.queries.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, nil, errors.New("invalid credentials")
	}

	if !user.PasswordHash.Valid {
		return nil, nil, errors.New("invalid credentials")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash.String), []byte(password)); err != nil {
		return nil, nil, errors.New("invalid credentials")
	}

	pair, err := s.issueTokenPair(ctx, user.ID)
	if err != nil {
		return nil, nil, err
	}

	return &user, pair, nil
}

func (s *AuthService) AppleSignIn(ctx context.Context, identityToken string) (*db.User, *TokenPair, error) {
	if s.appleVerifier == nil {
		return nil, nil, errors.New("apple sign-in not configured")
	}

	claims, err := s.appleVerifier.Verify(ctx, identityToken)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid apple token: %w", err)
	}

	// Try to find existing user by apple_sub
	user, err := s.queries.GetUserByAppleSub(ctx, pgtype.Text{String: claims.Subject, Valid: true})
	if err == nil {
		// Existing user — issue tokens
		pair, err := s.issueTokenPair(ctx, user.ID)
		if err != nil {
			return nil, nil, err
		}
		return &user, pair, nil
	}

	// New user — create
	email := claims.Email
	if email == "" {
		email = claims.Subject + "@privaterelay.appleid.com"
	}
	displayName := email
	if idx := len(email); idx > 0 {
		for i, c := range email {
			if c == '@' {
				displayName = email[:i]
				break
			}
		}
	}

	user, err = s.queries.CreateAppleUser(ctx, db.CreateAppleUserParams{
		Email:       email,
		DisplayName: displayName,
		AppleSub:    pgtype.Text{String: claims.Subject, Valid: true},
	})
	if err != nil {
		return nil, nil, fmt.Errorf("create apple user: %w", err)
	}

	pair, err := s.issueTokenPair(ctx, user.ID)
	if err != nil {
		return nil, nil, err
	}
	return &user, pair, nil
}

func (s *AuthService) RefreshToken(ctx context.Context, refreshToken string) (*TokenPair, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	qtx := s.queries.WithTx(tx)

	rt, err := qtx.GetValidRefreshToken(ctx, refreshToken)
	if err != nil {
		return nil, errors.New("invalid refresh token")
	}

	// Revoke old token
	if err := qtx.RevokeRefreshToken(ctx, rt.ID); err != nil {
		return nil, fmt.Errorf("revoke old token: %w", err)
	}

	// Issue new pair
	accessToken, err := s.generateAccessToken(rt.UserID)
	if err != nil {
		return nil, err
	}

	newRT, err := s.generateAndStoreRefreshToken(ctx, qtx, rt.UserID)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: newRT,
		ExpiresIn:    int64(s.cfg.JWTAccessTTL.Seconds()),
	}, nil
}

func (s *AuthService) Logout(ctx context.Context, refreshToken string) error {
	rt, err := s.queries.GetValidRefreshToken(ctx, refreshToken)
	if err != nil {
		return nil // Idempotent — already revoked or doesn't exist
	}
	return s.queries.RevokeRefreshToken(ctx, rt.ID)
}

// --- internal helpers ---

func (s *AuthService) issueTokenPair(ctx context.Context, userID pgtype.UUID) (*TokenPair, error) {
	accessToken, err := s.generateAccessToken(userID)
	if err != nil {
		return nil, err
	}

	refreshToken, err := s.generateAndStoreRefreshToken(ctx, s.queries, userID)
	if err != nil {
		return nil, err
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(s.cfg.JWTAccessTTL.Seconds()),
	}, nil
}

func (s *AuthService) generateAccessToken(userID pgtype.UUID) (string, error) {
	uidBytes := userID.Bytes
	uidStr := fmt.Sprintf("%x-%x-%x-%x-%x", uidBytes[0:4], uidBytes[4:6], uidBytes[6:8], uidBytes[8:10], uidBytes[10:16])

	claims := jwt.RegisteredClaims{
		Subject:   uidStr,
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.cfg.JWTAccessTTL)),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.cfg.JWTSecret))
}

func (s *AuthService) generateAndStoreRefreshToken(ctx context.Context, q *db.Queries, userID pgtype.UUID) (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("generate random token: %w", err)
	}
	token := hex.EncodeToString(raw)

	_, err := q.CreateRefreshToken(ctx, db.CreateRefreshTokenParams{
		UserID:    userID,
		Token:     token,
		ExpiresAt: pgtype.Timestamptz{Time: time.Now().Add(s.cfg.JWTRefreshTTL), Valid: true},
	})
	if err != nil {
		return "", fmt.Errorf("store refresh token: %w", err)
	}

	return token, nil
}

// TxQuerier wraps a pgx.Tx for use with sqlc queries.
var _ pgx.Tx = (pgx.Tx)(nil) // compile-time check that pgx.Tx is available
