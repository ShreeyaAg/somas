# Following agents and energy in the dashboard

This UI revision makes existing game mechanics observable. It does not introduce collision rules, change movement permissions, or alter resource calculations.

## Distinct agents on shared tiles

Every agent has a separate selectable token. Agents that share the same backend coordinate are displayed in a non-overlapping sub-grid **inside that logical tile**. Tile display dimensions expand with crowding; coordinates remain the simulation coordinates. Map markers include an ID, brain/body bars and compact recorded-action labels. Colours identify institution membership.

The default **Focus activity** extent centres on patches and agent positions instead of showing unused surrounding land. **Whole world** restores the complete grid. Fit keeps markers readable rather than shrinking a large crowd into illegible dots, so large populations may require scrolling. Zoom changes only presentation. Display slots are not movement destinations or a change to the shared-tile game rule.

Below the map, the searchable **Agent activity** roster shows brain, body, stock, utility and each agent's actual round totals for harvesting, eating and CTK. The roster scrolls independently for large populations. Click a token or card to inspect that agent's full ledger. The gold bar represents brain; the green bar represents body.

A visible mode badge distinguishes spatial movement from classic operation. Existing classic experiments remain classic when reopened. The **New spatial experiment** shortcut opens the setup with spatial/automatic sizing selected; it does not change the current run.

## Sunlight and energy

Sunlight is processed once at the start of each round. The sun panel displays actual **brain credited this round**, average sampled cloud fraction across that round's living participants, and energy not credited because an agent reached its brain cap. Clouds are sampled separately for each agent; this average is not a new global weather model.

The inspector explains an individual agent's accounting:

```
current brain = starting brain + actual sunlight credited
              - proposal/vote costs - message costs
              - movement brain cost - harvest brain cost - CTK provision
```

Body and stock changes are recorded separately: body gained through eating, body spent on movement/metabolism, vegetables harvested/eaten and stock rot. Sunlight does not directly replenish body or vegetables. Brain invested in the institution's CTK enables CPR regrowth under the existing tier/productivity rules.

The backend now adds an optional `Player.activity` ledger to operator snapshots. It records **applied** amounts, including caps and validation, not raw strategy requests. It resets for every new round; agents that were already dead receive no new ledger. Records persist with phase snapshots and appear in JSON export/replay. They are not exposed in public peer observations, and own observations deep-copy their ledger. Private memories/inboxes stay excluded from dashboard responses.

Previously saved snapshots have no ledger. The UI labels their energy/action history **Not recorded** rather than reconstructing sunlight from resource differences. Formation before round one also has no sunlight credit to display. Nothing is backfilled into old experiments.

## Reading the phases

The phase helper describes what is recorded and what the next step will do:

- **formation**: ready; next step applies sunlight and political decisions.
- **political**: sunlight and voting are recorded; next step handles messages/movement.
- **operational**: movement is recorded; next step harvests/eats/provisions/regrows.
- **round_end**: operational totals are recorded; next step starts another round.
- **complete**: finished; replay saved phases to inspect earlier state.

Action badges show the recorded state/totals for the selected phase. They are not an invented real-time animation of individual harvest turns; the engine still executes an operational phase atomically. Fast playback may skip intermediate phases on screen; pause, use Next phase, or replay to inspect each one.

## Validation

`pkg/game/activity_test.go` reconciles brain, body and stock ledgers in classic and spatial modes, verifies cap handling, rounding tolerance, reformation/death reset, applied action amounts and copy/privacy boundaries. A pre-telemetry binary comparison showed identical seeded game/config/history results after excluding only the new ledger fields, including reformation: no RNG draws or rule behavior changed.

`scripts/browser_clarity.py` checks rendered SVG bounding boxes for non-overlap in classic ten-agent/one-patch, spatial 100-agent and spatial 200-agent/one-patch scenarios, including mobile. It checks actual API sunlight sums, resource/action cards, inspection, replay and no-ledger archived fallback. Use a test server/group:

```sh
SOMAS_TEST_URL=http://127.0.0.1:8080 SOMAS_TEST_TOKEN=your-test-token python3 scripts/browser_clarity.py
```

The test writes screenshots to `work/clarity-*.png`. It uses Python Playwright and Chromium, which are optional test dependencies, not required to run the Go application.
