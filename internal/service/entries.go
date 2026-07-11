package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	db "ranke-be/internal/db/sqlc"
)

// Sentinel errors returned by UpsertEntry. Handlers match these with
// errors.Is rather than comparing err.Error() strings, so the mapping to
// an HTTP status/code can't silently drift when a message is reworded.
var (
	// ErrInvalidValueType — submitted value fields don't match the list's
	// value_type (e.g. a text value on a number list).
	ErrInvalidValueType = errors.New("value does not match list type")
	// ErrListLocked — a write was attempted on a frozen (locked) list.
	ErrListLocked = errors.New("list is locked")
)

type EntryService struct {
	queries *db.Queries
	pool    *pgxpool.Pool
	lists   *ListService
}

func NewEntryService(queries *db.Queries, pool *pgxpool.Pool, lists *ListService) *EntryService {
	return &EntryService{queries: queries, pool: pool, lists: lists}
}

type UpsertEntryInput struct {
	ValueNumber     *float64
	ValueDurationMs *int64
	ValueText       *string
	Note            *string
}

// UpsertEntry submits / updates the caller's entry on a list.
//
// Two pieces of bookkeeping happen on every write:
//
//  1. **status** — set to `pending` when the list is locked (moderation
//     turned on by the owner) so an admin must approve before the entry
//     joins the leaderboard. Otherwise `approved`, joining immediately.
//
//  2. **previous_rank** — we read the user's current rank inside the same
//     transaction (BEFORE the upsert) and store it on the row. This lets
//     the leaderboard render rank-change deltas without a separate history
//     table. First-ever submissions store NULL (no previous rank).
func (s *EntryService) UpsertEntry(ctx context.Context, listID, userID pgtype.UUID, list *db.List, input UpsertEntryInput) (*db.Entry, error) {
	// Value-type validation per spec
	switch list.ValueType {
	case "number":
		if input.ValueDurationMs != nil || input.ValueText != nil {
			return nil, ErrInvalidValueType
		}
	case "duration":
		if input.ValueNumber != nil || input.ValueText != nil {
			return nil, ErrInvalidValueType
		}
	case "text":
		if input.ValueNumber != nil || input.ValueDurationMs != nil {
			return nil, ErrInvalidValueType
		}
	}

	// Refuse writes once the owner has frozen the list. Note: a `locked`
	// list still allows reads — the freeze only blocks new submissions.
	if list.Locked {
		return nil, ErrListLocked
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	qtx := s.queries.WithTx(tx)

	// Capture the user's current rank (if any) so we can stash it in
	// previous_rank on the new row. Use the same-tx queries view so the
	// computation is consistent with the upsert that follows.
	prevRank, _ := s.lookupCurrentRank(ctx, qtx, listID, userID, list.ValueType, list.RankOrder)

	// Default to "approved" — moderation only kicks in if a future flag
	// (e.g. `lists.requires_moderation`) gets added. For now we still wire
	// the `status` column end-to-end so mobile can read it.
	status := "approved"

	params := db.UpsertEntryParams{
		ListID: listID,
		UserID: userID,
		Status: status,
	}
	if input.ValueNumber != nil {
		params.ValueNumber = pgtype.Float8{Float64: *input.ValueNumber, Valid: true}
	}
	if input.ValueDurationMs != nil {
		params.ValueDurationMs = pgtype.Int8{Int64: *input.ValueDurationMs, Valid: true}
	}
	if input.ValueText != nil {
		params.ValueText = pgtype.Text{String: *input.ValueText, Valid: true}
	}
	if input.Note != nil {
		params.Note = pgtype.Text{String: *input.Note, Valid: true}
	}
	if prevRank != nil {
		params.PreviousRank = pgtype.Int4{Int32: int32(*prevRank), Valid: true}
	}

	entry, err := qtx.UpsertEntry(ctx, params)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}
	return &entry, nil
}

// lookupCurrentRank dispatches GetCurrentRankBy* on the same querier `q`.
// Returns nil (no error) when the user has no entry yet.
func (s *EntryService) lookupCurrentRank(ctx context.Context, q *db.Queries, listID, userID pgtype.UUID, valueType, rankOrder string) (*int, error) {
	var rank int32
	var err error

	switch valueType {
	case "number":
		if rankOrder == "desc" {
			rank, err = q.GetCurrentRankByNumberDesc(ctx, db.GetCurrentRankByNumberDescParams{ListID: listID, UserID: userID})
		} else {
			rank, err = q.GetCurrentRankByNumber(ctx, db.GetCurrentRankByNumberParams{ListID: listID, UserID: userID})
		}
	case "duration":
		if rankOrder == "desc" {
			rank, err = q.GetCurrentRankByDurationDesc(ctx, db.GetCurrentRankByDurationDescParams{ListID: listID, UserID: userID})
		} else {
			rank, err = q.GetCurrentRankByDuration(ctx, db.GetCurrentRankByDurationParams{ListID: listID, UserID: userID})
		}
	case "text":
		rank, err = q.GetCurrentRankByText(ctx, db.GetCurrentRankByTextParams{ListID: listID, UserID: userID})
	}

	if err != nil {
		return nil, nil // pgx.ErrNoRows is the common case — user has no entry yet
	}
	v := int(rank)
	return &v, nil
}

func (s *EntryService) DeleteOwnEntry(ctx context.Context, listID, userID pgtype.UUID) error {
	return s.queries.DeleteEntryByListAndUser(ctx, db.DeleteEntryByListAndUserParams{
		ListID: listID,
		UserID: userID,
	})
}

func (s *EntryService) DeleteEntry(ctx context.Context, entryID pgtype.UUID) error {
	return s.queries.DeleteEntry(ctx, entryID)
}

// ── moderation ───────────────────────────────────────────────────────

func (s *EntryService) GetPendingEntries(ctx context.Context, listID pgtype.UUID) ([]db.GetPendingEntriesRow, error) {
	return s.queries.GetPendingEntries(ctx, listID)
}

func (s *EntryService) ApproveEntry(ctx context.Context, listID, entryID pgtype.UUID) error {
	return s.queries.ApproveEntry(ctx, db.ApproveEntryParams{ID: entryID, ListID: listID})
}

func (s *EntryService) RejectEntry(ctx context.Context, listID, entryID pgtype.UUID) error {
	return s.queries.RejectEntry(ctx, db.RejectEntryParams{ID: entryID, ListID: listID})
}
