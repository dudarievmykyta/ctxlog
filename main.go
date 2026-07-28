// ctxlog — a lightweight CLI tool for persistent, sharded context logging.
//
// Stores append-only JSONL entries in .ctxlog/<shard>.jsonl within the
// current working directory. Uses BSD flock for cross-process safety.
package main

import (
	_ "embed"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/dudarievmykyta/ctxlog/memory"
)

//go:embed skills/claude/SKILL.md
var claudeSkill []byte

func printUsage() {
	fmt.Fprintf(os.Stderr, `ctxlog — persistent, sharded context logging for AI agent sessions

Usage:
  ctxlog <command> [flags]

Commands:
  append    Append a new entry to a shard
  read      Read recent entries from a shard
  search    Search entries in a shard by substring
  update    Update an existing entry by line number
  delete    Delete an entry by line number
  clear     Remove an entire shard file
  install   Install agent skill (-type=claude)

Global flags:
  -global    Use ~/.ctxlog/ instead of <cwd>/.ctxlog/

Flags by command:

  append  -shard=<name>  -msg=<text>  [-agent=<id>]  [-global]
  read    -shard=<name>  [-lines=10]  [-global]
  search  -shard=<name>  -term=<text>  [-global]
  update  -shard=<name>  -line=<num>  -msg=<new_text>  [-global]
  delete  -shard=<name>  -line=<num>  [-global]
  clear   -shard=<name>  [-global]

Examples:
  ctxlog append -shard="auth-refactor" -msg="switched to JWT middleware" -agent="claude"
  ctxlog read   -shard="auth-refactor" -lines=5
  ctxlog search -shard="auth-refactor" -term="jwt"
  ctxlog append -global -shard="notes" -msg="global note across projects"
  ctxlog read   -global -shard="notes"
  ctxlog update -shard="auth-refactor" -line=2 -msg="JWT middleware validated in staging"
  ctxlog delete -shard="auth-refactor" -line=3
  ctxlog clear  -shard="auth-refactor"
  ctxlog install -type=claude
`)
}

func main() {
	if len(os.Args) < 2 || os.Args[1] == "--help" || os.Args[1] == "-h" || os.Args[1] == "help" {
		printUsage()
		if len(os.Args) < 2 {
			os.Exit(1)
		}
		return
	}

	switch os.Args[1] {
	case "append":
		cmdAppend(os.Args[2:])
	case "read":
		cmdRead(os.Args[2:])
	case "search":
		cmdSearch(os.Args[2:])
	case "update":
		cmdUpdate(os.Args[2:])
	case "delete":
		cmdDelete(os.Args[2:])
	case "clear":
		cmdClear(os.Args[2:])
	case "install":
		cmdInstall(os.Args[2:])
	default:
		fatalf("unknown command: %q\nRun 'ctxlog --help' for usage.", os.Args[1])
	}
}

func cmdAppend(args []string) {
	shard, msg, _, agent, global := parseFlags("append", args, true, true, false, true)

	store := getStore(global)
	err := store.Append(shard, memory.Entry{
		Agent: agent,
		Msg:   msg,
	})
	if err != nil {
		fatalf("append: %v", err)
	}

	fmt.Fprintf(os.Stderr, "ok: appended to .ctxlog/%s.jsonl\n", shard)
}

func cmdRead(args []string) {
	fs := flag.NewFlagSet("read", flag.ExitOnError)
	shard := fs.String("shard", "", "shard name (required)")
	lines := fs.Int("lines", 10, "number of recent lines to return")
	global := fs.Bool("global", false, "use ~/.ctxlog/")
	fs.Parse(args)

	if *shard == "" {
		fatalf("read: -shard is required")
	}

	store := getStore(*global)

	matches, err := store.ReadRecent(*shard, *lines)
	if err != nil {
		fatalf("read: %v", err)
	}

	printMatches(matches)
}

func cmdSearch(args []string) {
	fs := flag.NewFlagSet("search", flag.ExitOnError)
	shard := fs.String("shard", "", "shard name (required)")
	term := fs.String("term", "", "text to search for (required)")
	global := fs.Bool("global", false, "use ~/.ctxlog/")
	fs.Parse(args)

	if *shard == "" {
		fatalf("search: -shard is required")
	}
	if *term == "" {
		fatalf("search: -term is required")
	}

	store := getStore(*global)

	matches, err := store.Search(*shard, *term)
	if err != nil {
		fatalf("search: %v", err)
	}

	printMatches(matches)
}

