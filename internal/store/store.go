// Package store persists scoped run history and private memories in SQLite.
package store

import (
	"database/sql"
	"encoding/json"
	"errors"
	_ "modernc.org/sqlite"
	"os"
	"path/filepath"
	"somas/pkg/game"
)

type Store struct{ DB *sql.DB }
type Run struct {
	ID      string        `json:"id"`
	Group   string        `json:"-"`
	Config  game.Config   `json:"config"`
	State   game.Snapshot `json:"state"`
	Error   string        `json:"error,omitempty"`
	Playing bool          `json:"playing"`
}

func Open(path string) (*Store, error) {
	if path != ":memory:" {
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			return nil, err
		}
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	_, err = db.Exec(`PRAGMA journal_mode=WAL; PRAGMA busy_timeout=5000;
 CREATE TABLE IF NOT EXISTS runs(id TEXT PRIMARY KEY,owner TEXT NOT NULL,config TEXT NOT NULL,state TEXT NOT NULL,error TEXT NOT NULL DEFAULT '');
 CREATE TABLE IF NOT EXISTS history(run TEXT NOT NULL,seq INTEGER NOT NULL,state TEXT NOT NULL,PRIMARY KEY(run,seq));
 CREATE TABLE IF NOT EXISTS memory(owner TEXT NOT NULL,scope TEXT NOT NULL,agent TEXT NOT NULL,data TEXT NOT NULL,PRIMARY KEY(owner,scope,agent));`)
	if err != nil {
		db.Close()
		return nil, err
	}
	return &Store{db}, nil
}
func (s *Store) Close() error { return s.DB.Close() }
func (s *Store) Memories(owner, scope string) (map[string]map[string]string, error) {
	rows, err := s.DB.Query("SELECT agent,data FROM memory WHERE owner=? AND scope=?", owner, scope)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	m := map[string]map[string]string{}
	for rows.Next() {
		var id, data string
		if err = rows.Scan(&id, &data); err != nil {
			return nil, err
		}
		var a map[string]string
		if err = json.Unmarshal([]byte(data), &a); err != nil {
			return nil, err
		}
		m[id] = a
	}
	return m, rows.Err()
}
func (s *Store) ResetMemory(owner, scope string) error {
	_, err := s.DB.Exec("DELETE FROM memory WHERE owner=? AND scope=?", owner, scope)
	return err
}
func (s *Store) Save(r Run, scope string, mem map[string]map[string]string, seq int) error {
	cfg, err := json.Marshal(r.Config)
	if err != nil {
		return err
	}
	state, err := json.Marshal(r.State)
	if err != nil {
		return err
	}
	tx, err := s.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	_, err = tx.Exec("INSERT INTO runs(id,owner,config,state,error) VALUES(?,?,?,?,?) ON CONFLICT(id) DO UPDATE SET state=excluded.state,error=excluded.error", r.ID, r.Group, string(cfg), string(state), r.Error)
	if err != nil {
		return err
	}
	_, err = tx.Exec("INSERT INTO history(run,seq,state) VALUES(?,?,?)", r.ID, seq, string(state))
	if err != nil {
		return err
	}
	for id, m := range mem {
		b, e := json.Marshal(m)
		if e != nil {
			return e
		}
		if _, err = tx.Exec("INSERT INTO memory(owner,scope,agent,data) VALUES(?,?,?,?) ON CONFLICT(owner,scope,agent) DO UPDATE SET data=excluded.data", r.Group, scope, id, string(b)); err != nil {
			return err
		}
	}
	return tx.Commit()
}
func (s *Store) Runs(owner string) ([]Run, error) {
	rows, err := s.DB.Query("SELECT id,config,state,error FROM runs WHERE owner=? ORDER BY rowid DESC", owner)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Run{}
	for rows.Next() {
		r := Run{Group: owner}
		var c, st string
		if err = rows.Scan(&r.ID, &c, &st, &r.Error); err != nil {
			return nil, err
		}
		if err = json.Unmarshal([]byte(c), &r.Config); err != nil {
			return nil, err
		}
		if err = json.Unmarshal([]byte(st), &r.State); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
func (s *Store) History(owner, id string) ([]game.Snapshot, error) {
	var n int
	if err := s.DB.QueryRow("SELECT count(*) FROM runs WHERE owner=? AND id=?", owner, id).Scan(&n); err != nil {
		return nil, err
	}
	if n == 0 {
		return nil, errors.New("run not found")
	}
	rows, err := s.DB.Query("SELECT state FROM history WHERE run=? ORDER BY seq", id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []game.Snapshot{}
	for rows.Next() {
		var data string
		if err = rows.Scan(&data); err != nil {
			return nil, err
		}
		var st game.Snapshot
		if err = json.Unmarshal([]byte(data), &st); err != nil {
			return nil, err
		}
		out = append(out, st)
	}
	return out, rows.Err()
}
