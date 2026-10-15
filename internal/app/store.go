package app

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	_ "modernc.org/sqlite"
	"os"
	"path/filepath"
	"time"
)

func ID() string {
	b := make([]byte, 16)
	if _, e := rand.Read(b); e != nil {
		panic(e)
	}
	return hex.EncodeToString(b)
}
func secret() string         { return ID() + ID() }
func digest(s string) string { h := sha256.Sum256([]byte(s)); return hex.EncodeToString(h[:]) }
func now() int64             { return time.Now().UnixMilli() }

type Store struct {
	DB  *sql.DB
	Dir string
}
type Note struct {
	ID         string   `json:"id"`
	Owner      string   `json:"owner"`
	Notebook   string   `json:"notebook_id"`
	Title      string   `json:"title"`
	Body       string   `json:"body"`
	Tags       []string `json:"tags"`
	Starred    bool     `json:"starred"`
	Pinned     bool     `json:"pinned"`
	Archived   bool     `json:"archived"`
	Deleted    bool     `json:"deleted"`
	Template   bool     `json:"template"`
	Reminder   int64    `json:"reminder"`
	Created    int64    `json:"created"`
	Updated    int64    `json:"updated"`
	Revision   int      `json:"revision"`
	Permission string   `json:"permission"`
}
type Notebook struct {
	ID         string `json:"id"`
	Owner      string `json:"owner"`
	Name       string `json:"name"`
	Parent     string `json:"parent_id"`
	Permission string `json:"permission"`
}
type User struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Email   string `json:"email"`
	Avatar  string `json:"avatar"`
	Admin   bool   `json:"admin"`
	Subject string `json:"-"`
}

const schema = `
CREATE TABLE IF NOT EXISTS migrations(version INTEGER PRIMARY KEY, applied INTEGER NOT NULL);
CREATE TABLE IF NOT EXISTS users(id TEXT PRIMARY KEY, issuer TEXT NOT NULL, subject TEXT NOT NULL, name TEXT NOT NULL, email TEXT NOT NULL, avatar TEXT NOT NULL, quota_mb INTEGER NOT NULL, disabled INTEGER NOT NULL DEFAULT 0, UNIQUE(issuer,subject));
CREATE TABLE IF NOT EXISTS sessions(hash TEXT PRIMARY KEY,user_id TEXT NOT NULL REFERENCES users(id),csrf TEXT NOT NULL,expires INTEGER NOT NULL);
CREATE INDEX IF NOT EXISTS sessions_expiry ON sessions(expires);
CREATE TABLE IF NOT EXISTS logins(state TEXT PRIMARY KEY,binding TEXT NOT NULL,nonce TEXT NOT NULL,verifier TEXT NOT NULL,expires INTEGER NOT NULL);
CREATE TABLE IF NOT EXISTS notebooks(id TEXT PRIMARY KEY,owner TEXT NOT NULL REFERENCES users(id),name TEXT NOT NULL,parent_id TEXT NOT NULL DEFAULT '');
CREATE INDEX IF NOT EXISTS notebooks_owner ON notebooks(owner);
CREATE TABLE IF NOT EXISTS notes(id TEXT PRIMARY KEY,owner TEXT NOT NULL REFERENCES users(id),notebook_id TEXT NOT NULL DEFAULT '',title TEXT NOT NULL,body TEXT NOT NULL,tags TEXT NOT NULL DEFAULT '[]',starred INTEGER NOT NULL DEFAULT 0,pinned INTEGER NOT NULL DEFAULT 0,archived INTEGER NOT NULL DEFAULT 0,deleted INTEGER NOT NULL DEFAULT 0,template INTEGER NOT NULL DEFAULT 0,reminder INTEGER NOT NULL DEFAULT 0,created INTEGER NOT NULL,updated INTEGER NOT NULL,revision INTEGER NOT NULL DEFAULT 1);
CREATE INDEX IF NOT EXISTS notes_owner_updated ON notes(owner,updated DESC);
CREATE INDEX IF NOT EXISTS notes_notebook ON notes(notebook_id);
CREATE VIRTUAL TABLE IF NOT EXISTS notes_fts USING fts5(title,body,tags,content=notes,content_rowid=rowid,tokenize='unicode61');
CREATE TRIGGER IF NOT EXISTS notes_ai AFTER INSERT ON notes BEGIN INSERT INTO notes_fts(rowid,title,body,tags)VALUES(new.rowid,new.title,new.body,new.tags);END;
CREATE TRIGGER IF NOT EXISTS notes_ad AFTER DELETE ON notes BEGIN INSERT INTO notes_fts(notes_fts,rowid,title,body,tags)VALUES('delete',old.rowid,old.title,old.body,old.tags);END;
CREATE TRIGGER IF NOT EXISTS notes_au AFTER UPDATE ON notes BEGIN INSERT INTO notes_fts(notes_fts,rowid,title,body,tags)VALUES('delete',old.rowid,old.title,old.body,old.tags);INSERT INTO notes_fts(rowid,title,body,tags)VALUES(new.rowid,new.title,new.body,new.tags);END;
CREATE TABLE IF NOT EXISTS versions(note_id TEXT NOT NULL REFERENCES notes(id) ON DELETE CASCADE,revision INTEGER NOT NULL,title TEXT NOT NULL,body TEXT NOT NULL,created INTEGER NOT NULL,PRIMARY KEY(note_id,revision));
CREATE TABLE IF NOT EXISTS attachments(id TEXT PRIMARY KEY,note_id TEXT NOT NULL REFERENCES notes(id) ON DELETE CASCADE,name TEXT NOT NULL,mime TEXT NOT NULL,size INTEGER NOT NULL);
CREATE TABLE IF NOT EXISTS shares(notebook_id TEXT NOT NULL REFERENCES notebooks(id) ON DELETE CASCADE,user_id TEXT NOT NULL REFERENCES users(id),role TEXT NOT NULL CHECK(role IN ('viewer','editor')),PRIMARY KEY(notebook_id,user_id));
CREATE TABLE IF NOT EXISTS invitations(hash TEXT PRIMARY KEY,notebook_id TEXT NOT NULL REFERENCES notebooks(id) ON DELETE CASCADE,role TEXT NOT NULL CHECK(role IN ('viewer','editor')),expires INTEGER NOT NULL);
CREATE TABLE IF NOT EXISTS audit(id INTEGER PRIMARY KEY,at INTEGER NOT NULL,actor TEXT NOT NULL,event TEXT NOT NULL,target TEXT NOT NULL);
INSERT OR IGNORE INTO migrations VALUES(1,0);
`

