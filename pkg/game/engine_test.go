package game

import (
	"github.com/MattSScott/basePlatformSOMAS/v2/pkg/agent"
	"github.com/google/uuid"
	"math"
	"math/rand"
	"reflect"
	"testing"
)

type testAgent struct {
	*Base
	hostile bool
}

func (*testAgent) Strategy() string { return "test" }
func (a *testAgent) Act(o Observation, m Memory, r *rand.Rand) Decision {
	m.Set("secret", o.Self.ID)
	if a.hostile {
		o.Institution.Members[0] = "intruder"
		o.Institution.Shared["hacked"] = "true"
		o.Config.Strategies[0] = "hacked"
		return Decision{Harvest: math.NaN(), Eat: math.Inf(1), Contribute: -100}
	}
	return Decision{Harvest: 10, Eat: 2, Contribute: .5}
}
func (*testAgent) Propose(o Observation, m Memory, r *rand.Rand) *Proposal {
	if o.Institution.Rules.Quota != 1 {
		return &Proposal{Key: "quota", Value: 1}
	}
	return nil
}
func (*testAgent) Vote(Observation, Proposal, Memory, *rand.Rand) bool { return true }
func reg(hostile bool) Registry {
	return Registry{"test": func(s agent.IExposedServerFunctions[Agent]) Agent { return &testAgent{NewBase(s), hostile} }}
}
func config() Config { c := DefaultConfig(); c.Strategies = []string{"test"}; return c }
func mustEngine(t *testing.T, c Config, r Registry) *Engine {
	t.Helper()
	e, err := New(c, r, nil)
	if err != nil {
		t.Fatal(err)
	}
	return e
}
func TestGrowthBoundaries(t *testing.T) {
	c := config()
	for w, want := range map[int]float64{0: 0, 1: .7, 5: 7.5, 6: 8.5, 10: 12.5} {
		if math.Abs(Productivity(w, c)-want) > 1e-9 {
			t.Fatalf("yield %d", w)
		}
	}
	for _, tt := range []struct{ v, w float64 }{{0, 0}, {1, .5}, {2.499, .5}, {2.5, 1}, {4, 1.5}, {5, 1.5}, {100, 1.5}} {
		if got := Multiplier(tt.v, c); got != tt.w {
			t.Fatalf("multiplier %v=%v", tt.v, got)
		}
	}
}
func TestDeterminismInvariantsAndIsolation(t *testing.T) {
	c := config()
	for _, hostile := range []bool{false, true} {
		e := mustEngine(t, c, reg(hostile))
		other := mustEngine(t, c, reg(hostile))
		for e.Step() {
			if !other.Step() {
				t.Fatal("steps differ")
			}
			s := e.Snapshot()
			if !reflect.DeepEqual(s, other.Snapshot()) {
				t.Fatal("seeded states differ")
			}
			for _, g := range s.Institutions {
				if g.CPR < 0 || g.CPR > c.Capacity || g.CTK != 0 || g.Shared["hacked"] != "" {
					t.Fatal("invalid patch / mutated shared view")
				}
			}
			for _, p := range s.Players {
				if math.IsNaN(p.Brain) || p.Brain < 0 || p.Brain > c.MaxBrain || p.Body < 0 || p.Body > c.MaxBody || p.Stock < 0 {
					t.Fatalf("invalid resources %+v", p)
				}
			}
		}
		if e.config.Strategies[0] != "test" {
			t.Fatal("config aliased")
		}
		if e.Memory("agent-001").Get("secret") == "agent-002" {
			t.Fatal("private memory leaked")
		}
		s := e.Snapshot()
		s.Institutions[0].Shared["hacked"] = "true"
		if e.Snapshot().Institutions[0].Shared["hacked"] != "" {
			t.Fatal("snapshot aliased")
		}
	}
}
func TestAdvisoryQuotaAndZeroBrain(t *testing.T) {
	c := config()
	c.Rounds = 1
	c.Sun = 0
	c.InitialBrain = 10
	e := mustEngine(t, c, reg(false))
	e.Run()
	s := e.Snapshot()
	if s.Metrics.Accepted == 0 || s.Metrics.Violations == 0 {
		t.Fatal("politics or advisory quota missing")
	}
	c.InitialBrain = 0
	e = mustEngine(t, c, reg(false))
	e.Step()
	if len(e.votes) > 0 {
		t.Fatal("zero brain can vote")
	}
	if !e.players[0].Alive {
		t.Fatal("zero brain killed agent")
	}
}
func TestContributorThresholdAndCTKReset(t *testing.T) {
	c := config()
	c.Agents = 5
	c.Institutions = 1
	c.Rounds = 1
	c.InitialBrain = 10
	c.HarvestCost = 0
	e := mustEngine(t, c, reg(false))
	e.Run()
	g := e.Snapshot().Institutions[0]
	if g.Contributors != 5 || g.LastCTK != 2.5 || g.CTK != 0 || g.Regrowth != 7.5 {
		t.Fatalf("incorrect regrowth %+v", g)
	}
}
func TestValidation(t *testing.T) {
	for _, mutate := range []func(*Config){func(c *Config) { c.Cloud = 2 }, func(c *Config) { c.MaxBody = math.NaN() }, func(c *Config) { c.InitialPatch = c.Capacity + 1 }, func(c *Config) { c.Thresholds[1] = 0 }, func(c *Config) { c.Rot = -1 }, func(c *Config) { c.Agents = 0 }, func(c *Config) { c.FoodToBody = 0 }} {
		c := config()
		mutate(&c)
		if c.Validate() == nil {
			t.Fatal("accepted invalid config")
		}
	}
}
func TestPlatformLifecycle(t *testing.T) {
	c := config()
	c.Rounds = 2
	a := mustEngine(t, c, reg(false))
	b := mustEngine(t, c, reg(false))
	a.Run()
	b.RunWithPlatform()
	if !reflect.DeepEqual(a.Snapshot(), b.Snapshot()) {
		t.Fatal("platform adapter differs")
	}
	for _, ag := range b.agents {
		if ag.AccessAgentByID(uuid.Nil) != nil || len(ag.ViewAgentIdSet()) != 1 {
			t.Fatal("framework exposes peer implementations")
		}
	}
}
func TestObservationDoesNotExposePrivatePeerResources(t *testing.T) {
	e := mustEngine(t, config(), reg(false))
	o := e.observe("agent-001")
	if len(o.Peers) != 10 || o.Peers[1].Contribution != nil {
		t.Fatal("unexpected peer view")
	}
	if _, ok := reflect.TypeOf(o.Peers[1]).FieldByName("Brain"); ok {
		t.Fatal("peer brain visible")
	}
}

