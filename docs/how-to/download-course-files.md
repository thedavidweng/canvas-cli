# Download course files

Use this to pull lecture slides, readings, and assignment sheets onto your machine — one file or the whole course — with a manifest you can script against.

> Outputs in the guides were captured from real runs of `canvas-cli` against a Canvas instance for a fictional student, *Alex Rivera*. IDs, names, and sizes will differ in your account.

## Prerequisites

A working connection (`canvas auth test` succeeds — see [Getting started](../tutorials/getting-started.md)) and a course ID (`canvas courses list`).

## 1. See what's available

```shell
canvas files list --course 101
```

```text
301	syllabus.pdf	241152	application/pdf
302	lecture-07-recursion.pptx	1846272	application/vnd.openxmlformats-officedocument.presentationml.presentation
303	problem-set-4.pdf	88413	application/pdf
```

Columns: file ID, name, size in bytes, content type.

## 2. Download one file

Use the file ID from the first column:

```shell
canvas files download 303 --out problem-set-4.pdf
```

```text
Downloaded file 303 to problem-set-4.pdf
```

`--out` is required and lands wherever you point it. Add `--no-overwrite` to fail instead of replacing an existing local copy.

## 3. Download everything at once

```shell
canvas files download-course --course 101 --out course-files
```

```text
Downloaded 3/3 files
Manifest: course-files/manifest.json
```

Alongside the files you get two manifests for scripting:

```shell
ls -l course-files
```

```text
total 40
-rw-r--r--@ 1 david  wheel   20 Aug 21 05:51 lecture-07-recursion.pptx
-rw-r--r--@ 1 david  wheel  813 Aug 21 05:51 manifest.json
-rw-r--r--@ 1 david  wheel  667 Aug 21 05:51 manifest.ndjson
-rw-r--r--@ 1 david  wheel   31 Aug 21 05:51 problem-set-4.pdf
-rw-r--r--@ 1 david  wheel   28 Aug 21 05:51 syllabus.pdf
```

`manifest.json` records each file's ID, size, local path, and download status (excerpt — the `problem-set-4.pdf` entry is omitted):

```json
[
  {
    "file_id": "301",
    "filename": "syllabus.pdf",
    "display_name": "syllabus.pdf",
    "content_type": "application/pdf",
    "size": 241152,
    "local_path": "course-files/syllabus.pdf",
    "download_status": "ok"
  },
  {
    "file_id": "302",
    "filename": "lecture-07-recursion.pptx",
    "display_name": "lecture-07-recursion.pptx",
    "content_type": "application/vnd.openxmlformats-officedocument.presentationml.presentation",
    "size": 1846272,
    "local_path": "course-files/lecture-07-recursion.pptx",
    "download_status": "ok"
  },
  ...
]
```

(`manifest.ndjson` holds the same entries, one JSON object per line.) Re-running overwrites local copies by default; add `--no-overwrite` to skip files that already exist.

## Next steps

- [Check what's due](check-what-is-due.md) — pair the reading with the assignment it belongs to
- [Command Reference](../command-spec.md#files) — all files flags
