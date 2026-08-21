# Collect student submissions

Use this to download every submission attachment for an assignment into an organized local folder, with a manifest that records who submitted what.

> Outputs in the guides were captured from real runs of `canvas-cli` against a Canvas instance for a fictional teaching team. IDs, names, and sizes will differ in your course.

## Prerequisites

A teacher/TA enrollment in the course (Canvas enforces this — students can't list other students' submissions), and a working connection (`canvas auth test`).

## 1. See who has submitted

```shell
canvas submissions list --course 101 --assignment 202
```

```text
9011	Jamie Park	graded	47.5	2026-08-26T20:12:40Z
9012	Sam Okafor	unsubmitted
9013	Lena Fischer	graded	42	2026-08-29T09:04:55Z
```

Columns: submission ID, student, state, score, submitted-at. Sam hasn't submitted; Lena's timestamp is after the due date (her `late` flag is set — check with `--json`).

## 2. Download all attachments

```shell
canvas submissions download --course 101 --assignment 202 --out submissions
```

```text
Downloaded 2/2 files
Manifest: submissions/202/manifest.json
```

This command only writes to your local disk — it changes nothing in Canvas, so there's no confirm gate. Two of three students had attachments; the unsubmitted student contributes no files.

Each student gets their own directory named `Last, First_USERID`:

```shell
find submissions -type f | sort
```

```text
submissions/202/Fischer, Lena_61004/9013_ps4-lena-fischer.pdf
submissions/202/manifest.json
submissions/202/manifest.ndjson
submissions/202/Park, Jamie_61002/9011_ps4-jamie-park.pdf
```

## 3. Work from the manifest

`manifest.json` is the machine-readable record of the download — pair it with grading scripts or an agent:

```json
[
  {
    "submission_id": "9011",
    "user_id": "61002",
    "user_name": "Jamie Park",
    "sortable_name": "Park, Jamie",
    "attachment_id": "9501",
    "filename": "ps4-jamie-park.pdf",
    "original_url": "http://127.0.0.1:8787/file-content/9501",
    "local_path": "submissions/202/Park, Jamie_61002/9011_ps4-jamie-park.pdf",
    "size": 121344,
    "download_status": "ok"
  },
  {
    "submission_id": "9013",
    "user_id": "61004",
    "user_name": "Lena Fischer",
    "sortable_name": "Fischer, Lena",
    "attachment_id": "9502",
    "filename": "ps4-lena-fischer.pdf",
    "original_url": "http://127.0.0.1:8787/file-content/9502",
    "local_path": "submissions/202/Fischer, Lena_61004/9013_ps4-lena-fischer.pdf",
    "size": 109568,
    "download_status": "ok"
  }
]
```

`manifest.ndjson` holds the same entries, one JSON object per line — handy for `jq` pipelines. Re-runs overwrite local files by default; add `--no-overwrite` to skip attachments you already downloaded.

## Next steps

- [Enter grades](enter-grades.md) — score what you collected
- [Safety model](../safety-model.md) — why grading commands behave differently from this one