func printMatches(matches []memory.Match) {
	for _, m := range matches {
		data, _ := json.Marshal(m.Entry)
		fmt.Printf("%d: %s\n", m.Line, data)
	}
}

func cmdUpdate(args []string) {
	shard, msg, line, _, global := parseFlags("update", args, true, true, true, false)

	store := getStore(global)
	if err := store.Update(shard, line, msg); err != nil {
		fatalf("update: %v", err)
	}

	fmt.Fprintf(os.Stderr, "ok: updated line %d in .ctxlog/%s.jsonl\n", line, shard)
}

func cmdDelete(args []string) {
	shard, _, line, _, global := parseFlags("delete", args, true, false, true, false)

	store := getStore(global)
	if err := store.Delete(shard, line); err != nil {
		fatalf("delete: %v", err)
	}

	fmt.Fprintf(os.Stderr, "ok: deleted line %d from .ctxlog/%s.jsonl\n", line, shard)
}

func cmdClear(args []string) {
	shard, _, _, _, global := parseFlags("clear", args, true, false, false, false)

	store := getStore(global)
	if err := store.Clear(shard); err != nil {
		fatalf("clear: %v", err)
	}

	fmt.Fprintf(os.Stderr, "ok: cleared .ctxlog/%s.jsonl\n", shard)
}

var agents = map[string]struct {
	content  []byte
	checkDir string // directory that must exist (relative to $HOME)
	destDir  string // install destination (relative to $HOME)
}{
	"claude": {
		content:  claudeSkill,
		checkDir: ".claude",
		destDir:  filepath.Join(".claude", "skills", "ctxlog"),
	},
}

func cmdInstall(args []string) {
	fs := flag.NewFlagSet("install", flag.ExitOnError)
	agentType := fs.String("type", "", "agent type to install skill for (e.g. claude)")
	fs.Parse(args)

	if *agentType == "" {
		supported := make([]string, 0, len(agents))
		for k := range agents {
			supported = append(supported, k)
		}
		fatalf("install: -type is required\nSupported: %v", supported)
	}

	agent, ok := agents[*agentType]
	if !ok {
		fatalf("install: unknown agent type %q", *agentType)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		fatalf("install: cannot determine home directory: %v", err)
	}

	checkPath := filepath.Join(home, agent.checkDir)
	if _, err := os.Stat(checkPath); os.IsNotExist(err) {
		fatalf("install: %s not found.\nPlease install %s first, then re-run this command.", checkPath, *agentType)
	}

	dir := filepath.Join(home, agent.destDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		fatalf("install: mkdir: %v", err)
	}

	path := filepath.Join(dir, "SKILL.md")
	if err := os.WriteFile(path, agent.content, 0o644); err != nil {
		fatalf("install: write: %v", err)
	}

	fmt.Printf("installed %s skill to %s\n", *agentType, path)
}

func parseFlags(cmd string, args []string, needShard, needMsg, needLine, needAgent bool) (shard, msg string, line int, agent string, global bool) {
	fs := flag.NewFlagSet(cmd, flag.ExitOnError)
	shardP := fs.String("shard", "", "")
	msgP := fs.String("msg", "", "")
	lineP := fs.Int("line", 0, "")
	agentP := fs.String("agent", "", "")
	globalP := fs.Bool("global", false, "")
	fs.Parse(args)

	if needShard && *shardP == "" {
		fatalf("%s: -shard is required", cmd)
	}
	if needMsg && *msgP == "" {
		fatalf("%s: -msg is required", cmd)
	}
	if needLine && *lineP <= 0 {
		fatalf("%s: -line is required (must be >= 1)", cmd)
	}
	return *shardP, *msgP, *lineP, *agentP, *globalP
}

func getStore(global bool) *memory.Store {
	var base string
	if global {
		home, err := os.UserHomeDir()
		if err != nil {
			fatalf("home: %v", err)
		}
		base = home
	} else {
		cwd, err := os.Getwd()
		if err != nil {
			fatalf("getwd: %v", err)
		}
		base = cwd
	}
	store := memory.NewStore(base)
	if err := store.Init(); err != nil {
		fatalf("init: %v", err)
	}
	return store
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
