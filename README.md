# canvas-cli

[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![CI](https://github.com/thedavidweng/canvas-cli/actions/workflows/ci.yml/badge.svg)](https://github.com/thedavidweng/canvas-cli/actions/workflows/ci.yml)

Agent-friendly CLI for Canvas LMS. Stable JSON output, predictable exit codes, safe mutation gates, raw API escape hatch.

The repository is `canvas-cli`. The installed binary is `canvas`.

## Install

```bash
go install github.com/thedavidweng/canvas-cli/cmd/canvas@latest
```

Or via Homebrew:

```bash
brew tap thedavidweng/tap
brew install canvas
```

## Auth

For authentication, users should use one of the following methods (the `--token` flag is not supported to avoid tokens in shell history):

```bash
# Securely pass token via stdin
echo "..." | canvas auth login --base-url https://school.instructure.com --token-stdin

# Or use an environment variable
export CANVAS_TOKEN="..."
canvas auth login --base-url https://school.instructure.com --token-env CANVAS_TOKEN
```

Manual tokens are suitable for personal/local automation. Multi-user applications should use OAuth2.

## Quickstart

```bash
canvas courses list
canvas modules list --course 123
canvas assignments list --course 123
canvas files list --course 123
canvas courses export-context --course 123 --out course-context.json
```

## JSON for agents

```bash
canvas assignments list --course 123 --bucket unsubmitted --json
```

JSON uses a stable envelope:

```json
{
  "ok": true,
  "data": [],
  "meta": {
    "schema_version": "2026-06-12",
    "command": "assignments.list"
  }
}
```

## Safety

Remote mutations support dry-run and confirmation gates:

```bash
canvas assignments submit --course 123 456 --file paper.pdf --dry-run
canvas assignments submit --course 123 456 --file paper.pdf --confirm
```

High-risk teaching team operations require explicit confirmation and write local audit logs.

## Raw API

```bash
canvas api get /api/v1/courses
canvas api get /api/v1/courses/123/assignments --paginate --json
```

## Command Reference

### Foundation

| Command | Status | Description |
|---|---|---|
| `canvas version` | Implemented | Print CLI version |
| `canvas doctor` | Implemented | Validate CLI environment and configuration |
| `canvas completion bash\|zsh\|fish\|powershell` | Implemented | Generate shell completion scripts |
| `canvas auth status` | Implemented | Show current authentication status |
| `canvas auth test` | Implemented | Test authentication by calling the API |
| `canvas auth login` | Implemented | Save authentication credentials |
| `canvas auth logout` | Implemented | Remove token from current profile |
| `canvas auth profiles` | Implemented | List all configured profiles |
| `canvas auth use PROFILE` | Implemented | Switch the active profile |
| `canvas me get` | Implemented | Get current user information |
| `canvas me activity` | Implemented | Show recent activity |
| `canvas me todo` | Implemented | Show todo items |
| `canvas me upcoming` | Implemented | Show upcoming events |
| `canvas api get PATH` | Implemented | Execute a GET request to the Canvas API |
| `canvas api post PATH` | Implemented | Execute a POST request |
| `canvas api put PATH` | Implemented | Execute a PUT request |
| `canvas api delete PATH` | Implemented | Execute a DELETE request |

### Student Read

| Command | Status | Description |
|---|---|---|
| `canvas courses list` | Implemented | List courses |
| `canvas courses get COURSE_ID` | Implemented | Get a course by ID |
| `canvas courses tabs --course COURSE_ID` | Implemented | List course tabs |
| `canvas courses export-context --course COURSE_ID --out context.json` | Implemented | Export course context to JSON |
| `canvas modules list --course COURSE_ID` | Implemented | List modules for a course |
| `canvas modules get --course COURSE_ID MODULE_ID` | Implemented | Get a module by ID |
| `canvas modules items --course COURSE_ID --module MODULE_ID` | Implemented | List items in a module |
| `canvas modules item --course COURSE_ID --module MODULE_ID ITEM_ID` | Implemented | Get a module item |
| `canvas assignments list --course COURSE_ID` | Implemented | List assignments for a course |
| `canvas assignments get --course COURSE_ID ASSIGNMENT_ID` | Implemented | Get an assignment by ID |
| `canvas assignments groups --course COURSE_ID` | Implemented | List assignment groups |
| `canvas announcements list --course COURSE_ID` | Implemented | List announcements for a course |
| `canvas announcements get ANNOUNCEMENT_ID` | Implemented | Get an announcement |
| `canvas discussions list --course COURSE_ID` | Implemented | List discussion topics |
| `canvas discussions get --course COURSE_ID DISCUSSION_ID` | Implemented | Get a single discussion topic |
| `canvas discussions entries --course COURSE_ID DISCUSSION_ID` | Implemented | List entries (replies) for a discussion |
| `canvas files list --course COURSE_ID` | Implemented | List files in a course |
| `canvas files get FILE_ID` | Implemented | Get a file |
| `canvas files download FILE_ID --out PATH` | Implemented | Download a file to a local path |
| `canvas files download-course --course COURSE_ID --out DIR` | Implemented | Download course files |
| `canvas pages list --course COURSE_ID` | Implemented | List wiki pages |
| `canvas pages get --course COURSE_ID PAGE_URL` | Implemented | Get a single wiki page |
| `canvas submissions get --course COURSE_ID --assignment ASSIGNMENT_ID --user USER_ID` | Implemented | Get a submission for a specific user |

### Student Actions

| Command | Status | Description |
|---|---|---|
| `canvas assignments submit --course COURSE_ID ASSIGNMENT_ID --text BODY` | Implemented | Submit assignment with text |
| `canvas assignments submit --course COURSE_ID ASSIGNMENT_ID --file PATH` | Implemented | Submit assignment with file |
| `canvas assignments submit --course COURSE_ID ASSIGNMENT_ID --url URL` | Implemented | Submit assignment with URL |
| `canvas discussions reply --course COURSE_ID --did DISCUSSION_ID --message BODY` | Implemented | Reply to a discussion topic |
| `canvas discussions reply-entry --course COURSE_ID --did DISCUSSION_ID --entry ENTRY_ID --message BODY` | Implemented | Reply to a discussion entry |
| `canvas inbox list` | Implemented | List inbox conversations |
| `canvas inbox get CONVERSATION_ID` | Implemented | Get a single conversation |
| `canvas inbox send --to USER_ID --subject SUBJECT --body BODY` | Implemented | Send a new inbox message |
| `canvas inbox reply CONVERSATION_ID --body BODY` | Implemented | Reply to an inbox conversation |
| `canvas inbox archive CONVERSATION_ID` | Implemented | Archive an inbox conversation |
| `canvas submissions comment --course COURSE_ID --assignment ASSIGNMENT_ID --user USER_ID --comment TEXT` | Implemented | Comment on a submission |

### Teaching Team Read

| Command | Status | Description |
|---|---|---|
| `canvas enrollments list --course COURSE_ID` | Implemented | List enrollments for a course |
| `canvas sections list --course COURSE_ID` | Implemented | List sections for a course |
| `canvas users list --course COURSE_ID` | Implemented | List users in a course |
| `canvas rubrics list --course COURSE_ID` | Implemented | List rubrics for a course |
| `canvas submissions list --course COURSE_ID --assignment ASSIGNMENT_ID` | Implemented | List submissions for an assignment |
| `canvas submissions download --course COURSE_ID --assignment ASSIGNMENT_ID --out DIR` | Implemented | Download submission files |

### Teaching Team Actions

| Command | Status | Description |
|---|---|---|
| `canvas modules publish --course COURSE_ID MODULE_ID` | Implemented | Publish a module |
| `canvas modules unpublish --course COURSE_ID MODULE_ID` | Implemented | Unpublish a module |
| `canvas assignments update --course COURSE_ID ASSIGNMENT_ID --due-at TIME` | Implemented | Update an assignment |
| `canvas announcements create --course COURSE_ID --title TITLE --body-file FILE` | Implemented | Create an announcement |
| `canvas discussions create --course COURSE_ID --title TITLE --body-file BODY.md` | Implemented | Create a discussion |
| `canvas pages update --course COURSE_ID PAGE_URL --body-file FILE` | Implemented | Update a wiki page |
| `canvas files upload --course COURSE_ID --file PATH --folder FOLDER_ID` | Implemented | Upload a file |
| `canvas grade set --course COURSE_ID --assignment ASSIGNMENT_ID --user USER_ID --score SCORE` | Implemented | Set a grade |
| `canvas grade comment --course COURSE_ID --assignment ASSIGNMENT_ID --user USER_ID --comment TEXT` | Implemented | Add a comment to a submission |
| `canvas grade rubric --course COURSE_ID --assignment ASSIGNMENT_ID --user USER_ID --rubric-json FILE` | Implemented | Assess with rubric |
| `canvas grade import --course COURSE_ID --assignment ASSIGNMENT_ID --csv FILE` | Implemented | Import grades from CSV |

## Documentation

- [COMMANDS.md](COMMANDS.md) -- full command inventory with examples
- [JSON_SCHEMA.md](JSON_SCHEMA.md) -- JSON envelope contract and exit codes
- [SECURITY.md](SECURITY.md) -- security rules and audit logging
- [CONTRIBUTING.md](CONTRIBUTING.md) -- contributor guidelines
- [CHANGELOG.md](CHANGELOG.md) -- version history
- [docs/product-brief.md](docs/product-brief.md) -- product positioning and scope
- [docs/architecture.md](docs/architecture.md) -- package layout and design
- [docs/command-spec.md](docs/command-spec.md) -- command syntax reference
- [docs/json-contract.md](docs/json-contract.md) -- JSON envelope specification
- [docs/safety-model.md](docs/safety-model.md) -- safety levels and audit logging
- [docs/auth.md](docs/auth.md) -- authentication and configuration
- [docs/pagination-rate-limits.md](docs/pagination-rate-limits.md) -- pagination and rate limiting
- [docs/api-surface.md](docs/api-surface.md) -- Canvas API endpoint mapping
- [docs/raw-api.md](docs/raw-api.md) -- raw API passthrough commands
- [docs/audit-log.md](docs/audit-log.md) -- audit event schema and redaction
- [docs/export-context-spec.md](docs/export-context-spec.md) -- course export specification
- [docs/file-upload-download.md](docs/file-upload-download.md) -- file upload and download flows
- [docs/student-workflows.md](docs/student-workflows.md) -- student workflow guide
- [docs/teaching-team-workflows.md](docs/teaching-team-workflows.md) -- teaching team workflow guide
- [examples/config.example.yaml](examples/config.example.yaml) -- example configuration
- [examples/agent-workflows.md](examples/agent-workflows.md) -- agent workflow examples

## License

Apache License 2.0. See [LICENSE](LICENSE).
