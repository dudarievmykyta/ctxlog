// Package memory implements sharded JSONL storage with CRUD operations
// and BSD flock for cross-process safety.
package memory

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

// Entry is a single JSONL record.
type Entry struct {
	Ts    int64  `json:"ts"`
	Agent string `json:"agent"`
	Msg   string `json:"msg"`
}

// Store manages the on-disk .ctxlog directory.
type Store struct {
	basePath string
}

// NewStore creates a Store rooted at basePath/.ctxlog.
func NewStore(basePath string) *Store {
	return &Store{basePath: filepath.Join(basePath, ".ctxlog")}
}

// Init ensures the base .ctxlog directory exists.
func (s *Store) Init() error {
	return os.MkdirAll(s.basePath, 0o755)
}

func (s *Store) shardPath(shard string) (string, error) {
	path := filepath.Join(s.basePath, shard+".jsonl")
	if !strings.HasPrefix(path, s.basePath+string(filepath.Separator)) {
		return "", fmt.Errorf("invalid shard name %q: escapes storage directory", shard)
	}
	return path, nil
}

// Append writes a single Entry to the named shard.
func (s *Store) Append(shard string, e Entry) error {
	if e.Ts == 0 {
		e.Ts = time.Now().Unix()
	}

	path, err := s.shardPath(shard)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("mkdir shard dir: %w", err)
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("open shard: %w", err)
	}
	defer f.Close()

	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		return fmt.Errorf("flock: %w", err)
	}
	defer syscall.Flock(int(f.Fd()), syscall.LOCK_UN)

	data, err := json.Marshal(e)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	data = append(data, '\n')

	if _, err := f.Write(data); err != nil {
		return fmt.Errorf("write: %w", err)
	}
	return nil
}

// ReadAll returns all entries from the named shard.
// Returns an empty slice if the shard does not exist.
func (s *Store) ReadAll(shard string) ([]Entry, error) {
	path, err := s.shardPath(shard)
	if err != nil {
		return nil, err
	}

	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []Entry{}, nil
		}
		return nil, fmt.Errorf("open shard: %w", err)
	}
	defer f.Close()

	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_SH); err != nil {
		return nil, fmt.Errorf("flock: %w", err)
	}
	defer syscall.Flock(int(f.Fd()), syscall.LOCK_UN)

	var entries []Entry
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var e Entry
		if err := json.Unmarshal(line, &e); err != nil {
			continue
		}
		entries = append(entries, e)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan: %w", err)
	}
	return entries, nil
}

// ReadRecent returns the last maxLines entries from the named shard.
func (s *Store) ReadRecent(shard string, maxLines int) ([]Entry, error) {
	all, err := s.ReadAll(shard)
	if err != nil {
		return nil, err
	}
	if len(all) <= maxLines {
		return all, nil
	}
	return all[len(all)-maxLines:], nil
}

// Delete removes the entry at the given 1-based line index from the shard.
func (s *Store) Delete(shard string, lineIndex int) error {
	path, err := s.shardPath(shard)
	if err != nil {
		return err
	}

	f, err := os.OpenFile(path, os.O_RDWR, 0o644)
	if err != nil {
		return fmt.Errorf("open shard: %w", err)
	}
	defer f.Close()

	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		return fmt.Errorf("flock: %w", err)
	}
	defer syscall.Flock(int(f.Fd()), syscall.LOCK_UN)

	lines, err := readLines(f)
	if err != nil {
		return err
	}

	idx := lineIndex - 1
	if idx < 0 || idx >= len(lines) {
		return fmt.Errorf("line %d out of range (1-%d)", lineIndex, len(lines))
	}

	lines = append(lines[:idx], lines[idx+1:]...)
	return truncateAndWrite(f, lines)
}

// Update replaces the msg (and refreshes ts) of the entry at the given 1-based line index.
func (s *Store) Update(shard string, lineIndex int, newMsg string) error {
	path, err := s.shardPath(shard)
	if err != nil {
		return err
	}

	f, err := os.OpenFile(path, os.O_RDWR, 0o644)
	if err != nil {
		return fmt.Errorf("open shard: %w", err)
	}
	defer f.Close()

	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		return fmt.Errorf("flock: %w", err)
	}
	defer syscall.Flock(int(f.Fd()), syscall.LOCK_UN)

	lines, err := readLines(f)
	if err != nil {
		return err
	}

	idx := lineIndex - 1
	if idx < 0 || idx >= len(lines) {
		return fmt.Errorf("line %d out of range (1-%d)", lineIndex, len(lines))
	}

	var e Entry
	if err := json.Unmarshal(lines[idx], &e); err != nil {
		return fmt.Errorf("parse line %d: %w", lineIndex, err)
	}

	e.Msg = newMsg
	e.Ts = time.Now().Unix()

	updated, err := json.Marshal(e)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	lines[idx] = updated
	return truncateAndWrite(f, lines)
}

// Clear removes the entire shard file.
func (s *Store) Clear(shard string) error {
	path, err := s.shardPath(shard)
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove shard: %w", err)
	}
	return nil
}

func readLines(f *os.File) ([][]byte, error) {
	if _, err := f.Seek(0, 0); err != nil {
		return nil, fmt.Errorf("seek: %w", err)
	}
	var lines [][]byte
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		cp := make([]byte, len(line))
		copy(cp, line)
		lines = append(lines, cp)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan: %w", err)
	}
	return lines, nil
}

func truncateAndWrite(f *os.File, lines [][]byte) error {
	if err := f.Truncate(0); err != nil {
		return fmt.Errorf("truncate: %w", err)
	}
	if _, err := f.Seek(0, 0); err != nil {
		return fmt.Errorf("seek: %w", err)
	}
	for _, line := range lines {
		line = append(line, '\n')
		if _, err := f.Write(line); err != nil {
			return fmt.Errorf("write: %w", err)
		}
	}
	return nil
}
