// Package dto defines the JSON wire contract between the backend and the
// mobile app. Every field uses camelCase tags to match the Flutter client's
// expectations.
//
// Why a separate package: sqlc-generated row structs use snake_case JSON
// tags and leak DB-only fields (e.g. password_hash). Handlers must always
// translate sqlc rows into the DTOs declared here — never serialize a row
// directly into a response.
package dto

import (
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

// FormatUUID renders a pgtype.UUID as the canonical 8-4-4-4-12 string.
func FormatUUID(u pgtype.UUID) string {
	b := u.Bytes
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// pgtype.Text → *string. Returns nil when the column is NULL.
func TextPtr(t pgtype.Text) *string {
	if !t.Valid {
		return nil
	}
	v := t.String
	return &v
}

// pgtype.Float8 → *float64.
func Float8Ptr(f pgtype.Float8) *float64 {
	if !f.Valid {
		return nil
	}
	v := f.Float64
	return &v
}

// pgtype.Int8 → *int64.
func Int8Ptr(i pgtype.Int8) *int64 {
	if !i.Valid {
		return nil
	}
	v := i.Int64
	return &v
}

// pgtype.Int4 → *int.
func Int4Ptr(i pgtype.Int4) *int {
	if !i.Valid {
		return nil
	}
	v := int(i.Int32)
	return &v
}

// pgtype.Timestamptz → time.Time (zero value on NULL).
func Timestamptz(t pgtype.Timestamptz) time.Time {
	if !t.Valid {
		return time.Time{}
	}
	return t.Time
}

// pgtype.Timestamptz → *time.Time (nil on NULL).
func TimestamptzPtr(t pgtype.Timestamptz) *time.Time {
	if !t.Valid {
		return nil
	}
	v := t.Time
	return &v
}
