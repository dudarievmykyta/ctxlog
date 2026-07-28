package memory

import (
	"os"
	"path/filepath"
	"testing"
)

func testStore(t *testing.T) *Store {
	t.Helper()
	dir := t.TempDir()
	s := NewStore(dir)
	if err := s.Init(); err != nil {
		t.Fatal(err)
	}
	return s
}

func TestAppendAndReadAll(t *testing.T) {
	s := testStore(t)
	s.Append("s1", Entry{Msg: "one"})
	s.Append("s1", Entry{Msg: "two"})

	entries, err := s.ReadAll("s1")
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("got %d entries, want 2", len(entries))
	}
	if entries[0].Msg != "one" || entries[1].Msg != "two" {
		t.Fatalf("unexpected messages: %q, %q", entries[0].Msg, entries[1].Msg)
	}
	if entries[0].Ts == 0 {
		t.Fatal("expected ts to be set")
	}
}

func TestReadAllEmpty(t *testing.T) {
	s := testStore(t)
	entries, err := s.ReadAll("nonexistent")
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("got %d entries, want 0", len(entries))
	}
}

func TestReadRecent(t *testing.T) {
	s := testStore(t)
	for i := range 5 {
		s.Append("s1", Entry{Msg: string(rune('a' + i))})
	}

	matches, err := s.ReadRecent("s1", 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 3 {
		t.Fatalf("got %d entries, want 3", len(matches))
	}
	if matches[0].Entry.Msg != "c" {
		t.Fatalf("got %q, want 'c'", matches[0].Entry.Msg)
	}
	if matches[0].Line != 3 || matches[2].Line != 5 {
		t.Fatalf("got lines %d..%d, want 3..5", matches[0].Line, matches[2].Line)
	}
}

func TestSearch(t *testing.T) {
	s := testStore(t)
	s.Append("s1", Entry{Msg: "fixed JWT middleware"})
	s.Append("s1", Entry{Msg: "updated docs"})
	s.Append("s1", Entry{Msg: "jwt validated in staging"})

	matches, err := s.Search("s1", "JWT")
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 2 {
		t.Fatalf("got %d matches, want 2", len(matches))
	}
	if matches[0].Line != 1 || matches[1].Line != 3 {
		t.Fatalf("got lines %d,%d, want 1,3", matches[0].Line, matches[1].Line)
	}
}

func TestSearchNonexistentShard(t *testing.T) {
	s := testStore(t)
	matches, err := s.Search("nope", "x")
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 0 {
		t.Fatalf("got %d matches, want 0", len(matches))
	}
}

func TestSearchLineUsableByDelete(t *testing.T) {
	s := testStore(t)
	s.Append("s1", Entry{Msg: "keep one"})
	s.Append("s1", Entry{Msg: "drop me"})
	s.Append("s1", Entry{Msg: "keep two"})

	matches, err := s.Search("s1", "drop")
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 {
		t.Fatalf("got %d matches, want 1", len(matches))
	}
	if err := s.Delete("s1", matches[0].Line); err != nil {
		t.Fatal(err)
	}
	entries, _ := s.ReadAll("s1")
	if len(entries) != 2 || entries[0].Msg != "keep one" || entries[1].Msg != "keep two" {
		t.Fatalf("unexpected entries after delete: %v", entries)
	}
}

func TestUpdate(t *testing.T) {
	s := testStore(t)
	s.Append("s1", Entry{Msg: "original", Agent: "a1"})

	if err := s.Update("s1", 1, "modified"); err != nil {
		t.Fatal(err)
	}

	entries, _ := s.ReadAll("s1")
	if entries[0].Msg != "modified" {
		t.Fatalf("got %q, want 'modified'", entries[0].Msg)
	}
	if entries[0].Agent != "a1" {
		t.Fatalf("agent changed to %q", entries[0].Agent)
	}
}

func TestUpdateOutOfRange(t *testing.T) {
	s := testStore(t)
	s.Append("s1", Entry{Msg: "one"})

	if err := s.Update("s1", 5, "x"); err == nil {
		t.Fatal("expected error for out of range")
	}
}

func TestDelete(t *testing.T) {
	s := testStore(t)
	s.Append("s1", Entry{Msg: "a"})
	s.Append("s1", Entry{Msg: "b"})
	s.Append("s1", Entry{Msg: "c"})

	if err := s.Delete("s1", 2); err != nil {
		t.Fatal(err)
	}

	entries, _ := s.ReadAll("s1")
	if len(entries) != 2 {
		t.Fatalf("got %d entries, want 2", len(entries))
	}
	if entries[0].Msg != "a" || entries[1].Msg != "c" {
		t.Fatalf("unexpected: %q, %q", entries[0].Msg, entries[1].Msg)
	}
}

func TestDeleteOutOfRange(t *testing.T) {
	s := testStore(t)
	s.Append("s1", Entry{Msg: "one"})

	if err := s.Delete("s1", 0); err == nil {
		t.Fatal("expected error for line 0")
	}
	if err := s.Delete("s1", 3); err == nil {
		t.Fatal("expected error for out of range")
	}
}

func TestClear(t *testing.T) {
	s := testStore(t)
	s.Append("s1", Entry{Msg: "x"})

	if err := s.Clear("s1"); err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(s.basePath, "s1.jsonl")
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("shard file should not exist after clear")
	}
}

func TestClearNonexistent(t *testing.T) {
	s := testStore(t)
	if err := s.Clear("nope"); err != nil {
		t.Fatalf("clear nonexistent should not error: %v", err)
	}
}

func TestShardEscapeRejected(t *testing.T) {
	s := testStore(t)
	for _, shard := range []string{"../escape", "../../etc/cron", "a/../../escape"} {
		if err := s.Append(shard, Entry{Msg: "x"}); err == nil {
			t.Fatalf("Append(%q) should be rejected", shard)
		}
		if _, err := s.ReadAll(shard); err == nil {
			t.Fatalf("ReadAll(%q) should be rejected", shard)
		}
		if err := s.Update(shard, 1, "x"); err == nil {
			t.Fatalf("Update(%q) should be rejected", shard)
		}
		if err := s.Delete(shard, 1); err == nil {
			t.Fatalf("Delete(%q) should be rejected", shard)
		}
		if err := s.Clear(shard); err == nil {
			t.Fatalf("Clear(%q) should be rejected", shard)
		}
	}
}

func TestNestedShardAllowed(t *testing.T) {
	s := testStore(t)
	if err := s.Append("tasks/task_123", Entry{Msg: "x"}); err != nil {
		t.Fatal(err)
	}
	entries, err := s.ReadAll("tasks/task_123")
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("got %d entries, want 1", len(entries))
	}
}