type messagingAgent struct{ *testAgent }

func (*messagingAgent) Communicate(o Observation, m Memory, r *rand.Rand) []Message {
	if o.Self.ID != "agent-001" {
		return nil
	}
	return []Message{{From: "forged", To: "agent-002", Type: "private", Content: "private-direct-secret"}, {From: "forged", To: "institution", Type: "cooperation", Content: "public-broadcast"}, {To: "missing-agent", Type: "private", Content: "invalid-route"}}
}
func TestCommunicationRoutingAndVisibility(t *testing.T) {
	c := config()
	c.Institutions = 1
	r := Registry{"test": func(s agent.IExposedServerFunctions[Agent]) Agent {
		return &messagingAgent{&testAgent{NewBase(s), false}}
	}}
	e := mustEngine(t, c, r)
	e.Step()
	e.Step()
	if len(e.inbox["agent-002"]) != 2 || len(e.inbox["agent-003"]) != 1 || len(e.inbox["agent-001"]) != 0 {
		t.Fatal("incorrect delivery")
	}
	m := e.inbox["agent-002"][0]
	if m.From != "agent-001" || m.Round != 1 || m.Institution != 0 {
		t.Fatal("forged metadata accepted")
	}
	snap := e.Snapshot()
	if len(snap.PublicMessages) != 1 || snap.PublicMessages[0].Content != "public-broadcast" {
		t.Fatal("private messages leaked to snapshot")
	}
}
func TestDeathAndRevivalBoundary(t *testing.T) {
	for _, revive := range []bool{true, false} {
		c := config()
		c.Revive = revive
		c.InitialBody = .1
		c.MaxEat = 0
		c.Metabolism = 1
		c.ReformEvery = 1
		e := mustEngine(t, c, reg(false))
		e.Step()
		e.Step()
		e.Step()
		if e.players[0].Alive || e.players[0].Body != 0 {
			t.Fatal("body death failed")
		}
		e.Step()
		if e.players[0].Alive != revive {
			t.Fatal("revival policy ignored")
		}
	}
}
