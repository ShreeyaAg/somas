package agents

import (
	"github.com/MattSScott/basePlatformSOMAS/v2/pkg/agent"
	"math/rand"
	"somas/pkg/game"
	"strconv"
)

type Random struct{ *game.Base }

func (*Random) Strategy() string { return "random" }
func (*Random) Act(o game.Observation, m game.Memory, r *rand.Rand) game.Decision {
	return game.Decision{Harvest: r.Float64() * 5, Eat: r.Float64() * o.Config.MaxEat, Contribute: r.Float64() * o.Self.Brain}
}
func (*Random) Propose(o game.Observation, m game.Memory, r *rand.Rand) *game.Proposal {
	if r.Float64() < .8 {
		return nil
	}
	return &game.Proposal{Key: "quota", Value: 1 + r.Float64()*3}
}
func (*Random) Vote(o game.Observation, p game.Proposal, m game.Memory, r *rand.Rand) bool {
	return r.Intn(2) == 0
}

type Cooperative struct{ *game.Base }

func (*Cooperative) Strategy() string { return "cooperative" }
func (*Cooperative) Act(o game.Observation, m game.Memory, r *rand.Rand) game.Decision {
	n, _ := strconv.Atoi(m.Get("decisions"))
	m.Set("decisions", strconv.Itoa(n+1))
	return game.Decision{Harvest: min(o.Institution.Rules.Quota, max(0, o.Config.Metabolism+(.5-o.Self.Stock))), Eat: o.Config.MaxEat, Contribute: o.Config.MinContribution}
}
func (*Cooperative) Propose(o game.Observation, m game.Memory, r *rand.Rand) *game.Proposal {
	if o.Institution.Rules.Quota > 1.3 {
		return &game.Proposal{Key: "quota", Value: 1.2}
	}
	return nil
}
func (*Cooperative) Vote(o game.Observation, p game.Proposal, m game.Memory, r *rand.Rand) bool {
	return p.Key != "quota" || p.Value <= 2
}
func (*Cooperative) Communicate(o game.Observation, m game.Memory, r *rand.Rand) []game.Message {
	return []game.Message{{To: "institution", Type: "cooperation", Content: "Contribute to sustain our patch."}}
}
func Registry() game.Registry {
	r := game.Registry{}
	r.Register("random", func(s agent.IExposedServerFunctions[game.Agent]) game.Agent { return &Random{game.NewBase(s)} })
	r.Register("cooperative", func(s agent.IExposedServerFunctions[game.Agent]) game.Agent { return &Cooperative{game.NewBase(s)} })
	return r
}
