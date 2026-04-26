package dto

import (
	"time"

	db "ranke-be/internal/db/sqlc"
)

// User is the public user shape for auth + /users/me responses.
//
// `email` is omitted for any user lookup that is not the caller themselves
// (privacy). The auth flow can include it because the caller just signed in.
type User struct {
	ID          string  `json:"id"`
	Email       string  `json:"email,omitempty"`
	DisplayName string  `json:"displayName"`
	AvatarURL   *string `json:"avatarUrl,omitempty"`
	CreatedAt   string  `json:"createdAt"`
}

// AuthResponse is the flat shape returned by /auth/{login,register,apple}.
//
// Mobile expects all four fields at the top level — no nested "tokens"
// object — so AuthRepositoryImpl can persist the tokens and map the user
// in a single pass.
type AuthResponse struct {
	User         User   `json:"user"`
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
	ExpiresIn    int64  `json:"expiresIn"`
}

// RefreshResponse is the flat shape returned by /auth/refresh.
type RefreshResponse struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
	ExpiresIn    int64  `json:"expiresIn"`
}

// MapUser turns a sqlc User row into the wire DTO. Pass `includeEmail=true`
// only for the caller's own user (e.g. /auth/* and /users/me).
func MapUser(u db.User, includeEmail bool) User {
	out := User{
		ID:          FormatUUID(u.ID),
		DisplayName: u.DisplayName,
		AvatarURL:   TextPtr(u.AvatarUrl),
		CreatedAt:   Timestamptz(u.CreatedAt).UTC().Format(time.RFC3339),
	}
	if includeEmail {
		out.Email = u.Email
	}
	return out
}
