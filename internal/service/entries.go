package service

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgtype"

	db "ranke-be/internal/db/sqlc"
)

type EntryService struct {
	queries *db.Queries
}

func NewEntryService(queries *db.Queries) *EntryService {
	return &EntryService{queries: queries}
}

type UpsertEntryInput struct {
	ValueNumber     *float64
	ValueDurationMs *int64
	ValueText       *string
	Note            *string
}

func (s *EntryService) UpsertEntry(ctx context.Context, listID, userID pgtype.UUID, valueType string, input UpsertEntryInput) (*db.Entry, error) {
	// Value type validation per spec
	switch valueType {
	case "number":
		if input.ValueDurationMs != nil || input.ValueText != nil {
			return nil, errors.New("INVALID_VALUE_TYPE")
		}
	case "duration":
		if input.ValueNumber != nil || input.ValueText != nil {
			return nil, errors.New("INVALID_VALUE_TYPE")
		}
	case "text":
		if input.ValueNumber != nil || input.ValueDurationMs != nil {
			return nil, errors.New("INVALID_VALUE_TYPE")
		}
	}

	params := db.UpsertEntryParams{
		ListID: listID,
		UserID: userID,
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

	entry, err := s.queries.UpsertEntry(ctx, params)
	if err != nil {
		return nil, err
	}
	return &entry, nil
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
