# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [0.4.1](https://github.com/thedavidweng/canvas-cli/compare/v0.4.0...v0.4.1) (2026-08-22)


### Bug Fixes

* let release-please create tags and releases ([7dbf32c](https://github.com/thedavidweng/canvas-cli/commit/7dbf32c6ca5e6531ea1a6e53b578f1bf4c51d74e))


### Documentation

* add Diátaxis tutorial and how-to guides ([74cab50](https://github.com/thedavidweng/canvas-cli/commit/74cab50e85af411b606a1f7d7ddc3c6be16f5aac))
* remove retired Go Report Card badge ([041efce](https://github.com/thedavidweng/canvas-cli/commit/041efceaa98dc80564d5d22c1da6d1d988786cd4))

## [0.4.0](https://github.com/thedavidweng/canvas-cli/compare/v0.3.0...v0.4.0) (2026-07-25)


### Features

* add experimental session cookie authentication ([2cef687](https://github.com/thedavidweng/canvas-cli/commit/2cef6878ca39bb61842dd3942daf03f6985f9d54))
* align with CLI fleet standard ([fc14da6](https://github.com/thedavidweng/canvas-cli/commit/fc14da6b8e208683a9042c077b42596483215472))
* course export commands and expanded export-context ([afcd36d](https://github.com/thedavidweng/canvas-cli/commit/afcd36dedcd666e6fd03ebf72e35d32bc0b6d273))
* experimental session cookie authentication ([3c0428a](https://github.com/thedavidweng/canvas-cli/commit/3c0428af656d07d2c8e12df38ea76a025de5911a))
* fleet envelope core and zero-deviation lint config ([116b247](https://github.com/thedavidweng/canvas-cli/commit/116b247131612b6517769152ba9c32c8ec7e5dcf))
* initial release of canvas-cli ([03962ae](https://github.com/thedavidweng/canvas-cli/commit/03962ae831f850558d09eead3e55ace76955405e))
* interactive login, multi-profile support, better error messages ([0923bb4](https://github.com/thedavidweng/canvas-cli/commit/0923bb47d95bbcf4a55d83972b5b15b3481e5acd))


### Bug Fixes

* add DoURLWithHeaders for same-host multipart uploads ([55fd63e](https://github.com/thedavidweng/canvas-cli/commit/55fd63e66e1c4fc7d82c9d445f993ea6edbe7dbc))
* add force_push and fetch-depth: 0 ([d8dcdbb](https://github.com/thedavidweng/canvas-cli/commit/d8dcdbb669f48045dca7cefdd34874371aac33ca))
* add force_push and fetch-depth: 0 ([5ac736b](https://github.com/thedavidweng/canvas-cli/commit/5ac736b5ea44e5af3cab9425292b096ca165997e))
* address go production review findings ([8ce46da](https://github.com/thedavidweng/canvas-cli/commit/8ce46dadc330dfbc81e690259e408fa46bda7995))
* address Greptile review and Windows CI failure ([5ae5010](https://github.com/thedavidweng/canvas-cli/commit/5ae5010f25eed3fa73bac69b3277bb76c5196517))
* address PR review findings ([019a7e6](https://github.com/thedavidweng/canvas-cli/commit/019a7e693439eb723c493219dc3483b78b40acfc))
* address session cookie auth security and usability issues ([bf7a9f9](https://github.com/thedavidweng/canvas-cli/commit/bf7a9f94291790eeab77dd184af8e2c50c48907d))
* CI lint and Windows test failures ([6f55742](https://github.com/thedavidweng/canvas-cli/commit/6f55742b6f0bec122aff3a81506d43b086f9f13a))
* correct comment numbering in doctor.go ([3d42c76](https://github.com/thedavidweng/canvas-cli/commit/3d42c76374b83ef40c00cc0d99162121ca56019f))
* correct mirror action SHA ([3c6b71b](https://github.com/thedavidweng/canvas-cli/commit/3c6b71b86d50e969862076bfea89b8d7eab12051))
* correct mirror action SHA ([271ea96](https://github.com/thedavidweng/canvas-cli/commit/271ea96307d16646edf37b95be88de919f6e6cc0))
* cosign v2 args (--output-signature/--output-certificate → --bundle) ([d46ff46](https://github.com/thedavidweng/canvas-cli/commit/d46ff468e13e2c37e6bf3a17cadb4129baf34d6e))
* explicitly disable errcheck linter (default in golangci-lint v2) ([bee419f](https://github.com/thedavidweng/canvas-cli/commit/bee419f575e60beb144906593200fbbec39908dd))
* migrate goreleaser brews to homebrew_casks ([6be96f7](https://github.com/thedavidweng/canvas-cli/commit/6be96f7758dd3543d120b8c9c97ab64ab822f53e))
* only check cookie session expiry in pagination when cookie auth active ([ec5e3fc](https://github.com/thedavidweng/canvas-cli/commit/ec5e3fc6addfc7705818356d9beb1f8a0f53a536))
* pin action SHA, remove test.txt, add permissions ([2dc44a6](https://github.com/thedavidweng/canvas-cli/commit/2dc44a68ec865eb1c9a73747e8b6c8fb5c470ba0))
* repair release-please workflow YAML ([fe9932d](https://github.com/thedavidweng/canvas-cli/commit/fe9932dd9ab2bd7628c4c7d5fecd5155c0e0647a))
* resolve all golangci-lint issues ([1a8e3b1](https://github.com/thedavidweng/canvas-cli/commit/1a8e3b1cdc1716ad17d586491e8979563e814836))
* resolve all golangci-lint issues ([eba33d5](https://github.com/thedavidweng/canvas-cli/commit/eba33d532de5bece36edcb6e2039969c578bf57a))
* resolve errcheck findings and test assertions ([20f7140](https://github.com/thedavidweng/canvas-cli/commit/20f71409b6eb3c5478c9e08c15bb152f2507f8f7))
* restrict mirror workflow to main and tags ([4a62716](https://github.com/thedavidweng/canvas-cli/commit/4a627166e3de03e1e504685dedf236c1411233b0))
* route upload requests through client auth/CSRF/redirect path ([fbc2982](https://github.com/thedavidweng/canvas-cli/commit/fbc29823795353fe3e3e52344bb7668ab3933be7))
* skip permission-rejection tests on Windows ([6460f85](https://github.com/thedavidweng/canvas-cli/commit/6460f855d9029ee2ca0d2b767a04be21bf3665f9))
* skip secret file permission check on Windows ([294da83](https://github.com/thedavidweng/canvas-cli/commit/294da83a8f326fd6aa1359c2e2f553a4be4cfc04))
* skip XDG_STATE_HOME test on Windows ([8a9d8dd](https://github.com/thedavidweng/canvas-cli/commit/8a9d8ddaf0f9708ce9d053e0008c36c4c70dec88))
* unblock CI — gitignore was excluding cmd/canvas and internal/canvas ([04866b5](https://github.com/thedavidweng/canvas-cli/commit/04866b5a6ab36a4a01cefc353cfde333071c2e0b))
* use Homebrew Cask and fix install script quoting ([156d886](https://github.com/thedavidweng/canvas-cli/commit/156d886be7960681afd2d9d4437861cddcc71615))
* use OS-appropriate config and state directories ([36ba715](https://github.com/thedavidweng/canvas-cli/commit/36ba7152ca0eb225893c07c855ab33b82b791b02))


### Performance

* optimize logo to WebP ([50594f4](https://github.com/thedavidweng/canvas-cli/commit/50594f40120c2fe06c79404b8973db1e2f0278bb))


### Refactoring

* code review cleanup across CLI and canvas packages ([#15](https://github.com/thedavidweng/canvas-cli/issues/15)) ([bc95f53](https://github.com/thedavidweng/canvas-cli/commit/bc95f53684b4c705c5c400f23097e74947a94028))
* deep modules and architecture overhaul ([4c3eac2](https://github.com/thedavidweng/canvas-cli/commit/4c3eac202b40aea46101f1fb32d6fbff366470cc))
* deepen shallow modules and centralize mutation pipeline ([#17](https://github.com/thedavidweng/canvas-cli/issues/17)) ([30f3ace](https://github.com/thedavidweng/canvas-cli/commit/30f3ace50b63415136062ebb137c715c4ff58c3e))
* use reusable workflows from cli-workflow-template ([618db62](https://github.com/thedavidweng/canvas-cli/commit/618db62c71cc2e2eea1a3f8e8595ce3197920617))


### Documentation

* add Canvas LMS logo to README ([cd2c76c](https://github.com/thedavidweng/canvas-cli/commit/cd2c76cb7906adf430b8c484edea550320423b92))
* add Go Report Card badge ([27acf1d](https://github.com/thedavidweng/canvas-cli/commit/27acf1d281a2be8fd456049ae866004880a62f9d))
* add infrastructure links (CI/CD and docs) ([2ae79c6](https://github.com/thedavidweng/canvas-cli/commit/2ae79c6c4f555b4df56afd018afc8eeda882f680))
* add install scripts and streamline README ([011f77a](https://github.com/thedavidweng/canvas-cli/commit/011f77acdfcdd0f5a8a81242f4b32d22a0300ddc))
* add root-level docs for site sync (COMMANDS.md, JSON_SCHEMA.md, CONTEXT.md) ([9f5cb8f](https://github.com/thedavidweng/canvas-cli/commit/9f5cb8f4393bfbd7575f526416e91e132717e6b1))
* add verification checklist to AGENTS.md ([01c427b](https://github.com/thedavidweng/canvas-cli/commit/01c427bbf022b60e9e9b256f6e763e3c79bd389a))
* remove stale plan docs and clean up command reference ([7487835](https://github.com/thedavidweng/canvas-cli/commit/748783531cd0836b331e3088facc77dec33390ee))
* standardize README badges ([32972d2](https://github.com/thedavidweng/canvas-cli/commit/32972d23b3e073b42ba73b03a3e9cde7fb6e75c8))

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
