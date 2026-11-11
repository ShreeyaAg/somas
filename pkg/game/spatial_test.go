package game

import (
	"github.com/MattSScott/basePlatformSOMAS/v2/pkg/agent"
	"math/rand"
	"reflect"
	"testing"
)

type movingAgent struct {
	*testAgent
	target Position
}

func (a *movingAgent) Move(Observation, Memory, *rand.Rand) Position { return a.target }
func spatialEngine(t *testing.T) *Engine {
	t.Helper()
	c := config()
	c.Spatial = true
	c.AutoPatches = true
	c.Rounds = 3
	return mustEngine(t, c, reg(false))
}
func TestAutomaticWorldScaling(t *testing.T) {
	for _, n := range []int{1, 5, 6, 10, 50, 200} {
		c := config()
		c.Agents = n
		c.AutoPatches = true
		c.AgentsPerPatch = 5
		c.GridWidth = 6
		c.GridHeight = 6
		e := mustEngine(t, c, reg(false))
		if len(e.world.Patches) != (n+4)/5 {
			t.Fatal("patch count", n)
		}
		occupied := map[Position]bool{}
		for _, p := range e.world.Patches {
			for x := p.Origin.X; x < p.Origin.X+p.Width; x++ {
				for y := p.Origin.Y; y < p.Origin.Y+p.Height; y++ {
					pos := Position{x, y}
					if occupied[pos] || x < 0 || y < 0 || x >= e.world.Width || y >= e.world.Height {
						t.Fatalf("overlap or out of bounds: n=%d patch=%+v", n, p)
					}
					occupied[pos] = true
				}
			}
		}
		for _, p := range e.players {
			patch := e.patchAt(p.Position)
			if patch == nil || patch.Institution != p.Institution {
				t.Fatal("bad spawn")
			}
		}
	}
}
func TestMovementValidationAndCosts(t *testing.T) {
	for _, kind := range []string{"valid", "diagonal", "teleport", "outside", "unaffordable", "dead"} {
		e := spatialEngine(t)
		p := &e.players[0]
		from := p.Position
		to := Position{from.X + 1, from.Y}
		switch kind {
		case "diagonal":
			to.Y++
		case "teleport":
			to.X += 2
		case "outside":
			to.X = int(^uint(0) >> 1)
		case "unaffordable":
			p.Brain = 0
		case "dead":
			p.Alive = false
		}
		originalBrain, originalBody := p.Brain, p.Body
		a := e.agents[p.ID].(*testAgent)
		e.agents[p.ID] = &movingAgent{testAgent: a, target: to}
		e.moveAgents()
		if kind == "valid" {
			if p.Position != to || p.Brain != originalBrain-e.config.MoveBrainCost || p.Body != originalBody-e.config.MoveBodyCost || p.Moves != 1 {
				t.Fatal("move or cost incorrect")
			}
		} else if p.Position != from || p.Brain != originalBrain || p.Body != originalBody || p.Moves != 0 {
			t.Fatal("invalid move changed state", kind)
		}
	}
}
func TestSpatialHarvestAccessAndContributionDestination(t *testing.T) {
	for _, access := range []string{"members", "open"} {
		e := spatialEngine(t)
		e.config.HarvestAccess = access
		e.config.HarvestCost = 0
		e.config.MaxEat = 0
		e.config.Rot = 0
		e.config.MinimumResidual = e.config.Capacity
		e.config.Sun = 0
		p := &e.players[0]
		home := p.Institution
		foreign := (home + 1) % len(e.institutions)
		for i := range e.players {
			e.players[i].Position = Position{0, 0}
		}
		p.Position = e.world.Patches[foreign].Origin
		before := e.institutions[foreign].CPR
		e.operate()
		if p.Institution != home {
			t.Fatal("walking changed membership")
		}
		if access == "open" {
			if p.Stock != 10 || e.institutions[foreign].CPR != before-10 {
				t.Fatal("visitor harvest not applied to target patch")
			}
		} else if p.Stock != 0 || e.institutions[foreign].CPR != before {
			t.Fatal("unauthorised harvest")
		}
		if e.institutions[home].LastCTK != .5*float64(len(e.institutions[home].Members)) {
			t.Fatal("contribution did not follow membership")
		}
		if e.Snapshot().Metrics.FreeRiders != 0 {
			t.Fatal("travelling contributors miscounted")
		}
	}
}
func TestSpatialSeedAndViews(t *testing.T) {
	a := spatialEngine(t)
	b := spatialEngine(t)
	for a.Step() {
		b.Step()
		if !reflect.DeepEqual(a.Snapshot(), b.Snapshot()) {
			t.Fatal("spatial run not deterministic")
		}
	}
	o := a.observe(a.players[0].ID)
	o.World.Patches[0].Origin.X = -999
	if a.world.Patches[0].Origin.X < 0 {
		t.Fatal("world aliased")
	}
	snap := a.Snapshot()
	snap.World.Patches[0].Width = 900
	if a.world.Patches[0].Width == 900 {
		t.Fatal("snapshot world aliased")
	}
}
func TestExistingBaseAgentDoesNotHaveToImplementMovement(t *testing.T) {
	e := spatialEngine(t)
	before := e.players[0].Position
	e.moveAgents()
	if e.players[0].Position != before {
		t.Fatal("default Base moved")
	}
}
func TestSpatialLifecycleMatchesPlatform(t *testing.T) {
	a := spatialEngine(t)
	b := spatialEngine(t)
	a.Run()
	b.RunWithPlatform()
	if !reflect.DeepEqual(a.Snapshot(), b.Snapshot()) {
		t.Fatal("spatial platform lifecycle differs")
	}
}
func TestSpatialConfigurationRejected(t *testing.T) {
	for _, mutate := range []func(*Config){func(c *Config) { c.MoveBrainCost = -1 }, func(c *Config) { c.HarvestAccess = "secret" }, func(c *Config) { c.GridWidth = 65 }, func(c *Config) { c.AgentsPerPatch = -1 }} {
		c := config()
		mutate(&c)
		if _, err := New(c, reg(false), nil); err == nil {
			t.Fatal("invalid spatial config accepted")
		}
	}
}

var _ agent.IAgent[Agent] = (*movingAgent)(nil)

func TestSpatialSunlightFollowsCloudOverTile(t *testing.T) {
	c := config()
	c.Spatial = true
	c.AutoPatches = true
	c.Rounds = 6
	c.Cloud = .8
	e := mustEngine(t, c, reg(false))
	if len(e.world.Clouds) != e.world.Width*e.world.Height {
		t.Fatal("cloud field missing from world")
	}
	varied := false
	for r := 0; r < 6; r++ {
		e.Step() // sunlight is applied at the start of the round
		for _, p := range e.players {
			if p.Activity == nil {
				continue
			}
			want := e.world.Clouds[p.Position.Y*e.world.Width+p.Position.X]
			if p.Activity.Cloud != want || p.Activity.Cloud > c.Cloud {
				t.Fatal("agent cloud must equal the cloud over its tile")
			}
		}
		for _, v := range e.world.Clouds {
			if v != e.world.Clouds[0] {
				varied = true
			}
		}
		e.Step()
		e.Step()
	}
	if !varied {
		t.Fatal("clouds should vary across the grid")
	}
}
