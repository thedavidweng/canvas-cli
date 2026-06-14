# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [0.3.0] - 2026-06-14

### Added
- Experimental session cookie authentication via `--cookie-stdin`, `--cookie-env`, `--cookie-file`, or browser auto-extraction.
- Browser cookie extraction from Chrome using kooky (macOS/Linux/Windows).
- Default browser detection on macOS for cookie extraction.
- `canvas auth login --browser` flag for interactive browser cookie extraction.
- Keyring access warning before browser cookie extraction.
- `DoURLWithHeaders` for same-host multipart uploads with auth/CSRF headers.

### Fixed
- Register Chrome cookie store finder — previously `ExtractCookies` always returned 0 cookies.
- Stop at first valid session cookie instead of reading all browser stores in parallel.
- Skip permission-rejection tests on Windows.
- Skip secret file permission check on Windows.

## [0.2.0] - 2026-06-13

### Added
- Interactive login flow with token and cookie auth methods.
- Multi-profile support with `canvas auth use`, `canvas auth profiles`.
- Course export commands and expanded `export-context`.
- OS-appropriate config and state directories.

### Changed
- Deep modules and architecture overhaul.

## [0.1.0] - 2026-06-12

### Added
- Initial release of canvas-cli.
- Full Canvas LMS CLI with 50+ commands across courses, modules, assignments, submissions, discussions, files, pages, inbox, enrollments, sections, users, rubrics, and grading.
- JSON envelope output with stable schema versioning.
- Automatic pagination with Link header following.
- Rate limit handling with exponential backoff and Retry-After support.
- Mutation safety model with --dry-run, --confirm, --read-only gates.
- Audit logging for write operations.
- Raw API passthrough (`canvas api get/post/put/delete`).
- Course context export (`canvas courses export-context`).
- File upload (3-step Canvas flow) and bulk download.
- Submission download with deterministic path layout and manifest.
- Grade import from CSV.
- Shell completion for bash, zsh, fish, and powershell.
- Cross-platform builds (linux, darwin, windows x amd64, arm64).
- Homebrew distribution via tap.
- Cosign-signed release checksums and SBOM generation.
- CodeQL security analysis.
