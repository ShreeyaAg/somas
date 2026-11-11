# Security and deployment model

## Group isolation

The operator configures `SOMAS_GROUPS` as comma-separated `group=token` pairs. Group and token must be nonempty; tokens must be unique and at least 16 characters. Use cryptographically random tokens and HTTPS. Group authentication compares SHA-256 token digests in constant time. Tokens are never accepted in query strings or embedded in dashboard HTML. Authorization headers are required for all API calls, including history, exports and defaults. No wildcard CORS is enabled. A malicious site cannot invoke authenticated controls without the token; the app uses no ambient auth cookies.

Every run and memory query is scoped by the authenticated owner. Clients cannot submit an owner field. Unowned run IDs produce the same 404 response as missing IDs. All authenticated group members have equal privileges over that group's runs. There is no per-person role system or web operator account. Run IDs are random 96-bit identifiers, but randomness is not the authorization check. Cross-group metrics remain private; share selected exported results manually.

## Agent permissions

Agent strategies receive cloned observations and a private memory capability without an owner selector. Peer observation types exclude private resources and memories. Institution summaries are separate from private storage. Typed messages have engine-stamped sender/round/institution metadata, same-institution routing and bounded payloads. Generic framework direct peer lookup/delivery is disabled by the supplied facade. Dashboard responses contain the group's own full physical state and public governance, but no private memories or inboxes. This administrative dashboard view is intentionally broader than an agent's observation.

Only reviewed strategy implementations are compiled. The interface is not an OS sandbox: approved code can still import operating system packages, retain capabilities, loop indefinitely or crash the process. Runtime panic recovery is an availability aid, not protection from hostile code. No untrusted source is uploaded, built or executed by the web server. Review/build procedures are in AGENT_DEVELOPMENT.md.

## Reliability boundaries

The server serializes state mutation and database saves, uses SQLite WAL and parameterized SQL, enforces configuration limits, limits request bodies to 64 KiB, bounds message traffic, rejects invalid/non-finite actions and has HTTP timeouts. Browser content is escaped and a same-origin Content Security Policy disallows inline scripts and framing. Static assets embed into the Go binary. Tokens reside only in browser memory. No private memory download endpoint exists.

A strategy panic pauses the live run with an error; reset is required before play. CPU loops, allocation attacks, `os.Exit`, filesystem/network side effects and malicious dependencies are not contained. There is no distributed scheduler, durable work queue, per-group CPU accounting, disk quota or automatic retention policy. Use modest experiments, monitor disk usage and maintain offline backups. Authenticated high-volume requests can still exhaust resources; restrict cohort access and add proxy rate limits for internet hosting.

After restart, saved simulations are archived/read-only replay; carried memory is reusable by new runs. One unfinished live run per group/memory namespace prevents lost updates. A completed run cannot reset its shared namespace while another unfinished run uses it. Only one server process should use a database. Failed database writes pause the run; in-memory state may have advanced beyond the last durable snapshot, which remains the replay authority after restart.

## Operator checklist

- Bind loopback by default; expose through TLS and university network controls.
- Store group credentials outside the repository; rotate if disclosed.
- Run as a non-root service account with a private persistent data directory.
- Do not give build jobs production credentials or database mounts.
- Back up with SQLite's backup API; protect backups like the live database.
- Review dependencies and strategy source before building/deploying.
