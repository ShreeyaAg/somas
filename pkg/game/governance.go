package game

import (
	"fmt"
	"math"
)

func (e *Engine) politics() {
	for gi := range e.institutions {
		g := &e.institutions[gi]
		for pass := 0; pass < e.config.PoliticalPasses; pass++ {
			proposed := false
			for _, id := range append([]string(nil), g.Members...) {
				p := e.player(id)
				if !p.Alive || p.Brain <= 0 || p.Brain < e.config.ProposalCost {
					continue
				}
				proposal := e.agents[id].Propose(e.observe(id), e.Memory(id), e.rng)
				if proposal == nil {
					continue
				}
				q := *proposal
				if !validProposal(q, e.config) {
					e.note(id + " invalid proposal rejected")
					continue
				}
				proposed = true
				p.Brain -= e.config.ProposalCost
				v := VoteRecord{Round: e.round, Institution: gi, Proposer: id, Proposal: q}
				for _, voter := range g.Members {
					vp := e.player(voter)
					if !vp.Alive || vp.Brain <= 0 || vp.Brain < e.config.VoteCost {
						continue
					}
					vp.Brain -= e.config.VoteCost
					v.Cast++
					if e.agents[voter].Vote(e.observe(voter), q, e.Memory(voter), e.rng) {
						v.Yes++
					}
				}
				v.Accepted = v.Cast > 0 && float64(v.Yes) > float64(v.Cast)*g.Rules.VoteThreshold
				if v.Accepted {
					switch q.Key {
					case "quota":
						g.Rules.Quota = q.Value
					case "order":
						g.Rules.Order = q.Order
					case "vote_threshold":
						g.Rules.VoteThreshold = q.Value
					}
				}
				e.votes = append(e.votes, v)
				e.note(fmt.Sprintf("institution %d %s accepted=%t votes=%d/%d", gi, q.Key, v.Accepted, v.Yes, v.Cast))
			}
			if !proposed {
				break
			}
			if pass == e.config.PoliticalPasses-1 {
				e.note(fmt.Sprintf("institution %d political safety limit", gi))
			}
		}
	}
}
func validProposal(p Proposal, c Config) bool {
	if math.IsNaN(p.Value) || math.IsInf(p.Value, 0) {
		return false
	}
	switch p.Key {
	case "quota":
		return p.Value >= 0 && p.Value <= c.Capacity
	case "order":
		return p.Order == "random" || p.Order == "fixed" || p.Order == "rotating"
	case "vote_threshold":
		return p.Value >= .5 && p.Value < 1
	}
	return false
}
