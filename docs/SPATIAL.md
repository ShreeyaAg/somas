# Spatial world and automatic patch sizing

This is an explicit, optional game extension. The supplied PDFs discuss grid movement as a suggestion, without agreed movement costs or access rules. Classic CPR/CTK operation is retained with `spatial:false`; the browser enables the spatial option for new experiments. Existing database histories remain replayable. Histories saved before world coordinates were added display a notice instead of inventing positions.

## Population determines the world

With `auto_patches:true`, the engine derives:

```
institutions = patches = ceil(agents / agents_per_patch)
```

The default target is 5. Examples: 10 agents → 2 patches; 12 agents with target 4 → 3 patches; 100 agents → 20 patches; 200 agents → 40 patches. The manual institutions setting is ignored in automatic mode. Turn automatic mode off to specify the institution/patch count yourself.

Every institution owns one physical patch (currently a 2×2 tile region), its own CPR and negotiated rules. Patches are distinct public geometry objects associated with an institution, rather than positions embedded in strategy code. The current supported relationship is one patch per institution. The engine determines a compact deterministic layout with paths between regions. Grid dimensions are **minimums**, not fixed boundaries: the engine enlarges them until all requested patches fit. No layout edits or engine code changes are needed when population/strategy counts change. The API returns the normalized configuration and authoritative world dimensions.

The target is a sizing heuristic, not a hard membership limit: agents may choose an institution and create uneven membership. Patches are sized from the configured population at experiment creation; they are not deleted when agents die. Reformation keeps the same patch geometry/count, resets resources/AVAs as in the classic game, and places living/revived agents back on their home patch. Agent source registration never creates a patch by itself; the population selected for an experiment determines provisioning.

## Movement

Strategies may implement the optional interface:

```go
type Mover interface {
    Move(Observation, Memory, *rand.Rand) Position
}
```

`Base` supplies a stay-in-place default, so existing agents compile unchanged. `Move` returns an absolute destination tile. It is called once per living agent per round after political negotiation and message delivery, before harvesting. Valid destinations are the current tile or one orthogonally adjacent in-bounds tile. No diagonals, teleports or wrapping. Extreme integer coordinates are rejected before arithmetic. Costs are charged only for an accepted actual move, and only if both resources are affordable. Body zero kills immediately; dead agents do not move or harvest.

Defaults: `move_brain_cost:0.05`, `move_body_cost:0.02`. Zero costs are supported deliberately. Multiple agents may share a tile; marker offsets on the map only help visibility and do not change their physical coordinates. There are no obstacles, collision blocking, fog of war or distance-based communication in this version. Map geometry/CPR and peer positions are public; private memories and resource observations remain restricted.

Physical movement **does not change institutional membership**, voting rights, message routing or CTK destination. An agent walking to patch B while belonging to institution A continues to vote/communicate/provision within A. Institutional reassignment still happens at formation/reformation. This separation lets strategies explore movement without silently bypassing governance.

## Harvesting and patch access

In spatial mode, an agent can harvest only while on a patch tile, and only when allowed by `harvest_access`:

- `members`: only members of the owning institution may harvest there (default).
- `open`: visitors may harvest; their own institution stays unchanged.

Paths contain no vegetables. Harvest queues comprise agents actually present and eligible on each patch, ordered using that patch's AVA. A visitor is subject to the visited patch's voluntary quota; physically possible violations are recorded. All patches harvest before any patch regrows, preventing a same-round second access to already-regrown resources. Eating/provisioning then occur for every living member, including travellers. CTK contributors/free-riders count living provisioning participants, not just harvesters. Movement may therefore reduce brain available for CTK and make travel costly in the social dilemma.

There is no click-to-move administrative override: the map is a visualization/inspection surface; strategy actions and the backend determine locations. Run, pause, step and replay work on persisted positions. Zoom is visual only. Crop opacity and resource bars reflect actual CPR, rather than a separate frontend resource simulation.

## Strategies and configuration

- **random**: seeded random cardinal movement or stay.
- **cooperative**: returns toward/stays on its own patch, while preserving its cooperative resource policy.
- **roaming**: under open access, walks toward the currently richest public patch; under member access, returns home. This is an example spatial policy, not a claim of optimal behavior.

`Observation.World` contains patch geometry, `Self.Position` is own location, and public peer observations include positions. `Observation.Institution` always means membership; `OccupiedPatch` describes the physical patch at the agent's location or is nil on a path. All are copied. A helper `game.StepToward(from,target)` returns one cardinal step; `game.Wander` returns a seeded random valid neighbour/stay. They do not mutate state.

Start a headless spatial experiment:

```sh
go run ./cmd/somas -batch -config configs/spatial.json -rounds 30 -out spatial.json
```

The config file contains partial overrides of existing defaults. In the browser, select **Enable spatial movement**, **Scale patches automatically**, a target population per patch and an access policy. The preview shows the derived patch count before creating the experiment. Top-level controls override matching JSON-editor fields; other parameters, including movement costs and minimum dimensions, stay editable in JSON.

| Field | Default | Notes |
|---|---:|---|
| spatial | false in core/CLI; checked in browser | Opt-in resource and movement mechanics |
| auto_patches | false in core/CLI; checked in browser | Derive institution/patch count from population |
| agents_per_patch | 5 | Sizing target, 1–200; 0 in older configs normalizes to 5 |
| grid_width / grid_height | 12 / 9 | Minimum map dimensions; zero normalizes to defaults, 1–5 expands to 6, max input 64 |
| move_brain_cost / move_body_cost | 0.05 / 0.02 | Finite nonnegative amounts per accepted cardinal step |
| harvest_access | members | `members` or `open`; older empty value normalizes to members |

The existing 200-agent MVP limit remains intentional. Automatic world growth removes manual provisioning work; it is not unlimited compute or horizontal scaling. Simulation execution remains one Go process with SQLite and serialized trusted strategy calls. Large maps can be inspected using zoom and scroll. Adding/removing agents during an active run, heterogeneous patch capacities, negotiable access AVAs and dynamic patch ownership are deferred.

An example [100-agent / 20-patch dashboard capture](grid-100-agents.png) is included alongside the main dashboard screenshot.
