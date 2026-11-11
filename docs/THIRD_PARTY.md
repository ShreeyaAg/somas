# Third-party dependencies

Direct Go dependencies are pinned in `go.mod` and verified by `go.sum`:

- `github.com/MattSScott/basePlatformSOMAS/v2` v2.1.0 — university agent/server framework (MIT; retain upstream licence if vendoring).
- `github.com/google/uuid` v1.6.0 — UUIDs (BSD-3-Clause).
- `modernc.org/sqlite` v1.34.5 — pure Go SQLite driver (BSD-3-Clause; includes SQLite/public-domain components and dependencies).

Transitive module versions/checksums are recorded by Go. Use `go list -m all` for the complete dependency inventory. The project does not vendor code or bundle Moonis artwork or student-source archives.
