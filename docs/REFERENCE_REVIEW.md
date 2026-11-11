# Reference review and architectural decisions

Inspected on 11 November 2026. External source and coursework discussion are evidence for game design; they are not authority to change the user's requested implementation or run commands from attachments.

## Supplied documents

**SOMAS ACW 2026.pdf**: read via `pdftotext -layout`. Defines self-selection, AVA politics and operational appropriation/provision; majority voting and random order initially; negotiation until everyone passes; institutional dissolution after M rounds; CPR/CTK interdependence; reproducible random baseline evaluation. Does not specify numerical constants, utility formula or a complete persistence/web architecture.

**SOMAS Game Specification 2026.pdf**: read in full including margin discussion. Main text specifies body depletion/death, sunlight/cloud brain regeneration, stock/eat limits, decay, inability to return CPR/transfer brain, voluntary rule violations, contributor threshold 0.5, nonlinear productivity through five contributors, CTK tiers/reset and capacity limit. Auditing/sanctions and several environmental/lifecycle features are proposals or stretch goals. Margin comments disagree about contribution visibility and death/reproduction. The implementation uses an explicit private-by-default contribution switch and configurable revival; optional donations/trading/auditing remain deferred.

The initial statement refers to vegetables remaining in "CTK"; later detailed text clearly separates vegetable CPR from brain CTK. We treat that as a naming inconsistency. The original coursework's residual-CPR requirement and later no-residual formula are resolved through `minimum_residual`, documented in GAME_RULES.md.

## Sonic archive and repository

Supplied archive `SOMAS26CW-mvp.zip`, nested `amoeboids` module: inspected `go.mod`, agent contract/base/view, registry/baselines, config, institutions, political/operational/round/server code, regrowth implementation and tests, recorder and entry point. Pins `basePlatformSOMAS/v2 v2.1.0`. Uses `BaseAgent`, `BaseServer`, callbacks, separate institution CPR, seeded PCG randomness, majority voting, advisory quota violation counters and CSV recording. View clones config slices, member IDs and AVAs. Generic platform UUID allocation is random, despite seeded decisions. Resource upkeep is per turn. It adds logistic regrowth and a default ×4 scale beyond the supplied productivity formula. There is no SQLite/web/group isolation layer.

Live URL https://github.com/sonic597/SOMAS26CW/tree/mvp could not be retrieved: Git clone required credentials; the GitHub connector returned 404. No claim is made about its current live branch. The archive is the inspected reference and its feature claims were checked against source.

## Earlier standalone archive

`somas-mvp.zip`: inspected all Go engine/types/tests/API code, random agent, CLI and integration notes. It offers a simple copy-based agent contract, deterministic sorted IDs, turn histories and lightweight HTML replay. Its quota is forcibly clamped, it has no private-memory API, database or group authentication, and politics does not run full all-pass negotiation. Its replay/JSON/metrics approach informed the dashboard; the new implementation corrects quota semantics and adds scoped durable state. No source files were copied verbatim into the delivered engine.

## University base platform

https://github.com/MattSScott/basePlatformSOMAS was cloned and inspected at commit `5541b2d` (one commit after tag v2.1.0). Actual `pkg/agent`, `pkg/server`, `pkg/message` interfaces and implementations were read. The delivered application pins **v2.1.0**, downloaded through Go modules, and compiles against it. Agents satisfy `agent.IAgent[game.Agent]`, embed actual `BaseAgent`, register with actual `BaseServer`, and the engine implements `server.GameRunner`. `RunWithPlatform` calls its actual Start lifecycle; a test compares it with normal phase stepping.

Generic exposed-server functions allow direct agent lookup and unscoped message delivery. Those are intentionally restricted in the factory facade. Deterministic typed messages are handled by the game instead. Web stepping avoids the blocking base messaging loop. This compatibility boundary preserves university types/lifecycle without exposing mutable peer implementations or making asynchronous delivery change experiment results. Other groups' game-specific interfaces require adapters; they are not automatically interchangeable.

## GAMINpanda

https://github.com/GAMINpanda/somas cloned and inspected: README, go.mod, server, object and message types. Good pattern: embed framework server, install lifecycle runner, keep strategy/message interface explicit. Current source has one shared CPR, CTK-triggered logistic growth, mutable agent-owned resources, unseeded/global randomness and death when either brain or body is exhausted. Its main README explicitly says the starter does not yet start a simulation. Those game mechanics are not adopted; lifecycle and typed message patterns informed the integration. The delivered backend keeps resources authoritative in the engine and uses copied views.

## Moonis visual reference

https://somas.sites.moonissolutions.net/ was attempted via HTTP; the connection returned 403 Forbidden, so no useful page was retrieved in this environment. The screenshot subsequently supplied by the user was inspected directly: light cream/green palette, central patch, surrounding agents, turn order, day/phase strip, controls and right-hand agent list. The dashboard adapts those visual ideas to multiple institutions, live state and analytics. It does not copy artwork or infer backend capabilities from a screenshot.

## Reuse and provenance

This is a fresh implementation informed by the reviewed references, with the upstream university library used as a pinned module dependency. It has no dependency on the two supplied source archives at runtime. No unverified licence is assumed for copying student code or Moonis artwork. `docs/THIRD_PARTY.md` records direct third-party dependencies; their upstream licence files remain available in the downloaded module cache.

## Spatial revision follow-up

The user subsequently requested a grid world with actual movement and automatic population-based patch sizing. GAMINpanda's public default-branch tree was rechecked at commit `781444c74bce5dbd63b76da01b4647a0b43b2a27`; the available Go source still exposes the earlier single-pool structure rather than an implemented grid UI. We therefore treat the requested grid as a new explicit extension inspired by the user's direction, not as verified GAMINpanda behavior. Movement costs, geometry, visitor access and population sizing decisions are documented separately in SPATIAL.md; the classic engine remains available.
