package handler

import (
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"ranke-be/internal/middleware"
)

func TestParsePublicListsLimit(t *testing.T) {
	cases := []struct {
		raw  string
		want int
	}{
		{"", publicListsDefaultLimit},
		{"garbage", publicListsDefaultLimit},
		{"0", publicListsDefaultLimit},
		{"-5", publicListsDefaultLimit},
		{"1", 1},
		{"30", 30},
		{"100", 100},
		{"101", publicListsMaxLimit},
		{"100000", publicListsMaxLimit},
	}
	for _, tc := range cases {
		if got := parsePublicListsLimit(tc.raw); got != tc.want {
			t.Errorf("parsePublicListsLimit(%q) = %d, want %d", tc.raw, got, tc.want)
		}
	}
}

func TestPublicListCursorRoundTrip(t *testing.T) {
	id, ok := middleware.ParseUUID("11111111-2222-3333-4444-555555555555")
	if !ok {
		t.Fatal("failed to parse fixture uuid")
	}
	// Truncate to microseconds — that's the resolution the cursor preserves.
	updatedAt := pgtype.Timestamptz{Time: time.Date(2026, 7, 14, 10, 30, 0, 123456000, time.UTC), Valid: true}

	token := encodePublicListCursor(updatedAt, id)
	got, valid := decodePublicListCursor(token)
	if !valid {
		t.Fatalf("decodePublicListCursor(%q) reported invalid", token)
	}
	if !got.UpdatedAt.Equal(updatedAt.Time) {
		t.Errorf("updatedAt round-trip = %v, want %v", got.UpdatedAt, updatedAt.Time)
	}
	if got.ID != id {
		t.Errorf("id round-trip = %v, want %v", got.ID, id)
	}
}

func TestDecodePublicListCursor(t *testing.T) {
	if cur, ok := decodePublicListCursor(""); !ok || cur != nil {
		t.Errorf("empty cursor = (%v, %v), want (nil, true)", cur, ok)
	}
	for _, raw := range []string{
		"not-base64!!",        // invalid base64url
		"bm9jb2xvbg",          // valid base64 but no ":" separator ("nocolon")
		"MTIzOm5vdC1hLXV1aWQ", // "123:not-a-uuid"
		"YWJjOjExMTExMTExLTIyMjItMzMzMy00NDQ0LTU1NTU1NTU1NTU1NQ", // "abc:<uuid>" — bad micros
	} {
		if cur, ok := decodePublicListCursor(raw); ok || cur != nil {
			t.Errorf("decodePublicListCursor(%q) = (%v, %v), want (nil, false)", raw, cur, ok)
		}
	}
}
