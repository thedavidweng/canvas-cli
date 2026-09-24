# 0006: E2E-First Testing

Status: Accepted

Context: Help-text, flag-default, envelope-shape, browser-list, and mock self-tests mirrored constants. Prompt string tests asserted interactive wording. Coverage patch target drove test creation. golden_test carries exit-code, safety, redaction, and stdout-purity contracts.

Decision: golden_test plus FakeCanvas integration is the E2E substitute with envelope, exit-code, read-only, dry-run, confirm, redaction, and stdout-purity artifacts. Isolated tests remain only for config permissions and precedence, audit hashing, rate-limit math, pagination walks, upload and download failures, error taxonomy, cookie matching, and safety gates with exact codes. Coverage informational: patch 80 to 50, threshold 2 to 5.

Consequences: Deleted 13 files and 100 prompt, mirror, and echo tests. PromptCookieAuth success flow retained. roundTripFunc removed with setter tests.
