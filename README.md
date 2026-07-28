# ctxlog

Lightweight CLI coordination journal for AI agent sessions. Per-shard JSONL files with BSD `flock` for safe concurrent writes. Zero external dependencies — only Go standard library.

## What it is (and is not)

ctxlog is the write side of agent context: a progress journal that agents append to as they work — "step 3 done, touched auth.go" — and read back when a new session or a parallel agent picks up the task. Appends are atomic under concurrency (`flock` + `O_APPEND`), so N subagents can safely share one journal during fan-out.

It is not a memory or knowledge base: entries are claims made by past agents, not verified facts. It is also not a transcript search tool — read-side tools like [ctxrs/ctx](https://github.com/ctxrs/ctx) index existing agent histories; ctxlog complements them rather than competing.

## Requirements

- Go 1.26+
- macOS or Linux (uses BSD `flock` for cross-process file locking)

## Install

```bash
brew tap dudarievmykyta/tools
brew install ctxlog
```

## Build from source

```bash
go build -o ctxlog .
```

## Usage

### Append an entry

```bash
ctxlog append -shard="auth" -msg="Fixed DB connection pooling bug" -agent="claude"
```

`-agent` is optional. `-shard` and `-msg` are required.

Status is printed to stderr:

```
ok: appended to .ctxlog/auth.jsonl
```

### Read recent entries

```bash
ctxlog read -shard="auth" -lines=5
```

`-lines` defaults to 10. Output goes to stdout; each entry is prefixed with its real line number in the shard file, so it can be passed to `update`/`delete` directly:

```
41: {"ts":1773685005,"agent":"claude","msg":"Fixed DB connection pooling bug"}
```

Returns empty output if the shard doesn't exist yet.

### Search entries

```bash
ctxlog search -shard="auth" -term="jwt"
```

Case-insensitive substring match over `msg`. Same output format as `read`, with real line numbers.

### Update an entry

```bash
ctxlog update -shard="auth" -line=2 -msg="Updated: connection pool size set to 25"
```

Updates the message and refreshes the timestamp at the given 1-based line number.

### Delete an entry

```bash
ctxlog delete -shard="auth" -line=3
```

Removes the entry at the given line number. Remaining lines are renumbered.

### Clear a shard

```bash
ctxlog clear -shard="auth"
```

Deletes the entire shard file.

### Install agent skill

```bash
ctxlog install -type=claude
```

Installs the skill file for the specified agent. Checks that the agent's config directory exists first (e.g. `~/.claude` for Claude Code).

Supported agents: `claude`. More coming soon.

## File structure on disk

```
<cwd>/
└── .ctxlog/
    ├── auth.jsonl
    └── tasks/
        └── task_123.jsonl
```

Each `.jsonl` file contains one JSON object per line:

```json
{"ts":1773685005,"agent":"claude","msg":"Fixed DB connection pooling bug"}
```

## Flags reference

| Command | Flag | Required | Default | Description |
|---------|------|----------|---------|-------------|
| `append` | `-shard` | yes | — | Shard name (supports `/` for nesting) |
| `append` | `-msg` | yes | — | Message to log |
| `append` | `-agent` | no | `""` | Agent identifier |
| `read` | `-shard` | yes | — | Shard name |
| `read` | `-lines` | no | `10` | Number of recent entries to return |
| `search` | `-shard` | yes | — | Shard name |
| `search` | `-term` | yes | — | Substring to match (case-insensitive) |
| `update` | `-shard` | yes | — | Shard name |
| `update` | `-line` | yes | — | 1-based line number to update |
| `update` | `-msg` | yes | — | New message text |
| `delete` | `-shard` | yes | — | Shard name |
| `delete` | `-line` | yes | — | 1-based line number to delete |
| `clear` | `-shard` | yes | — | Shard name to remove |
| `install` | `-type` | yes | — | Agent type (`claude`) |

All data commands (`append`, `read`, `search`, `update`, `delete`, `clear`) accept `-global` to use `~/.ctxlog/` instead of `<cwd>/.ctxlog/`.

## Concurrency

Safe for concurrent use across multiple processes:

- BSD `flock` on each shard file — exclusive for writes, shared for reads
- `O_APPEND` mode for kernel-level write atomicity

`update` and `delete` address entries by line number, which shifts when another process deletes a line. Treat them as single-writer maintenance operations; in fan-out (multiple concurrent writers) stick to `append`, `read`, and `search`.

## Project structure

```
├── go.mod
├── main.go               # CLI entry point: subcommands, flags, help, install
├── memory/
│   └── memory.go         # Store: Append, ReadAll, ReadRecent, Search, Update, Delete, Clear
├── skills/
│   └── claude/SKILL.md   # Embedded skill prompt for Claude Code
└── README.md
```
