package app

import (
	"archive/zip"
	"database/sql"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// BackupAll runs only while the executable's instance lock is held. The caller
// must stop the service first; no live DB or attachment copying is attempted.
func BackupAll(dir, out string) error {
	s, e := OpenStore(dir)
	if e != nil {
		return e
	}
	defer s.Close()
	temp := filepath.Join(dir, "backups", "snapshot-"+ID()+".db")
	defer os.Remove(temp)
	if _, e = s.DB.Exec("VACUUM INTO ?", temp); e != nil {
		return e
	}
	dest, e := os.OpenFile(out, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if e != nil {
		return e
	}
	z := zip.NewWriter(dest)
	add := func(name, source string) error {
		src, e := os.Open(source)
		if e != nil {
			return e
		}
		defer src.Close()
		entry, e := z.Create(name)
		if e != nil {
			return e
		}
		_, e = io.Copy(entry, src)
		return e
	}
	e = add("notes.db", temp)
	if e == nil {
		e = filepath.WalkDir(filepath.Join(dir, "attachments"), func(p string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			if !validID(d.Name()) || d.Type()&os.ModeSymlink != 0 {
				return errors.New("unexpected attachment path")
			}
			return add("attachments/"+d.Name(), p)
		})
	}
	ze := z.Close()
	fe := dest.Close()
	if e = errors.Join(e, ze, fe); e != nil {
		_ = os.Remove(out)
	}
	return e
}
func RestoreAll(source, dir string) error {
	if _, e := os.Stat(filepath.Join(dir, "notes.db")); e == nil {
		return errors.New("restore requires a destination without a database")
	}
	z, e := zip.OpenReader(source)
	if e != nil {
		return e
	}
	defer z.Close()
	if len(z.File) > 100000 {
		return errors.New("too many backup entries")
	}
	if e = os.MkdirAll(dir, 0700); e != nil {
		return e
	}
	stage, e := os.MkdirTemp(dir, "restore-")
	if e != nil {
		return e
	}
	defer func() {
		rel, err := filepath.Rel(dir, stage)
		if err == nil && !strings.HasPrefix(rel, "..") && rel != "." {
			_ = os.RemoveAll(stage)
		}
	}()
	seen := map[string]bool{}
	var total uint64
	for _, f := range z.File {
		if seen[f.Name] {
			return errors.New("duplicate backup entry")
		}
		seen[f.Name] = true
		valid := f.Name == "notes.db" || (strings.HasPrefix(f.Name, "attachments/") && validID(strings.TrimPrefix(f.Name, "attachments/")))
		if !valid || f.Mode()&os.ModeSymlink != 0 {
			return errors.New("unexpected backup path")
		}
		total += f.UncompressedSize64
		if total > 20*1024*1024*1024 {
			return errors.New("backup exceeds 20 GiB restore limit")
		}
		p := filepath.Join(stage, filepath.FromSlash(f.Name))
		if e = os.MkdirAll(filepath.Dir(p), 0700); e != nil {
			return e
		}
		src, err := f.Open()
		if err != nil {
			return err
		}
		dst, err := os.OpenFile(p, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if err != nil {
			src.Close()
			return err
		}
		_, err = io.Copy(dst, io.LimitReader(src, int64(f.UncompressedSize64)+1))
		ce := errors.Join(src.Close(), dst.Close())
		if err = errors.Join(err, ce); err != nil {
			return err
		}
	}
	if !seen["notes.db"] {
		return errors.New("backup has no database")
	}
	db, e := sql.Open("sqlite", filepath.Join(stage, "notes.db"))
	if e != nil {
		return e
	}
	var check string
	e = db.QueryRow("PRAGMA integrity_check").Scan(&check)
	if e != nil || check != "ok" {
		db.Close()
		return errors.New("backup database failed integrity check")
	}
	_, e = db.Exec("DELETE FROM sessions; DELETE FROM logins; DELETE FROM invitations;")
	if e != nil {
		db.Close()
		return e
	}
	if e = db.Close(); e != nil {
		return e
	}
	if _, e = os.Stat(filepath.Join(dir, "attachments")); e == nil {
		entries, err := os.ReadDir(filepath.Join(dir, "attachments"))
		if err != nil || len(entries) > 0 {
			return errors.New("destination attachments must be empty")
		}
		if e = os.Remove(filepath.Join(dir, "attachments")); e != nil {
			return e
		}
	}
	if _, e = os.Stat(filepath.Join(stage, "attachments")); os.IsNotExist(e) {
		if e = os.Mkdir(filepath.Join(stage, "attachments"), 0700); e != nil {
			return e
		}
	}
	if e = os.Rename(filepath.Join(stage, "attachments"), filepath.Join(dir, "attachments")); e != nil {
		return e
	}
	return os.Rename(filepath.Join(stage, "notes.db"), filepath.Join(dir, "notes.db"))
}
