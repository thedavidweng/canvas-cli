# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

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
