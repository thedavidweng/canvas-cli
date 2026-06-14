# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/).

## [0.3.0] - 2026-06-14


### ♻️ Refactoring

- Use reusable workflows from cli-workflow-template

### ✨ New Features

- Add experimental session cookie authentication

### 🐛 Bug Fixes

- Resolve all golangci-lint issues
- Correct comment numbering in doctor.go
- Address session cookie auth security and usability issues
- Route upload requests through client auth/CSRF/redirect path
- Only check cookie session expiry in pagination when cookie auth active
- Address Greptile review and Windows CI failure
- Skip secret file permission check on Windows
- Skip permission-rejection tests on Windows
- Add DoURLWithHeaders for same-host multipart uploads
- Register Chrome cookie finder and reduce keychain prompts

### 📝 Documentation

- Add verification checklist to AGENTS.md
- Add install scripts and streamline README
- Add infrastructure links (CI/CD and docs)
- Add v0.3.0 and v0.2.0 changelog entries

### 📦 Dependencies

- Add dependabot for gomod and github-actions
- **deps**: Bump golangci/golangci-lint-action from 7 to 9
- Add permissions block to codeql workflow

### 🔧 Chores

- Remove unnecessary code (ponytail sweep)
## [0.2.0] - 2026-06-13


### ♻️ Refactoring

- Deep modules and architecture overhaul

### ✨ New Features

- Interactive login, multi-profile support, better error messages
- Course export commands and expanded export-context

### 🐛 Bug Fixes

- Unblock CI — gitignore was excluding cmd/canvas and internal/canvas
- CI lint and Windows test failures
- Explicitly disable errcheck linter (default in golangci-lint v2)
- Resolve all golangci-lint issues
- Migrate goreleaser brews to homebrew_casks
- Use OS-appropriate config and state directories
## [0.1.0] - 2026-06-13


### ✨ New Features

- Initial release of canvas-cli

