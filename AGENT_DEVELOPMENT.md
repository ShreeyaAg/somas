# Implementing student agents

Strategies are compiled Go packages, registered at application startup. They return decisions; they do not implement resource accounting. The public contract is in `pkg/game/types.go`, with compiling baselines in `pkg/agents` and a template in `examples/custom`.

## Integration without engine changes

1. Copy `examples/custom` to `teams/yourteam`, rename the package/type/strategy, and implement methods you need. Embed `*game.Base` for default pass/no-op behavior and university framework plumbing.
2. Register the factory in the command's composition root, `cmd/somas/main.go`, after `registry := agents.Registry()`. No files in `pkg/game` need changes:

```go
import "somas/examples/custom" // replace with your package
// ... inside run(), immediately after registry := agents.Registry():
custom.Register(registry)
```

3. Test and run:

```sh
go test ./examples/custom ./pkg/game
go test -race ./...
go vet ./...
go run ./cmd/somas -batch -strategies student -seed 42 -rounds 30 -out student.json
# Compare the same seed/population/config against -strategies random.
```

The included `TestRegisteredStudentRuns` actually registers and runs the template. With the registration line above, `student` also appears in the dashboard strategy selector. Rebuild/restart to activate new source; there is no dynamic Go plugin dependency.

## Minimal decisions

```go
func (*Student) Act(o game.Observation, m game.Memory, rng *rand.Rand) game.Decision {
    lastPartner := m.Get("last_partner")
    _ = lastPartner // use to inform your policy
    m.Set("last_institution", fmt.Sprint(o.Self.Institution))
    return game.Decision{
        Harvest: min(1, o.Institution.Rules.Quota),
        Eat: 2,
        Contribute: o.Config.MinContribution,
    }
}
```

`Act` may be called several times per round: once per harvest selection and once for eating/provisioning. It is **not** an exactly-once callback. The engine uses only the relevant fields in each phase, revalidates them and owns every resource mutation. Avoid counting `Act` calls as rounds; store the last observed round in memory when updating per-round learning. No direct authoritative state pointers are supplied.

`Observation.Self` contains own resources, own last actions, own utility, strategy and membership. `Peers` contains IDs, membership, alive status and optionally contribution identities. It deliberately excludes peer brain, body, stock, utility and private memory. Institutions expose members, CPR, negotiated rules and aggregate shared information. Config, institutions, maps and slices are copied. `Inbox` is recipient-scoped. Current inbox messages are available until the next communication phase replaces them; deduplicate on `(Round, From, Type, Content)` when storing interactions.

Use only the passed `rng` for stochastic decisions and iterate maps in sorted order. Do not use global random functions, wall-clock time or random UUIDs for policy decisions. Stable string IDs (`o.Self.ID`) are the experiment identities and private-memory slots. The Base agent UUID is also derived from the seed and slot for framework registration. Strategy object fields persist only while the process lives; use `Memory` for durable state.

## Memory and reputation

`m.Get(key)` and `m.Set(key,value)` are bound to the caller's own slot. No owner, group or target-agent selector is accepted. Keys are 1–128 bytes, values at most 8192 bytes, at most 256 keys per agent. Oversized writes are ignored. Encode strategy-specific structs as JSON strings if desired; store trust under keys like `trust:agent-002`. Reputation claims received from peers are claims, not authenticated assessments. There is no automatic trust calculation or hidden-peer resource oracle.

The web server saves memories at every phase. They survive rounds. Cross-run memory is opt-in through `carry_memory` plus an operator-chosen namespace; memory is grouped by owner and namespace and then agent slot. Reuse a namespace only with compatible slot/strategy assignments and schema versions. Different groups and fresh runs have separate scopes. Headless trials always start empty. Reset run controls restore that run's initial memory; resetting a carried namespace removes its stored cross-run state. Private memories are never included in dashboard, history or export APIs.

Shared information is separate: `o.Institution.Shared` is a read-only engine summary of contributors and last CTK, not an arbitrary shared memory write channel. Broadcast typed messages for strategy-controlled shared facts. Institutional state resets on dissolution; snapshots preserve its history.

## Politics, association and communication

Override `ChooseInstitution` to return a target institution ID at formation/reformation; return `-1` for seeded balanced assignment. Peers' previous memberships and your private memory can guide this choice. No mid-round membership action is enabled in the MVP.

`Propose` returns nil to pass or `&game.Proposal{Key:"quota", Value:1.2}` to negotiate. Other supported keys are `order` (`Order:"fixed"`, `"random"`, `"rotating"`) and `vote_threshold` (`Value:.5` through <1). `Vote` returns true for acceptance. Policies may deliberately violate quotas; the engine records actual breaches. Physical bounds always apply.

`Communicate` returns typed `game.Message` values. Supported routing: a current same-institution agent ID or `"institution"` for broadcast. Message types are extensible strings (e.g. `cooperation`, `reputation`, `voting_intention`, `trade_offer`). Trade offers are informational; no resource transfer executes. Sender, round and institution are stamped by the engine, never trusted from strategy input. Limit: 8 outgoing messages per agent per phase, 64 inbox messages, 64-byte type and 2048-byte content. Invalid routes and messages are dropped; only delivered messages cost brain. Delivery is deterministic and bounded.

The university's generic message methods exist through the embedded v2 interface, but the supplied facade deliberately disables direct generic delivery and peer implementation lookup. Use `Communicate`, not `SendMessage`, `BroadcastMessage` or `AccessAgentByID`; those cannot bypass observation or routing permissions. No `SignalMessagingComplete` call is needed. `RunWithPlatform()` exercises the actual v2 `BaseServer.Start()` callbacks; web stepping uses the same phase functions without the base server's blocking listening loop. This is v2 compatibility, not drop-in compatibility with other groups' game-specific interfaces.

## Safe submission and activation

There is **no HTTP code-upload or execution endpoint**. The MVP uses an operator-approved source integration process:

1. A group submits a source-only branch/archive to the course maintainers (outside the running server). Retain it in an offline staging checkout, never under the web root.
2. Review source and dependency changes. Reject filesystem/network access, process execution, goroutines, global mutable state, unbounded loops and other server-side effects. Interface validation is not a security sandbox.
3. In a disposable environment without server credentials or the production database, register it in the composition root and run compilation, `go vet`, tests and fixed-seed batch comparisons. Build/test commands themselves can execute code; do not run unreviewed tests on the shared host.
4. Approve a commit and build an immutable binary. Back up SQLite, stop the shared service, deploy that binary and restart with its existing data directory and group config.

Only approved code becomes selectable. Rollback uses the previous binary and a compatible database. An actual untrusted runner would require process/container isolation, CPU/memory/time limits and a narrow RPC protocol; it is deliberately deferred. Go interfaces and panic recovery alone do not isolate arbitrary code.
