package store

import (
	"path/filepath"
	"reflect"
	"somas/pkg/game"
	"testing"
)

func TestPersistenceAndScope(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.db")
	db, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	c := game.DefaultConfig()
	r := Run{ID: "r1", Group: "a", Config: c, State: game.Snapshot{Round: 1}}
	m := map[string]map[string]string{"a1": {"secret": "private"}}
	if err = db.Save(r, "carry:test", m, 0); err != nil {
		t.Fatal(err)
	}
	db.Close()
	db, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	got, err := db.Memories("a", "carry:test")
	if err != nil || !reflect.DeepEqual(got, m) {
		t.Fatal(got, err)
	}
	other, _ := db.Memories("b", "carry:test")
	if len(other) != 0 {
		t.Fatal("group memory leak")
	}
	if _, err = db.History("b", "r1"); err == nil {
		t.Fatal("history leaked")
	}
	runs, _ := db.Runs("a")
	if len(runs) != 1 || runs[0].State.Round != 1 {
		t.Fatal("lost history")
	}
	if err = db.ResetMemory("a", "carry:test"); err != nil {
		t.Fatal(err)
	}
	got, _ = db.Memories("a", "carry:test")
	if len(got) != 0 {
		t.Fatal("reset failed")
	}
}
