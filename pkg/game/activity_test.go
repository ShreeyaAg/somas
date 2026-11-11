package game

import (
	"github.com/MattSScott/basePlatformSOMAS/v2/pkg/agent"
	"math"
	"math/rand"
	"testing"
)

type activityAgent struct{ *Base }

func (*activityAgent) Strategy() string { return "activity" }
func (*activityAgent) Act(o Observation, _ Memory, _ *rand.Rand) Decision {
	if o.Self.Activity != nil {
		o.Self.Activity.StartBrain = 99999
	}
	return Decision{Harvest: 1, Eat: .8, Contribute: .5}
}
func (*activityAgent) Propose(o Observation, _ Memory, _ *rand.Rand) *Proposal {
	if o.Institution.Rules.Quota != 1.2 {
		return &Proposal{Key: "quota", Value: 1.2}
	}
	return nil
}
func (*activityAgent) Vote(Observation, Proposal, Memory, *rand.Rand) bool { return true }
func (*activityAgent) Communicate(Observation, Memory, *rand.Rand) []Message {
	return []Message{{To: "institution", Type: "cooperate", Content: "cooperate"}}
}
func (*activityAgent) Move(o Observation, _ Memory, _ *rand.Rand) Position {
	p := o.Self.Position
	if o.Round%2 == 1 {
		p.X++
	} else {
		p.X--
	}
	return p
}
func activityRegistry() Registry {
	return Registry{"activity": func(s agent.IExposedServerFunctions[Agent]) Agent { return &activityAgent{NewBase(s)} }}
}
func assertActivityBalances(t *testing.T, p Player, round int) {
	t.Helper()
	a := p.Activity
	if a == nil {
		return
	}
	if a.Round != round {
		t.Fatal("stale activity round")
	}
	for _, tt := range []struct {
		name      string
		got, want float64
	}{
		{"brain", p.Brain, a.StartBrain + a.SunlightGain - a.ProposalCost - a.VoteCost - a.MessageCost - a.MoveBrainCost - a.HarvestCost - a.Contribution},
		{"body", p.Body, a.StartBody + a.BodyGained - a.MoveBodyCost - a.Metabolism},
		{"stock", p.Stock, a.StartStock + a.Harvested - a.Eaten - a.Rotted},
		{"sun", a.SunlightAvailable, a.SunlightGain + a.SunlightBlocked},
	} {
		if math.Abs(tt.got-tt.want) > 1e-9 {
			t.Fatalf("agent %s round %d %s balance got=%g want=%g ledger=%+v", p.ID, round, tt.name, tt.got, tt.want, *a)
		}
	}
}
func TestActivityReconcilesEveryPhaseAndReformation(t *testing.T) {
	for _, spatial := range []bool{false, true} {
		c := DefaultConfig()
		c.Agents = 4
		c.Institutions = 1
		c.Spatial = spatial
		c.Rounds = 4
		c.ReformEvery = 2
		c.Strategies = []string{"activity"}
		e := mustEngine(t, c, activityRegistry())
		for _, p := range e.Snapshot().Players {
			if p.Activity != nil {
				t.Fatal("formation invented activity")
			}
		}
		seen := map[string]bool{}
		for e.Step() {
			s := e.Snapshot()
			for _, p := range s.Players {
				assertActivityBalances(t, p, s.Round)
				a := p.Activity
				if a == nil {
					continue
				}
				if a.ProposalCost > 0 {
					seen["proposal"] = true
				}
				if a.VoteCost > 0 {
					seen["vote"] = true
				}
				if a.MessageCost > 0 {
					seen["message"] = true
				}
				if a.MoveBrainCost > 0 && a.MoveBodyCost > 0 {
					seen["move"] = true
				}
				if a.HarvestCost > 0 {
					seen["harvest"] = true
				}
				if a.Contribution > 0 {
					seen["contribution"] = true
				}
				if a.BodyGained > 0 {
					seen["eating"] = true
				}
				if a.Rotted > 0 {
					seen["rot"] = true
				}
				if a.Metabolism > 0 {
					seen["metabolism"] = true
				}
			}
		}
		for _, action := range []string{"proposal", "vote", "message", "harvest", "contribution", "eating", "rot", "metabolism"} {
			if !seen[action] {
				t.Fatalf("mode=%t missing action coverage %s", spatial, action)
			}
		}
		if spatial && !seen["move"] {
			t.Fatal("movement costs missing")
		}
	}
}
func TestActivitySunlightCapAndDeadAgentReset(t *testing.T) {
	c := DefaultConfig()
	c.Agents = 2
	c.Institutions = 1
	c.InitialBrain = c.MaxBrain
	c.InitialBody = .1
	c.MaxEat = 0
	c.Metabolism = 1
	c.Revive = false
	c.Strategies = []string{"activity"}
	e := mustEngine(t, c, activityRegistry())
	e.Step()
	for _, p := range e.players {
		a := p.Activity
		if a == nil {
			t.Fatal("activity nil")
		}
		if a.SunlightGain != 0 {
			t.Fatalf("gain!=0: gain=%v start=%v brain=%v cloud=%v avail=%v", a.SunlightGain, a.StartBrain, p.Brain, a.Cloud, a.SunlightAvailable)
		}
		if a.SunlightAvailable <= 0 {
			t.Fatalf("avail<=0: avail=%v cloud=%v", a.SunlightAvailable, a.Cloud)
		}
		if a.SunlightBlocked != a.SunlightAvailable {
			t.Fatalf("blocked!=avail: blocked=%v avail=%v", a.SunlightBlocked, a.SunlightAvailable)
		}
		if a.Cloud < 0 || a.Cloud > c.Cloud {
			t.Fatalf("cloud out of range: cloud=%v cfgCloud=%v", a.Cloud, c.Cloud)
		}
	}
	e.Step()
	e.Step()
	for _, p := range e.players {
		if p.Alive {
			t.Fatal("expected end-round death")
		}
	}
	e.Step()
	for _, p := range e.players {
		if p.Activity != nil {
			t.Fatal("dead agent retained stale sunlight activity")
		}
	}
}
func TestActivityObservationAndSnapshotAreCopies(t *testing.T) {
	c := DefaultConfig()
	c.Strategies = []string{"activity"}
	e := mustEngine(t, c, activityRegistry())
	e.Step()
	o := e.observe("agent-001")
	o.Self.Activity.StartBrain = -999
	s := e.Snapshot()
	s.Players[0].Activity.Cloud = -999
	if e.players[0].Activity.StartBrain < 0 || e.players[0].Activity.Cloud < 0 {
		t.Fatal("ledger pointer leaked authoritative state")
	}
}

type excessiveAgent struct{ *Base }

func (*excessiveAgent) Strategy() string { return "excessive" }
func (*excessiveAgent) Act(Observation, Memory, *rand.Rand) Decision {
	return Decision{Harvest: 999, Eat: 999, Contribute: 999}
}
func TestActivityRecordsAppliedAmountsInsteadOfRequests(t *testing.T) {
	c := DefaultConfig()
	c.Agents = 1
	c.Institutions = 1
	c.Rounds = 1
	c.Sun = 0
	c.InitialBrain = .25
	c.MaxEat = .5
	c.InitialBody = 1
	c.MaxBody = 2
	c.Strategies = []string{"excessive"}
	r := Registry{"excessive": func(s agent.IExposedServerFunctions[Agent]) Agent { return &excessiveAgent{NewBase(s)} }}
	e := mustEngine(t, c, r)
	e.Run()
	p := e.Snapshot().Players[0]
	a := p.Activity
	if math.Abs(a.Harvested-2.5) > 1e-9 || a.Eaten != .5 || a.Contribution != 0 || math.Abs(a.HarvestCost-.25) > 1e-9 {
		t.Fatalf("request values leaked into applied ledger: %+v", a)
	}
	assertActivityBalances(t, p, 1)
}
