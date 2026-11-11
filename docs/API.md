# HTTP API

All `/api/` requests require `Authorization: Bearer GROUP_TOKEN`. Request bodies are JSON, unknown fields and trailing JSON are rejected. Responses are JSON unless CSV is requested. No owner parameter is accepted. Static dashboard assets are public but contain no run data or credentials.

| Method | Endpoint | Behavior |
|---|---|---|
| GET | `/api/defaults` | Full default game configuration |
| GET | `/api/strategies` | Registered compiled strategy names |
| GET | `/api/runs` | Authenticated group's persisted runs and live flags |
| POST | `/api/runs` | Create independent run; body `{"config":{...full config...},"memory_namespace":""}` |
| GET | `/api/runs/{id}` | Config, current state, error and playing flag |
| POST | `/api/runs/{id}/control` | `{"action":"play"}` / pause / step / reset / archive; speed also needs `"speed_ms":500` |
| GET | `/api/runs/{id}/history` | Ordered phase snapshots, including formation and completion |
| GET | `/api/runs/{id}/export.json` | Config and history |
| GET | `/api/runs/{id}/export.csv` | Phase metrics in CSV (including cumulative moves) |
| POST | `/api/memory/reset` | `{"namespace":"learning"}` deletes own carried memory; blocked during unfinished use |

HTTP run creation requires a complete config: fetch `/api/defaults` and modify it. The CLI instead merges partial config files over defaults. `carry_memory:true` requires an explicit namespace, length 1–64. `speed_ms` is 50–10000 milliseconds **per phase**, not per round. `step` pauses automatic play and advances one phase. `reset` restores the original config, RNG seed and memory loaded at run creation; history appends a new formation state. It does not delete old snapshots. `archive` evicts a completed live engine, retaining database history; archived controls return 409. Server restart archives every run.

Status codes: 401 invalid/missing token; 404 missing/unowned run or endpoint; 400 invalid configuration/action/input; 409 archived run or memory scope conflict; 500 persistence/strategy failure. No server-side code submission or private-memory read API exists.

Example:

```sh
export TOKEN='your-token'
curl -fsS -H "Authorization: Bearer $TOKEN" http://localhost:8080/api/defaults
# Build a JSON body {"config": <defaults with edits>} and POST it to /api/runs.
curl -fsS -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d '{"action":"step"}' http://localhost:8080/api/runs/RUN_ID/control
```

Spatial requests may set `spatial`, `auto_patches`, `agents_per_patch`, minimum `grid_width`/`grid_height`, movement costs and `harvest_access`. When automatic sizing is enabled, the manual institution count is ignored. The returned run config is normalized: patch count and dimensions may differ from the request. Snapshots include `world` geometry, each player's `position` and cumulative `moves`; peer positions are public. Schema additions need no SQLite migration. See SPATIAL.md.

Snapshots may include optional `players[].activity`: per-round applied sunlight/clouds, resource costs and action totals. It is operator-scoped, not part of public peer observations. Old histories omit the field; clients must treat it as unavailable. JSON history/export preserves recorded ledgers. See INTERFACE.md.
