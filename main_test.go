package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestMain lets tests run the CLI by re-executing the test binary:
// with CTXLOG_RUN_MAIN=1 it calls main() instead of running tests.
func TestMain(m *testing.M) {
	if os.Getenv("CTXLOG_RUN_MAIN") == "1" {
		main()
		os.Exit(0)
	}
	os.Exit(m.Run())
}

func runCLI(t *testing.T, dir string, args ...string) (string, error) {
	t.Helper()
	cmd := exec.Command(os.Args[0], args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "CTXLOG_RUN_MAIN=1", "HOME="+dir)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func TestInvalidInputRejected(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{"negative lines", []string{"read", "-shard=s", "-lines=-1"}, "-lines must be >= 1"},
		{"unquoted msg", []string{"append", "-shard=s", "-msg=hello", "world"}, `unexpected argument "world"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			out, err := runCLI(t, dir, tt.args...)
			if err == nil {
				t.Fatalf("expected non-zero exit, output: %s", out)
			}
			if !strings.Contains(out, tt.want) {
				t.Fatalf("output %q does not contain %q", out, tt.want)
			}
			if _, err := os.Stat(filepath.Join(dir, ".ctxlog", "s.jsonl")); !os.IsNotExist(err) {
				t.Fatal("rejected command must not write to the shard")
			}
		})
	}
}

func TestReadDoesNotCreateDir(t *testing.T) {
	dir := t.TempDir()
	if out, err := runCLI(t, dir, "read", "-shard=s"); err != nil {
		t.Fatalf("read: %v, output: %s", err, out)
	}
	if _, err := os.Stat(filepath.Join(dir, ".ctxlog")); !os.IsNotExist(err) {
		t.Fatal("read created .ctxlog/")
	}
}
