# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.4.0] - 2026-07-28

### Added

- `search` command: case-insensitive substring search over entry messages (`ctxlog search -shard=<name> -term=<text>`), same output format as `read` ([#5])

### Changed

- `read` prints each entry's real line number in the shard file instead of its position within the returned window, so the numbers can be passed to `update`/`delete` directly ([#5])
- Documentation repositioned: ctxlog is a coordination journal for agent sessions, not a memory system; `update`/`delete` are documented as single-writer maintenance operations ([#3])
- GitHub Actions bumped to current majors ([#4])

### Fixed

- Shard names that escape the `.ctxlog/` directory (e.g. `../../x`) are rejected across all commands ([#1])

## [0.3.2] - 2026-06-09

### Changed

- Go 1.26; fixed goreleaser deprecation warnings and the brew formula test

## [0.3.1] - 2026-03-17

### Changed

- Repository moved: GitHub profile renamed from `dudarevnikita` to `dudarievmykyta`

## [0.3.0] - 2026-03-16

### Added

- `install -type=<agent>` flag; skill prompts live in `skills/<agent>/SKILL.md` and are embedded via `go:embed`
- `-global` flag on all data commands to use `~/.ctxlog/` instead of `<cwd>/.ctxlog/`

## [0.2.3] - 2026-03-16

### Added

- Tests for the `memory` package and a CI test workflow

## [0.2.2] - 2026-03-16

### Fixed

- Outdated "append-only" wording in docs and the package comment

## [0.2.1] - 2026-03-16

### Changed

- README and CLAUDE.md updated with the CRUD commands

## [0.2.0] - 2026-03-16

### Added

- `update`, `delete`, and `clear` commands
- Custom help output

## [0.1.2] - 2026-03-16

### Removed

- Unused release setup doc

## [0.1.1] - 2026-03-16

### Added

- Homebrew install instructions

## [0.1.0] - 2026-03-16

### Added

- Initial release: `append` and `read` over per-shard JSONL files with BSD `flock`, `install` command, release pipeline

[Unreleased]: https://github.com/dudarievmykyta/ctxlog/compare/v0.4.0...HEAD
[0.4.0]: https://github.com/dudarievmykyta/ctxlog/compare/v0.3.2...v0.4.0
[0.3.2]: https://github.com/dudarievmykyta/ctxlog/compare/v0.3.1...v0.3.2
[0.3.1]: https://github.com/dudarievmykyta/ctxlog/compare/v0.3.0...v0.3.1
[0.3.0]: https://github.com/dudarievmykyta/ctxlog/compare/v0.2.3...v0.3.0
[0.2.3]: https://github.com/dudarievmykyta/ctxlog/compare/v0.2.2...v0.2.3
[0.2.2]: https://github.com/dudarievmykyta/ctxlog/compare/v0.2.1...v0.2.2
[0.2.1]: https://github.com/dudarievmykyta/ctxlog/compare/v0.2.0...v0.2.1
[0.2.0]: https://github.com/dudarievmykyta/ctxlog/compare/v0.1.2...v0.2.0
[0.1.2]: https://github.com/dudarievmykyta/ctxlog/compare/v0.1.1...v0.1.2
[0.1.1]: https://github.com/dudarievmykyta/ctxlog/compare/v0.1.0...v0.1.1
[0.1.0]: https://github.com/dudarievmykyta/ctxlog/releases/tag/v0.1.0
[#1]: https://github.com/dudarievmykyta/ctxlog/pull/1
[#3]: https://github.com/dudarievmykyta/ctxlog/pull/3
[#4]: https://github.com/dudarievmykyta/ctxlog/pull/4
[#5]: https://github.com/dudarievmykyta/ctxlog/pull/5