func OpenStore(dir string) (*Store, error) {
	if e := os.MkdirAll(filepath.Join(dir, "attachments"), 0700); e != nil {
		return nil, e
	}
	if e := os.MkdirAll(filepath.Join(dir, "backups"), 0700); e != nil {
		return nil, e
	}
	db, e := sql.Open("sqlite", filepath.Join(dir, "notes.db"))
	if e != nil {
		return nil, e
	}
	db.SetMaxOpenConns(1)
	fail := func(e error) (*Store, error) { db.Close(); return nil, e }
	var migrationTable int
	if e = db.QueryRow("SELECT count(*) FROM sqlite_master WHERE type='table' AND name='migrations'").Scan(&migrationTable); e != nil {
		return fail(e)
	}
	if migrationTable != 0 {
		var version int
		if e = db.QueryRow("SELECT COALESCE(MAX(version),0) FROM migrations").Scan(&version); e != nil {
			return fail(e)
		}
		if version > 1 {
			return fail(errors.New("database was created by a newer version of notes; do not downgrade"))
		}
	} else {
		var tables int
		if e = db.QueryRow("SELECT count(*) FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%'").Scan(&tables); e != nil {
			return fail(e)
		}
		if tables > 0 {
			return fail(errors.New("database belongs to another application"))
		}
	}
	if _, e = db.Exec("PRAGMA journal_mode=WAL; PRAGMA foreign_keys=ON; PRAGMA busy_timeout=5000; PRAGMA synchronous=FULL;"); e != nil {
		return fail(e)
	}
	tx, e := db.Begin()
	if e != nil {
		return fail(e)
	}
	if _, e = tx.Exec(schema); e != nil {
		tx.Rollback()
		return fail(e)
	}
	if e = tx.Commit(); e != nil {
		return fail(e)
	}
	return &Store{db, dir}, nil
}
func (s *Store) Close() error {
	_, e := s.DB.Exec("PRAGMA wal_checkpoint(TRUNCATE)")
	ce := s.DB.Close()
	return errors.Join(e, ce)
}

const noteCols = `n.id,n.owner,n.notebook_id,n.title,n.body,n.tags,n.starred,n.pinned,n.archived,n.deleted,n.template,n.reminder,n.created,n.updated,n.revision`
const accessible = `(n.owner=? OR EXISTS(SELECT 1 FROM shares sh WHERE sh.notebook_id=n.notebook_id AND sh.user_id=?))`

