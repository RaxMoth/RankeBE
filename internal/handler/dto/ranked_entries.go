package dto

import (
	db "ranke-be/internal/db/sqlc"
)

// MapRankedEntries converts whatever sqlc returns (one of five value-type
// row variants) into the unified RankedEntry slice. The rank value comes
// from the ROW_NUMBER() column.
//
// We accept `any` because each ranking query returns its own row type —
// MapRankedEntries dispatches on the concrete slice type and returns an
// empty slice for unknown variants (the caller has already validated the
// list's value_type / rank_order, so this should never happen at runtime).
func MapRankedEntries(rows any) []RankedEntry {
	switch v := rows.(type) {
	case []db.GetRankedEntriesByNumberRow:
		out := make([]RankedEntry, len(v))
		for i, r := range v {
			out[i] = RankedEntry{
				ID:              FormatUUID(r.ID),
				UserID:          FormatUUID(r.UserID),
				DisplayName:     r.DisplayName,
				AvatarURL:       TextPtr(r.AvatarUrl),
				Rank:            int(r.Rank),
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
		return out
	case []db.GetRankedEntriesByNumberDescRow:
		out := make([]RankedEntry, len(v))
		for i, r := range v {
			out[i] = RankedEntry{
				ID:              FormatUUID(r.ID),
				UserID:          FormatUUID(r.UserID),
				DisplayName:     r.DisplayName,
				AvatarURL:       TextPtr(r.AvatarUrl),
				Rank:            int(r.Rank),
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
		return out
	case []db.GetRankedEntriesByDurationRow:
		out := make([]RankedEntry, len(v))
		for i, r := range v {
			out[i] = RankedEntry{
				ID:              FormatUUID(r.ID),
				UserID:          FormatUUID(r.UserID),
				DisplayName:     r.DisplayName,
				AvatarURL:       TextPtr(r.AvatarUrl),
				Rank:            int(r.Rank),
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
		return out
	case []db.GetRankedEntriesByDurationDescRow:
		out := make([]RankedEntry, len(v))
		for i, r := range v {
			out[i] = RankedEntry{
				ID:              FormatUUID(r.ID),
				UserID:          FormatUUID(r.UserID),
				DisplayName:     r.DisplayName,
				AvatarURL:       TextPtr(r.AvatarUrl),
				Rank:            int(r.Rank),
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
		return out
	case []db.GetRankedEntriesByTextRow:
		out := make([]RankedEntry, len(v))
		for i, r := range v {
			out[i] = RankedEntry{
				ID:              FormatUUID(r.ID),
				UserID:          FormatUUID(r.UserID),
				DisplayName:     r.DisplayName,
				AvatarURL:       TextPtr(r.AvatarUrl),
				Rank:            int(r.Rank),
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
		return out
	default:
		return []RankedEntry{}
	}
}
