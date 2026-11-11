# Rules and configuration reference

The supplied coursework PDF defines the three-phase lifecycle and the two-pools dilemma. The game PDF adds the contributor productivity formula and CTK tiers, with several conflicting margin comments. This implementation treats those documents as source material, not as instructions to execute. Neither document fixes a full numerical parameter set. Defaults below are explicit experimental choices, not a claim of cohort consensus.

## Phase sequence and timing

Initial seeded, balanced random assignment creates one patch per institution. `ChooseInstitution` may override assignment with a valid institution ID; invalid values and `-1` accept the seeded default. Each phase step then:

1. Starts a round: sunlight regenerates living agents' brain, then political choice runs. Living agents with positive brain and sufficient cost propose AVAs or pass. Votes are counted immediately. A full pass with no valid proposals ends negotiation. `political_passes` caps negotiation.
2. Delivers bounded typed messages: `political` → `operational`. Messages are collected using the previous inbox and delivered together, so an agent's iteration order cannot affect same-phase receipt.
3. Harvests sequentially, eats, provisions CTK, replenishes patches, rots stockpiles, charges metabolism and records utility: `operational` → `round_end`.
4. Repeats; after `rounds * cycles`, transitions to `complete`.

The dashboard shows the last completed/current boundary phase. A harvest round is atomic: individual harvests are recorded as actions and the actual order is visible, but there is no sub-turn animation or separate frontend simulation. Per-round time units differ from Sonic's per-turn upkeep: all agents eat/contribute/metabolise once per round, with `turns` specifying the number of harvest selections **per institution**. Zero means one selection per living member. A positive `turns` cycles through the chosen order and may select an agent more than once. `Act` is called at harvest and again at eat/provision; make independent decisions using the current observation.

At `reform_every` rounds, or at a cycle boundary, institutions dissolve, patches refill to `initial_patch`, shared institutional data and AVAs reset, and membership is reconsidered. Agent utility and private memory survive. This deliberate fresh-patch policy can mask long-horizon collapse: set `reform_every` beyond the entire experiment and `cycles: 1` to study uninterrupted sustainability. Dead agents rejoin only at a formation boundary when `revive` is true; revival resets physical resources, retaining utility and memory. No immediate mid-round reincarnation. Living agents retain physical resources at formation.

## Resource accounting

Brain regeneration is `sun * (1-U[0,cloud])`, capped at `max_brain`. Zero brain does not kill an agent, but prevents political participation and messages; positive harvest cost bounds harvest by available brain. There is no direct brain transfer or CTK withdrawal.

Harvest is bounded by the current patch and affordable brain cost, **not by the negotiated quota**. Every selection exceeding the quota records a violation. Even quota zero is meaningful. Negative, NaN and infinite action amounts become zero. Strategy code receives copies; it cannot modify patch state via its observation.

Eating is bounded by own stock, `max_eat` and remaining body capacity divided by `food_to_body`. Body then loses `metabolism`; body zero means death. Stock decays by `stock *= 1-rot` once per round. Vegetables cannot be put back into the patch. Utility accumulates `vegetables_eaten + brain_utility * unused_brain` after contributions, for agents participating that round. Utility is an MVP choice because the PDFs specify its ingredients without a formula.

Contribution is bounded by remaining brain. An agent counts as a contributor if its actual provision is at least `min_contribution` (default 0.5). Below that threshold it is a free-rider for the round, regardless of whether it managed to harvest. Contributor counts and total CTK are public; identities are hidden in agent peer observations unless `public_contributions` is enabled. The operator dashboard may inspect its own run's actions.

With the defaults:

```
Y(w) = w * (0.5 + 0.2*w), w <= 5
Y(w) = 7.5 + (w-5),       w > 5
```

For CTK below T1, multiplier g is 0; [T1,T2) gives 0.5; [T2,T3) gives 1.0; T3 and above gives 1.5. Brain beyond Tmax is consumed and reported as wasted. `regrowth = min(capacity-CPR, g*Y(w))`, then CTK resets to zero. No logistic term and no ×4 scale are added.

