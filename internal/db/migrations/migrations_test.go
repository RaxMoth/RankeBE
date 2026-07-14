package migrations

import (
	"sort"
	"strings"
	"testing"
)

// TestEmbeddedMigrationsPresent guards against two silent failure modes:
// the //go:embed pattern capturing zero files, and a migration filename that
// won't sort into the intended apply order.
func TestEmbeddedMigrationsPresent(t *testing.T) {
	files, err := sqlFiles()
	if err != nil {
		t.Fatalf("sqlFiles: %v", err)
	}
	if len(files) == 0 {
		t.Fatal("no *.sql migrations embedded — check the //go:embed directive")
	}

	if !sort.StringsAreSorted(files) {
		t.Errorf("sqlFiles not in lexical order: %v", files)
	}

	for _, name := range files {
		if !strings.HasSuffix(name, ".sql") {
			t.Errorf("non-sql file returned by sqlFiles: %q", name)
		}
		body, err := FS.ReadFile(name)
		if err != nil {
			t.Errorf("ReadFile(%q): %v", name, err)
			continue
		}
		if len(body) == 0 {
			t.Errorf("migration %q is empty", name)
		}
	}
}
