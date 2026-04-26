package dto

import (
	"time"

	db "ranke-be/internal/db/sqlc"
)

// ListSummary is the row shape returned by /lists, /lists/public,
// /users/:id/profile.boards. It carries enough to render a board tile
// without fetching the full leaderboard.
type ListSummary struct {
	ID              string  `json:"id"`
	Title           string  `json:"title"`
	Description     *string `json:"description,omitempty"`
	ValueType       string  `json:"valueType"`
	RankOrder       string  `json:"rankOrder"`
	IsPublic        bool    `json:"isPublic"`
	Locked          bool    `json:"locked"`
	Category        *string `json:"category,omitempty"`
	MemberCount     int     `json:"memberCount"`
	OwnRank         *int    `json:"ownRank,omitempty"`
	CurrentUserRole *string `json:"currentUserRole,omitempty"`
}

// RankedList is the full leaderboard payload returned by /lists/:id and
// the create/update/invite endpoints.
type RankedList struct {
	ID              string         `json:"id"`
	Title           string         `json:"title"`
	Description     *string        `json:"description,omitempty"`
	ValueType       string         `json:"valueType"`
	RankOrder       string         `json:"rankOrder"`
	IsPublic        bool           `json:"isPublic"`
	Locked          bool           `json:"locked"`
	Category        *string        `json:"category,omitempty"`
	InviteToken     *string        `json:"inviteToken,omitempty"`
	TelegramLink    *string        `json:"telegramLink,omitempty"`
	WhatsappLink    *string        `json:"whatsappLink,omitempty"`
	DiscordLink     *string        `json:"discordLink,omitempty"`
	MemberCount     int            `json:"memberCount"`
	CurrentUserRole *string        `json:"currentUserRole,omitempty"`
	Entries         []RankedEntry  `json:"entries"`
}

// RankedEntry is one row in a leaderboard.
type RankedEntry struct {
	ID              string    `json:"id"`
	UserID          string    `json:"userId"`
	DisplayName     string    `json:"displayName"`
	AvatarURL       *string   `json:"avatarUrl,omitempty"`
	Rank            int       `json:"rank"`
	PreviousRank    *int      `json:"previousRank,omitempty"`
	ValueNumber     *float64  `json:"valueNumber,omitempty"`
	ValueDurationMs *int64    `json:"valueDurationMs,omitempty"`
	ValueText       *string   `json:"valueText,omitempty"`
	ManualRank      *int      `json:"manualRank,omitempty"`
	Note            *string   `json:"note,omitempty"`
	Status          string    `json:"status"`
	SubmittedAt     time.Time `json:"submittedAt"`
}

// ListMember is the row shape for /lists/:id/members.
type ListMember struct {
	UserID      string  `json:"userId"`
	DisplayName string  `json:"displayName"`
	AvatarURL   *string `json:"avatarUrl,omitempty"`
	Role        string  `json:"role"`
}

// InviteLink is returned by the invite-link endpoints.
type InviteLink struct {
	InviteLink  string `json:"inviteLink"`
	InviteToken string `json:"inviteToken"`
}

// ── mappers ──────────────────────────────────────────────────────────

// MapList maps the bare list row (used after create/update where there
// are no entries / member count yet — caller fills those in).
func MapList(l db.List, memberCount int, role *string, entries []RankedEntry) RankedList {
	inviteToken := FormatUUID(l.InviteToken)
	return RankedList{
		ID:              FormatUUID(l.ID),
		Title:           l.Title,
		Description:     TextPtr(l.Description),
		ValueType:       l.ValueType,
		RankOrder:       l.RankOrder,
		IsPublic:        l.IsPublic,
		Locked:          l.Locked,
		Category:        TextPtr(l.Category),
		InviteToken:     &inviteToken,
		TelegramLink:    TextPtr(l.TelegramLink),
		WhatsappLink:    TextPtr(l.WhatsappLink),
		DiscordLink:     TextPtr(l.DiscordLink),
		MemberCount:     memberCount,
		CurrentUserRole: role,
		Entries:         entries,
	}
}

// MapUserListsRow maps GetUserListsRow → ListSummary, computing ownRank
// from the entry the row carries.
func MapUserListsRow(r db.GetUserListsRow, ownRank *int) ListSummary {
	role := r.UserRole
	return ListSummary{
		ID:              FormatUUID(r.ID),
		Title:           r.Title,
		Description:     TextPtr(r.Description),
		ValueType:       r.ValueType,
		RankOrder:       r.RankOrder,
		IsPublic:        r.IsPublic,
		Locked:          r.Locked,
		Category:        TextPtr(r.Category),
		MemberCount:     int(r.MemberCount),
		OwnRank:         ownRank,
		CurrentUserRole: &role,
	}
}

// MapPublicListsRow maps SearchPublicListsRow → ListSummary.
func MapPublicListsRow(r db.SearchPublicListsRow) ListSummary {
	return ListSummary{
		ID:          FormatUUID(r.ID),
		Title:       r.Title,
		Description: TextPtr(r.Description),
		ValueType:   r.ValueType,
		RankOrder:   r.RankOrder,
		IsPublic:    r.IsPublic,
		Locked:      r.Locked,
		Category:    TextPtr(r.Category),
		MemberCount: int(r.MemberCount),
	}
}

// MapPublicBoardsForUserRow maps GetPublicBoardsForUserRow → ListSummary
// (used by the public profile endpoint).
func MapPublicBoardsForUserRow(r db.GetPublicBoardsForUserRow, ownRank *int) ListSummary {
	role := r.UserRole
	return ListSummary{
		ID:              FormatUUID(r.ID),
		Title:           r.Title,
		Description:     TextPtr(r.Description),
		ValueType:       r.ValueType,
		RankOrder:       r.RankOrder,
		IsPublic:        r.IsPublic,
		Locked:          r.Locked,
		Category:        TextPtr(r.Category),
		MemberCount:     int(r.MemberCount),
		OwnRank:         ownRank,
		CurrentUserRole: &role,
	}
}

// MapMembersRow maps a ListMembersRow → ListMember.
func MapMembersRow(r db.ListMembersRow) ListMember {
	return ListMember{
		UserID:      FormatUUID(r.UserID),
		DisplayName: r.DisplayName,
		AvatarURL:   TextPtr(r.AvatarUrl),
		Role:        r.Role,
	}
}

// MapInvitePreviewRow maps GetInvitePreviewRow → RankedList (no entries —
// preview is read by non-members).
func MapInvitePreviewRow(r db.GetInvitePreviewRow) RankedList {
	inviteToken := FormatUUID(r.InviteToken)
	return RankedList{
		ID:           FormatUUID(r.ID),
		Title:        r.Title,
		Description:  TextPtr(r.Description),
		ValueType:    r.ValueType,
		RankOrder:    r.RankOrder,
		IsPublic:     r.IsPublic,
		Locked:       r.Locked,
		Category:     TextPtr(r.Category),
		InviteToken:  &inviteToken,
		TelegramLink: TextPtr(r.TelegramLink),
		WhatsappLink: TextPtr(r.WhatsappLink),
		DiscordLink:  TextPtr(r.DiscordLink),
		MemberCount:  int(r.MemberCount),
		Entries:      []RankedEntry{},
	}
}
