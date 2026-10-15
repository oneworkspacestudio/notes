package app

import (
	"database/sql"
	"path/filepath"
	"testing"
)

func TestRejectFutureDatabaseAndUnrelatedSQLite(t *testing.T) {
	dir := t.TempDir()
	s, e := OpenStore(dir)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = s.DB.Exec("INSERT INTO migrations VALUES(999,0)"); e != nil {
		t.Fatal(e)
	}
	s.Close()
	if reopened, e := OpenStore(dir); e == nil {
		reopened.Close()
		t.Fatal("downgrade accepted")
	}
	other := t.TempDir()
	db, e := sql.Open("sqlite", filepath.Join(other, "notes.db"))
	if e != nil {
		t.Fatal(e)
	}
	if _, e = db.Exec("CREATE TABLE important(data TEXT)"); e != nil {
		t.Fatal(e)
	}
	db.Close()
	if opened, e := OpenStore(other); e == nil {
		opened.Close()
		t.Fatal("accepted unrelated SQLite database")
	}
}
