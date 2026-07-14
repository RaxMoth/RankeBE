package service

import "testing"

// hashRefreshToken is a pure function guarding two invariants the storage
// layer depends on: it's deterministic (so a presented token re-derives the
// stored digest) and its output is exactly 64 hex chars (so it fits the
// existing `token TEXT UNIQUE` column that previously held raw-hex tokens).
func TestHashRefreshToken(t *testing.T) {
	const raw = "deadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef"

	got := hashRefreshToken(raw)

	if len(got) != 64 {
		t.Fatalf("digest width = %d, want 64", len(got))
	}
	if again := hashRefreshToken(raw); again != got {
		t.Fatalf("not deterministic: %q != %q", again, got)
	}
	if got == raw {
		t.Fatal("digest equals input — token stored in the clear")
	}
	if other := hashRefreshToken(raw + "x"); other == got {
		t.Fatal("distinct inputs collided")
	}
}