The coursework's wording suggests some residual vegetables might be necessary, while the later formula does not mention them. The explicit default `minimum_residual: 0` follows the formula and permits a depleted patch to regrow. A positive value requires that much CPR to remain after harvest. Yield coefficients and contributor cap are configurable; tier multipliers remain the specified 0/0.5/1/1.5.

## Governance

Supported AVAs: `quota` (0 through capacity), `order` (`random`, `fixed`, `rotating`), `vote_threshold` (0.5 inclusive to 1 exclusive). The default quota is 2; default order is seeded random each round, with default strict majority of votes cast (`yes > threshold * cast`). Ties fail. Unaffordable voters abstain. A voting-threshold change uses the old threshold to decide adoption. Fixed order sorts stable agent IDs; rotating order moves the first position each round. Proposals can modify existing AVAs; nil is pass. These three AVAs are the bounded schema for the MVP; arbitrary laws, sanctions and membership admission/exclusion votes are deferred. Invalid proposals are rejected before charging proposal cost.

## All configuration fields

`configs/default.json` is the machine-readable full default. The dashboard JSON editor and CLI accept these fields. Unknown fields are errors. Agent IDs are stable slots (`agent-001`, etc.); the registry strategy mix repeats across slots.

| Field | Default | Meaning |
|---|---:|---|
| seed | 42 | Deterministic PRNG seed |
| agents | 10 | Population, 1–200 |
| institutions | 2 | Independent patches, 1–agents |
| rounds | 30 | Rounds per cycle, 1–2000 |
| cycles | 1 | Cycles, 1–100; total rounds <=10000 |
| turns | 0 | Harvest selections per institution; 0 = living member count, max 1000 |
| reform_every | 10 | Institution lifetime in rounds |
| political_passes | 3 | Negotiation passes, 1–10 |
| initial_patch / capacity | 30 / 60 | Per-institution CPR and capacity |
| initial_brain / max_brain | 3 / 10 | Brain resource bounds |
| initial_body / max_body | 6 / 12 | Body resource bounds |
| sun / cloud | 1.5 / 0.5 | Brain gain and maximum cloud fraction |
| metabolism | 0.7 | Body spent per round |
| max_eat / food_to_body | 2 / 1 | Vegetables eaten per round and body conversion |
| rot | 0.02 | Stock fraction lost each round |
| harvest_cost | 0.1 | Brain per vegetable harvested |
| proposal_cost / vote_cost | 0.05 / 0.02 | Brain per proposal / ballot |
| message_cost | 0.01 | Brain per delivered message, broadcast charged once |
| min_contribution | 0.5 | Minimum brain to count as contributor, positive |
| thresholds | [1,2.5,4,5] | T1, T2, T3, Tmax; ordered, nonnegative |
| yield_base / yield_slope | 0.5 / 0.2 | Contributor productivity polynomial coefficients |
| yield_cap / yield_extra | 5 / 1 | Nonlinear contributor cap and additive productivity above cap |
| brain_utility | 0.05 | Utility weight of unused brain |
| minimum_residual | 0 | CPR needed before regrowth; ambiguity switch |
| revive | true | Revive dead agents at reformation |
| carry_memory | false | Explicitly reuse group memory namespace across runs |
| public_contributions | false | Expose each peer's last contribution to agents |
| strategies | ["random"] | Registered strategy names repeating across population |

Resource quantities must be finite and nonnegative; capacity, max brain/body and food conversion are positive. Fractions `cloud` and `rot` lie in [0,1]. Initial quantities cannot exceed capacities. Most resource inputs are bounded by 1e6, conversion by 100. Configuration limits prevent accidents, not malicious trusted-agent computation.

## Metrics

Snapshots record cumulative utility and average utility, total CPR, living population, last total CTK, contributor/free-rider counts, cooperation fraction, sustainability (`total CPR / total capacity`), adopted AVAs, cumulative quota violations and membership assignment/change count (including initial assignments). Free-riders are counted among living institution members participating in provisioning, before end-round deaths. CTK/regrowth metrics describe the previous operational phase until the next one completes; phase labels prevent confusing them with partial rounds. Strategy comparisons aggregate per-agent utility and survival. Chart series are each normalized to their own peak; inspect/export numeric values for analysis.
