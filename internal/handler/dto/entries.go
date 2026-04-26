package dto

import (
	"time"

	db "ranke-be/internal/db/sqlc"
)

// EntryResponse is what /lists/:id/entries/me returns after upsert.
// Same shape as RankedEntry but without a computed `rank` (the caller
// re-fetches the list to get fresh ranks).
type EntryResponse struct {
	ID              string    `json:"id"`
	UserID          string    `json:"userId"`
	ListID          string    `json:"listId"`
	ValueNumber     *float64  `json:"valueNumber,omitempty"`
	ValueDurationMs *int64    `json:"valueDurationMs,omitempty"`
	ValueText       *string   `json:"valueText,omitempty"`
	ManualRank      *int      `json:"manualRank,omitempty"`
	Note            *string   `json:"note,omitempty"`
	Status          string    `json:"status"`
	PreviousRank    *int      `json:"previousRank,omitempty"`
	SubmittedAt     time.Time `json:"submittedAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
}

// MapEntry converts a sqlc Entry → wire DTO.
func MapEntry(e db.Entry) EntryResponse {
	return EntryResponse{
		ID:              FormatUUID(e.ID),
		UserID:          FormatUUID(e.UserID),
		ListID:          FormatUUID(e.ListID),
		ValueNumber:     Float8Ptr(e.ValueNumber),
		ValueDurationMs: Int8Ptr(e.ValueDurationMs),
		ValueText:       TextPtr(e.ValueText),
		ManualRank:      Int4Ptr(e.ManualRank),
		Note:            TextPtr(e.Note),
		Status:          e.Status,
		PreviousRank:    Int4Ptr(e.PreviousRank),
		SubmittedAt:     Timestamptz(e.SubmittedAt),
		UpdatedAt:       Timestamptz(e.UpdatedAt),
	}
}

// MapPendingEntryRow → RankedEntry (for the moderation feed).
// Pending entries don't have a computed rank yet, so Rank stays 0.
func MapPendingEntryRow(r db.GetPendingEntriesRow) RankedEntry {
	return RankedEntry{
		ID:              FormatUUID(r.ID),
		UserID:          FormatUUID(r.UserID),
		DisplayName:     r.DisplayName,
		AvatarURL:       TextPtr(r.AvatarUrl),
		Rank:            0,
		PreviousRank:    Int4Ptr(r.PreviousRank),
		ValueNumber:     Float8Ptr(r.ValueNumber),
		ValueDurationMs: Int8Ptr(r.ValueDurationMs),
		ValueText:       TextPtr(r.ValueText),
		ManualRank:      Int4Ptr(r.ManualRank),
		Note:            TextPtr(r.Note),
		Status:          r.Status,
		SubmittedAt:     Timestamptz(r.SubmittedAt),
	}
}
