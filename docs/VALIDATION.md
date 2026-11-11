# Validation record

Executed in the supplied Linux workspace on 11 November 2026 using Go **1.25.4**, the pinned **basePlatformSOMAS/v2 v2.1.0**, pure-Go SQLite and system Chromium driven by Python Playwright.

## Automated checks

- `go test ./...` — passed.
- `go test -race ./...` — passed; no detected data races.
- `go vet ./...` — passed.
- `go test -cover ./...` — passed; game package statement coverage 89.3%. CLI and browser behavior were checked separately; this is not a claim of full application coverage.
- Built the actual server binary and ran it on local HTTP ports against file-backed SQLite.
- Registered the included student package in a separate copy's CLI composition root, then ran an actual `-batch -strategies student -rounds 3` invocation successfully. No engine source was changed.
- Used SQLite's online backup API successfully on a running file-backed server database.

Game tests cover productivity at/beyond five contributors, CTK tier boundaries, deterministic entire phase states, physical resource bounds, NaN/infinite/negative actions, observation/snapshot alias resistance, advisory quota violations, zero-brain survival/political restriction, majority adoption, CTK reset, input configuration rejection, university lifecycle equivalence, restricted framework peer access, private contribution observations, sender stamping, direct/broadcast message privacy and body-death/revival boundaries.

Storage/API tests cover file-backed database reopening, saved history, owner-scoped memory/history, memory reset, unauthenticated requests, cross-group run/history/export denial, controls and exports, default dashboard serving, exclusive carried-memory namespaces, application restart with opted-in memory reuse, independent other-group memory and fresh-run memory isolation.

## Browser checks

Run `scripts/browser_smoke.py` with Playwright installed and a server already listening:

```sh
SOMAS_TEST_URL=http://127.0.0.1:8080 \
SOMAS_TEST_TOKEN='your-configured-test-group-token' \
python3 scripts/browser_smoke.py
```

The script creates a short test experiment, so use a test group/database. It verified login, new experiment creation, two gardens, political phase stepping, agent inspection, speed changes, play to completion, timeline replay, return to live, reset, and a 390px responsive viewport without horizontal page overflow. Desktop and mobile screenshots were captured. No JavaScript exceptions or browser console errors remained in the final check. Screenshots are written under `work/`; an example desktop capture is included as `docs/dashboard.png`.

Issues found and fixed during development included narrow-screen grid overflow and an absent favicon. The strict CSP required locator-based browser assertions instead of string-evaluated polling. The workspace's `/usr/bin/go` was an unrelated executable, so an official Go toolchain was installed under scratch storage; this tooling is not included in the project or required on a machine with Go installed. Its invalid enclosing `.git` directory required `-buildvcs=false` locally; a normal project checkout does not.

## Small baseline experiment

Defaults, **10 agents, 2 institutions, 20 rounds, 1 cycle**, seeds **42,43,44**, homogeneous populations. Institutional reformation remains at its default every 10 rounds, with fresh patches; this affects interpretation. See `baseline-comparison.csv` for each seed's final metrics.

| Strategy | Mean individual final utility, averaged across seeds | Mean final CPR/capacity | Final survivors |
|---|---:|---:|---:|
| Random | 19.3623 | 0.0912 | 10/10 for all three |
| Cooperative | 26.6913 | 1.0000 | 10/10 for all three |

Reproduction commands:

```sh
go run ./cmd/somas -batch -strategies random -rounds 20 -repeat 3 -seed 42 -out random.json
go run ./cmd/somas -batch -strategies cooperative -rounds 20 -repeat 3 -seed 42 -out cooperative.json
```

A repeated random batch produced **byte-identical JSON** using the same binary/config/seeds. These three seeds demonstrate a usable comparison workflow; they do not establish statistical superiority, generality or calibrated game balance. For coursework conclusions, sweep more seeds, strategy mixes, costs, residual-CPR conditions and institution lifetimes. Turn off short-period patch resetting when studying uninterrupted resource sustainability.

## Known limits

No untrusted code execution/upload, no resumable process checkpoint, no independent per-agent execution timeout, no sanctions/auditing/trading/reproduction and no automatic cross-group publication. Web snapshots persist after each phase; an abrupt process failure may lose only the currently executing uncommitted phase. Active strategy object fields and RNG state are not serialized. The included tests do not prove isolation against malicious compiled Go code, nor throughput at the configured maximum population/round limits. See SECURITY.md and GAME_RULES.md for the exact boundaries.
