package game

import (
	"fmt"
	"sort"
)

func (e *Engine) operate() {
	if e.config.Spatial {
		e.operateSpatial()
	} else {
		e.operateLegacy()
	}
}
func (e *Engine) operateSpatial() {
	c := e.config
	e.participants = 0
	for i := range e.players {
		p := &e.players[i]
		if p.Alive {
			e.participants++
			p.LastAction += "; "
		}
	}
	// All patches harvest before any patch replenishes, so visitors never gain a
	// second harvest from an already-regrown patch in the same round.
	for gi := range e.institutions {
		g := &e.institutions[gi]
		g.Order = nil
		g.CTK = 0
		g.Contributors = 0
		g.Regrowth = 0
		patch := e.world.Patches[gi]
		for _, p := range e.players {
			if p.Alive && patch.Contains(p.Position) && (c.HarvestAccess == "open" || p.Institution == gi) {
				g.Order = append(g.Order, p.ID)
			}
		}
		sort.Strings(g.Order)
		switch g.Rules.Order {
		case "random":
			e.rng.Shuffle(len(g.Order), func(i, j int) { g.Order[i], g.Order[j] = g.Order[j], g.Order[i] })
		case "rotating":
			if len(g.Order) > 0 {
				n := (e.round - 1) % len(g.Order)
				g.Order = append(g.Order[n:], g.Order[:n]...)
			}
		}
		turns := c.Turns
		if turns == 0 {
			turns = len(g.Order)
		}
		for turn := 0; turn < turns && len(g.Order) > 0 && g.CPR > 0; turn++ {
			id := g.Order[turn%len(g.Order)]
			p := e.player(id)
			d := e.agents[id].Act(e.observe(id), e.Memory(id), e.rng)
			take := Clamp(d.Harvest, 0, g.CPR)
			if c.HarvestCost > 0 {
				take = min(take, p.Brain/c.HarvestCost)
			}
			p.Stock += take
			p.Brain = max(0, p.Brain-take*c.HarvestCost)
			g.CPR = max(0, g.CPR-take)
			if take > g.Rules.Quota+1e-9 {
				p.Violations++
				e.note(fmt.Sprintf("%s violated patch %d quota", id, gi))
			}
			p.LastAction += fmt.Sprintf("harvest %.2f from patch %d; ", take, gi)
		}
	}
	// CTK provision and political rights follow membership, even while travelling.
	for gi := range e.institutions {
		g := &e.institutions[gi]
		for _, id := range g.Members {
			p := e.player(id)
			if !p.Alive {
				continue
			}
			d := e.agents[id].Act(e.observe(id), e.Memory(id), e.rng)
			eat := Clamp(d.Eat, 0, min(p.Stock, c.MaxEat, max(0, (c.MaxBody-p.Body)/c.FoodToBody)))
			p.Stock -= eat
			p.Body += eat * c.FoodToBody
			p.Utility += eat
			provision := Clamp(d.Contribute, 0, p.Brain)
			p.Brain -= provision
			p.Contribution = provision
			g.CTK += provision
			if provision >= c.MinContribution {
				g.Contributors++
			}
			p.LastAction += fmt.Sprintf("eat %.2f; contribute %.2f to institution %d", eat, provision, p.Institution)
		}
	}
	for gi := range e.institutions {
		g := &e.institutions[gi]
		g.LastCTK = g.CTK
		g.Wasted = max(0, g.CTK-c.Thresholds[3])
		growth := Multiplier(min(g.CTK, c.Thresholds[3]), c) * Productivity(g.Contributors, c)
		if g.CPR < c.MinimumResidual {
			growth = 0
		}
		g.Regrowth = min(c.Capacity-g.CPR, growth)
		g.CPR += g.Regrowth
		g.CTK = 0
		g.Shared["contributors"] = fmt.Sprint(g.Contributors)
		g.Shared["last_ctk"] = fmt.Sprintf("%.3f", g.LastCTK)
		e.note(fmt.Sprintf("patch %d CTK %.2f contributors %d regrowth %.2f", gi, g.LastCTK, g.Contributors, g.Regrowth))
	}
	for i := range e.players {
		p := &e.players[i]
		if !p.Alive {
			continue
		}
		p.Stock *= 1 - c.Rot
		p.Body = max(0, p.Body-c.Metabolism)
		if p.Body == 0 {
			p.Alive = false
			e.note(p.ID + " died")
		}
		p.Utility += c.BrainUtility * p.Brain
	}
}
