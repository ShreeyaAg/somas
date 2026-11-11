package custom

import (
	"somas/pkg/agents"
	"somas/pkg/game"
	"testing"
)

func TestRegisteredStudentRuns(t *testing.T) {
	r := agents.Registry()
	Register(r)
	c := game.DefaultConfig()
	c.Strategies = []string{"student"}
	c.Rounds = 3
	e, err := game.New(c, r, nil)
	if err != nil {
		t.Fatal(err)
	}
	e.Run()
	if e.Snapshot().Phase != "complete" || e.Snapshot().Metrics.Alive != c.Agents {
		t.Fatal("custom strategy did not run")
	}
}
