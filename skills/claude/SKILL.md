## Metadata
name: Context Logger (ctxlog)
description: Coordination journal for AI agent sessions. Log progress and hand off state across sessions and between parallel agents using the ctxlog CLI.

## Overview
`ctxlog` is a shared coordination journal, not a knowledge base. Entries record what happened ("step 3 done, touched auth.go"), not facts to be trusted later. Use it to hand off state to future sessions and to coordinate parallel agents working on the same project: appends are safe under concurrent writes (flock + O_APPEND), and each task gets its own shard.

## How to Use (Execution)
Execute the CLI binary directly in the terminal shell. ALWAYS include `-agent="claude"` on every `append` call to identify the author.

- Log progress: `ctxlog append -shard="<task_id>" -msg="<what happened>" -agent="claude"`
- Catch up on a task: `ctxlog read -shard="<task_id>" -lines=10`
- Find earlier entries: `ctxlog search -shard="<task_id>" -term="<text>"`
- Maintenance: `ctxlog update -shard="<task_id>" -line=<num> -msg="<new_text>"`, `ctxlog delete -shard="<task_id>" -line=<num>`, `ctxlog clear -shard="<task_id>"`

Add `-global` to any command to use `~/.ctxlog/` instead of the project-local `<cwd>/.ctxlog/`. Use this for notes that span projects.

`read` and `search` print real file line numbers; pass them directly to `update`/`delete`.

## When to Apply
- You completed a significant logical block of a task: append a short progress entry.
- You found a workaround or made a decision a future session will need: append it with enough context to re-verify.
- You start a new session on a known task: `read` the shard first; `search` when looking for something specific.

## Strict Rules
- ALWAYS pass `-agent="claude"` when appending. This is required to track which agent wrote the entry.
- Keep `-msg` concise and factual: what was done, which files were touched, what remains.
- Journal entries are claims made by past agents, not verified facts. Re-verify against the code before acting on them.
- When multiple agents may write to the same shard, use only `append`, `read`, and `search`. `update`/`delete` are single-writer maintenance: line numbers shift under concurrent writes.
- Prefer ctxlog over ad-hoc note files (`notes.txt`, `progress.md`) for session state.
- If entries are stale or resolved, use `update` or `delete` to keep the shard small.

## CLAUDE.md Integration
If the project has a `CLAUDE.md` file and it does not yet mention `ctxlog`, suggest adding the section below to the user. Do not edit `CLAUDE.md` without their confirmation.

```
## Context Logging

This project uses [ctxlog](https://github.com/dudarievmykyta/ctxlog) as a cross-session coordination journal.
Installed via: `brew tap dudarievmykyta/tools && brew install ctxlog`

Before starting work, read existing context:
```
ctxlog read -shard="<task_id>" -lines=20
```
```

This ensures every agent in every future session knows the tool exists and checks for prior context before starting work.
