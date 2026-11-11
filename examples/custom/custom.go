// Package custom is a buildable student strategy template.
package custom

import (
	"github.com/MattSScott/basePlatformSOMAS/v2/pkg/agent"
	"math/rand"
	"somas/pkg/game"
)

type Student struct{ *game.Base }

func (*Student) Strategy() string { return "student" }
func (*Student) Act(o game.Observation, m game.Memory, r *rand.Rand) game.Decision {
	m.Set("previous_institution", o.Self.ID)
	return game.Decision{Harvest: min(1, o.Institution.Rules.Quota), Eat: 2, Contribute: .5}
}
func Register(r game.Registry) {
	r.Register("student", func(s agent.IExposedServerFunctions[game.Agent]) game.Agent { return &Student{game.NewBase(s)} })
}
