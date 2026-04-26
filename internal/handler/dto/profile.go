package dto

import (
	"time"

	db "ranke-be/internal/db/sqlc"
)

// UserProfile is the public profile shape returned by /users/:id/profile.
//
// Mobile maps `memberSince` to `DateTime?` and `boards` to a list of public
// boards the user belongs to. We do **not** expose the email here — public
// profiles must not leak contact information.
type UserProfile struct {
	UserID      string        `json:"userId"`
	DisplayName string        `json:"displayName"`
	AvatarURL   *string       `json:"avatarUrl,omitempty"`
	MemberSince *time.Time    `json:"memberSince,omitempty"`
	Boards      []ListSummary `json:"boards"`
}

// MapUserProfile combines the profile row + a slice of public board rows
// into the wire DTO.
func MapUserProfile(p db.GetUserProfileRow, boards []ListSummary) UserProfile {
	return UserProfile{
		UserID:      FormatUUID(p.ID),
		DisplayName: p.DisplayName,
		AvatarURL:   TextPtr(p.AvatarUrl),
		MemberSince: TimestamptzPtr(p.CreatedAt),
		Boards:      boards,
	}
}
