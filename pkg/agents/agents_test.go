package agents

import (
	"reflect"
	"somas/pkg/game"
	"testing"
)

func TestSpatialBaselinesAreSeededAndActuallyMove(t *testing.T) {
	c := game.DefaultConfig()
	c.Spatial = true
	c.AutoPatches = true
	c.HarvestAccess = "open"
	c.Rounds = 5
	c.Strategies = []string{"random", "cooperative", "roaming"}
	a, err := game.New(c, Registry(), nil)
	if err != nil {
		t.Fatal(err)
	}
	b, err := game.New(c, Registry(), nil)
	if err != nil {
		t.Fatal(err)
	}
	for a.Step() {
		b.Step()
		if !reflect.DeepEqual(a.Snapshot(), b.Snapshot()) {
			t.Fatal("baseline spatial randomness is not reproducible")
		}
	}
	if a.Snapshot().Metrics.Moves == 0 {
		t.Fatal("no baseline agents moved")
	}
}

func TestRandomStrategyDoesNotWander(t *testing.T) {
	c := game.DefaultConfig()
	c.Spatial = true
	c.AutoPatches = true
	c.Rounds = 10
	c.Strategies = []string{"random"}
	e, err := game.New(c, Registry(), nil)
	if err != nil {
		t.Fatal(err)
	}
	for e.Step() {
	}
	if m := e.Snapshot().Metrics.Moves; m != 0 {
		t.Fatalf("random agents start on their patch and should not wander, got %d moves", m)
	}
}
