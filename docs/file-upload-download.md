# File Upload and Download

Canvas workflows require robust file handling for course resources, assignment submissions, submission comments, and bulk grading workflows.

## Downloads

Commands:

```bash
canvas files download FILE_ID --out PATH
canvas files download-course --course COURSE_ID --out DIR
canvas submissions download --course COURSE_ID --assignment ASSIGNMENT_ID --out DIR
```

Rules:

- never overwrite existing files unless `--overwrite`
- write temp file then atomic rename
- preserve original filename when safe
- sanitize path components
- emit manifest for bulk operations
- progress goes to stderr
- JSON result includes file metadata and local path

## Uploads

Commands:

```bash
canvas assignments submit --course COURSE_ID ASSIGNMENT_ID --file PATH
canvas files upload --course COURSE_ID --folder FOLDER_ID --file PATH
canvas submissions comment --course COURSE_ID --assignment ASSIGNMENT_ID --user USER_ID --file PATH
```

Rules:

- validate local file path
- detect size and content type
- show dry-run summary before upload when requested
- include uploaded file IDs in JSON result
- record audit event for remote mutations

The Canvas file upload process requires implementing the full 3-step flow via a shared helper:
1. **Step 1: Notify Canvas.** `POST` to the appropriate `/api/v1/...` file endpoint (e.g. `/api/v1/courses/{id}/files` or `/api/v1/courses/{id}/assignments/{id}/submissions/{user_id}/files`) with the file's `name`, `size`, and `content_type`. This returns an `upload_url` and `upload_params`.
2. **Step 2: Upload file.** `POST` to the returned `upload_url` using multipart/form-data. Include all `upload_params` as form fields, and attach the file content.
3. **Step 3: Follow redirect / finalize.** Canvas responds to Step 2 with either a `201 Created` containing the final file info, or a `3xx Redirect`. If a redirect is returned, issue a `GET` (or `POST`, depending on the endpoint) to the `Location` header to finalize the upload and receive the final file JSON.

## Bulk submission download manifest

```json
{
  "course_id": "123",
  "assignment_id": "456",
  "generated_at": "2026-06-12T19:20:00Z",
  "items": [
    {
      "user_id": "111",
      "submission_id": "222",
      "attempt": 1,
      "file_id": "333",
      "filename": "paper.pdf",
      "local_path": "assignment-456/user_111/submission-222_paper.pdf",
      "status": "downloaded"
    }
  ]
}
```

### Platform notes

- **Path sanitization:** Student names may contain characters invalid in filenames (e.g., `/`, `\`, `:`, `*`). Sanitize by replacing these with `_`.
- **Cross-platform paths:** Config and audit paths are resolved via `os.UserConfigDir()` / `os.UserStateDir()`, which return OS-appropriate directories on Linux, macOS, and Windows automatically.
