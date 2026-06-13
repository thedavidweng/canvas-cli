# Architecture Deepening Plan — canvas-cli

All work follows TDD: vertical slices, one test → one implementation → repeat.

## Phase 1: Quick Wins (exit codes + schema version)
- Extract hardcoded `"2026-06-12"` literals → reference `output.SchemaVersion`
- Unify `SafetyError.ExitCode` with `output.CodeSafetyBlocked`
- Wire `audit.HashBody()` into CLI consumers (replace inline sha256)
- Delete dead `DoWithRetry` standalone function
- Unify `RateLimitMeta` / `RateLimit` into one type

## Phase 2: Config Unification
- Remove `os.Getenv("CANVAS_READ_ONLY")` from `safety.Policy.Check()`
- Surface env var parse errors instead of silent fallback
- Consolidate `ResolvedConfig` fields with `canvas.Profile`/`canvas.Config`

## Phase 3: Canvas Client Interface
- Define `CanvasAPI` interface in `internal/canvas/`
- Adapt existing `*Client` to satisfy it
- Create `MockCanvasAPI` for CLI tests (lightweight, no HTTP server)
- Wire commands to accept interface via context or constructor

## Phase 4: Deepen Request Module
- Generic `List[T]` / `Get[T]` / `Mutate[T]` helpers
- Route conversations.go through `Request()`
- Route upload.go through `Client` methods
- Fix pagination re-marshal waste

## Phase 5: Command Handler Pipeline
- Extract `getClientFromContext()` helper
- Extract `writeJSONorHuman()` helper
- Move cross-cutting helpers out of discussions.go
- Normalize dry-run preview format

## Phase 6: Resource Wrappers
- Replace mechanical resource files with generic helpers
- One-liner registrations for simple List/Get resources