type scanner interface{ Scan(...any) error }

func scanNote(r scanner) (Note, error) {
	var n Note
	var tags string
	e := r.Scan(&n.ID, &n.Owner, &n.Notebook, &n.Title, &n.Body, &tags, &n.Starred, &n.Pinned, &n.Archived, &n.Deleted, &n.Template, &n.Reminder, &n.Created, &n.Updated, &n.Revision)
	if e == nil {
		e = json.Unmarshal([]byte(tags), &n.Tags)
	}
	if n.Tags == nil {
		n.Tags = []string{}
	}
	return n, e
}
func (s *Store) Note(uid, id string) (Note, error) {
	n, e := scanNote(s.DB.QueryRow("SELECT "+noteCols+" FROM notes n WHERE n.id=? AND "+accessible, id, uid, uid))
	if e != nil {
		return n, e
	}
	n.Permission = "owner"
	if n.Owner != uid {
		e = s.DB.QueryRow("SELECT role FROM shares WHERE notebook_id=? AND user_id=?", n.Notebook, uid).Scan(&n.Permission)
	}
	return n, e
}
func (s *Store) CanNotebook(uid, id string, write bool) bool {
	if id == "" {
		return true
	}
	var owner string
	if s.DB.QueryRow("SELECT owner FROM notebooks WHERE id=?", id).Scan(&owner) != nil {
		return false
	}
	if owner == uid {
		return true
	}
	var role string
	if s.DB.QueryRow("SELECT role FROM shares WHERE notebook_id=? AND user_id=?", id, uid).Scan(&role) != nil {
		return false
	}
	return !write || role == "editor"
}
func (s *Store) Provision(issuer, sub, name, email, avatar string, quota int64) (User, error) {
	u := User{ID: ID(), Name: name, Email: email, Avatar: avatar, Subject: sub}
	if name == "" {
		u.Name = "Notes user"
	}
	_, e := s.DB.Exec(`INSERT INTO users(id,issuer,subject,name,email,avatar,quota_mb)VALUES(?,?,?,?,?,?,?) ON CONFLICT(issuer,subject) DO UPDATE SET name=excluded.name,email=excluded.email,avatar=excluded.avatar`, u.ID, issuer, sub, u.Name, email, avatar, quota)
	if e != nil {
		return u, e
	}
	var disabled bool
	e = s.DB.QueryRow("SELECT id,disabled FROM users WHERE issuer=? AND subject=?", issuer, sub).Scan(&u.ID, &disabled)
	if disabled {
		return u, errors.New("account is disabled")
	}
	return u, e
}
func (s *Store) WithinQuota(uid string, extra int64) bool {
	var used, quota int64
	e := s.DB.QueryRow(`SELECT COALESCE((SELECT SUM(length(CAST(body AS BLOB))+length(CAST(title AS BLOB))) FROM notes WHERE owner=u.id),0)+COALESCE((SELECT SUM(a.size) FROM attachments a JOIN notes n ON n.id=a.note_id WHERE n.owner=u.id),0)+COALESCE((SELECT SUM(length(CAST(v.body AS BLOB))+length(CAST(v.title AS BLOB))) FROM versions v JOIN notes n ON n.id=v.note_id WHERE n.owner=u.id),0),quota_mb FROM users u WHERE id=?`, uid).Scan(&used, &quota)
	return e == nil && used+extra <= quota*1024*1024
}
func (s *Store) Audit(actor, event, target string) {
	_, _ = s.DB.Exec("INSERT INTO audit(at,actor,event,target)VALUES(?,?,?,?)", now(), actor, event, target)
}
func validID(v string) bool {
	if len(v) != 32 {
		return false
	}
	_, e := hex.DecodeString(v)
	return e == nil
}
func jsonString(v any) string { b, _ := json.Marshal(v); return string(b) }
func validation(n Note) error {
	if len(n.Title) > 512 || len(n.Body) > 2*1024*1024 || len(n.Tags) > 32 {
		return errors.New("note exceeds title, body or tag limits")
	}
	for _, t := range n.Tags {
		if len(t) > 64 || t == "" {
			return fmt.Errorf("invalid tag")
		}
	}
	if n.Notebook != "" && !validID(n.Notebook) {
		return errors.New("invalid notebook")
	}
	return nil
}
