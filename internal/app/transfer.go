package app

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	htmltomarkdown "github.com/JohannesKaufmann/html-to-markdown/v2"
	"github.com/microcosm-cc/bluemonday"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"html"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type Backup struct {
	Format      string       `json:"format"`
	Version     int          `json:"version"`
	Notes       []Note       `json:"notes"`
	Notebooks   []Notebook   `json:"notebooks"`
	Attachments []Attachment `json:"attachments"`
}

func (a *App) Export(w http.ResponseWriter, r *http.Request) {
	a.writeMu.Lock()
	defer a.writeMu.Unlock()
	a.fileMu.Lock()
	defer a.fileMu.Unlock()
	uid := who(r).User.ID
	b := Backup{Format: "oneworkspace-notes", Version: 1, Notes: []Note{}, Notebooks: []Notebook{}, Attachments: []Attachment{}}
	rows, e := a.Store.DB.Query("SELECT "+noteCols+" FROM notes n WHERE n.owner=? ORDER BY n.created", uid)
	if e != nil {
		problem(w, 500, "Could not export notes")
		return
	}
	for rows.Next() {
		n, err := scanNote(rows)
		if err != nil {
			rows.Close()
			problem(w, 500, "Could not export notes")
			return
		}
		b.Notes = append(b.Notes, n)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		problem(w, 500, "Could not export notes")
		return
	}
	rows, e = a.Store.DB.Query("SELECT id,owner,name,parent_id FROM notebooks WHERE owner=?", uid)
	if e != nil {
		problem(w, 500, "Could not export notebooks")
		return
	}
	for rows.Next() {
		var n Notebook
		_ = rows.Scan(&n.ID, &n.Owner, &n.Name, &n.Parent)
		b.Notebooks = append(b.Notebooks, n)
	}
	rows.Close()
	rows, e = a.Store.DB.Query("SELECT a.id,a.note_id,a.name,a.mime,a.size FROM attachments a JOIN notes n ON n.id=a.note_id WHERE n.owner=?", uid)
	if e != nil {
		problem(w, 500, "Could not export attachments")
		return
	}
	for rows.Next() {
		var at Attachment
		_ = rows.Scan(&at.ID, &at.NoteID, &at.Name, &at.MIME, &at.Size)
		b.Attachments = append(b.Attachments, at)
	}
	rows.Close()
	// Preflight resources before sending response headers so failures do not look like successful backups.
	for _, at := range b.Attachments {
		info, err := os.Stat(a.filePath(at.ID))
		if err != nil || info.Size() != at.Size {
			problem(w, 500, "An attachment is missing; backup was not created")
			return
		}
	}
	format := r.URL.Query().Get("format")
	if format != "" && format != "json" && format != "markdown" && format != "html" {
		problem(w, 400, "Unknown export format")
		return
	}
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", `attachment; filename="notes-backup.zip"`)
	z := zip.NewWriter(w)
	defer z.Close()
	write := func(name string, data []byte) error {
		f, e := z.Create(name)
		if e != nil {
			return e
		}
		_, e = f.Write(data)
		return e
	}
	data, _ := json.MarshalIndent(b, "", "  ")
	if write("backup.json", data) != nil {
		return
	}
	md := goldmark.New(goldmark.WithExtensions(extension.GFM))
	sanitizer := bluemonday.UGCPolicy()
	for _, n := range b.Notes {
		body := n.Body
		for _, other := range b.Notes {
			suffix := ".md"
			if format == "html" {
				suffix = ".html"
			}
			body = strings.ReplaceAll(body, ":/"+other.ID, "./"+other.ID+suffix)
		}
		for _, at := range b.Attachments {
			body = strings.ReplaceAll(body, ":/"+at.ID, "../attachments/"+at.ID)
			body = strings.ReplaceAll(body, a.Config.BasePath+"/api/attachments/"+at.ID+"?inline=1", "../attachments/"+at.ID)
		}
		if format == "html" {
			var out bytes.Buffer
			if md.Convert([]byte(body), &out) != nil {
				return
			}
			doc := "<!doctype html><meta charset=utf-8><title>" + html.EscapeString(n.Title) + "</title><h1>" + html.EscapeString(n.Title) + "</h1>" + sanitizer.Sanitize(out.String())
			if write("html/"+n.ID+".html", []byte(doc)) != nil {
				return
			}
		} else if format == "markdown" {
			if write("markdown/"+n.ID+".md", []byte("# "+n.Title+"\n\n"+body)) != nil {
				return
			}
		}
	}
	for _, at := range b.Attachments {
		f, err := os.Open(a.filePath(at.ID))
		if err != nil {
			return
		}
		dest, err := z.Create("attachments/" + at.ID)
		if err == nil {
			_, err = io.Copy(dest, f)
		}
		f.Close()
		if err != nil {
			return
		}
	}
}
func (a *App) Import(w http.ResponseWriter, r *http.Request) {
	a.fileMu.Lock()
	defer a.fileMu.Unlock()
	r.Body = http.MaxBytesReader(w, r.Body, a.Config.UploadMB*1024*1024+1024*1024)
	mr, e := r.MultipartReader()
	if e != nil {
		problem(w, 400, "Expected a file upload")
		return
	}
	part, e := mr.NextPart()
	if e != nil || part.FormName() != "file" {
		problem(w, 400, "Missing import file")
		return
	}
	defer part.Close()
	data, e := io.ReadAll(io.LimitReader(part, a.Config.UploadMB*1024*1024+1))
	if e != nil || int64(len(data)) > a.Config.UploadMB*1024*1024 {
		problem(w, 413, "Import exceeds upload limit")
		return
	}
	var backup Backup
	files := map[string][]byte{}
	ext := strings.ToLower(filepath.Ext(part.FileName()))
	switch ext {
	case ".html", ".htm":
		if len(data) > 2*1024*1024 {
			problem(w, 413, "HTML note exceeds the 2 MB content limit")
			return
		}
		var converted string
		converted, e = htmltomarkdown.ConvertString(bluemonday.UGCPolicy().Sanitize(string(data)))
		backup = Backup{Format: "oneworkspace-notes", Version: 1, Notes: []Note{{ID: ID(), Title: strings.TrimSuffix(cleanName(part.FileName()), ext), Body: converted, Tags: []string{}}}}
	case ".md", ".markdown":
		backup = Backup{Format: "oneworkspace-notes", Version: 1, Notes: []Note{{ID: ID(), Title: strings.TrimSuffix(cleanName(part.FileName()), ext), Body: string(data), Tags: []string{}}}}
	case ".json":
		e = json.Unmarshal(data, &backup)
	case ".zip":
		backup, files, e = readBackupZip(data)
	default:
		problem(w, 400, "Import Markdown, HTML or a notes JSON/ZIP backup. JEX and encrypted sync items are not supported.")
		return
	}
	if e != nil {
		problem(w, 400, "Invalid backup archive")
		return
	}
	count, e := a.restoreUser(who(r).User.ID, backup, files)
	if e != nil {
		problem(w, 400, e.Error())
		return
	}
	respond(w, 201, map[string]int{"imported": count})
}
func readBackupZip(data []byte) (Backup, map[string][]byte, error) {
	var b Backup
	out := map[string][]byte{}
	z, e := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if e != nil {
		return b, nil, e
	}
	if len(z.File) > 15000 {
		return b, nil, errors.New("too many archive entries")
	}
	var total uint64
	for _, f := range z.File {
		if f.Name != "backup.json" && !(strings.HasPrefix(f.Name, "attachments/") && validID(strings.TrimPrefix(f.Name, "attachments/"))) {
			continue
		}
		if _, exists := out[f.Name]; exists {
			return b, nil, errors.New("duplicate archive entry")
		}
		total += f.UncompressedSize64
		if total > 100*1024*1024 || f.UncompressedSize64 > 25*1024*1024 {
			return b, nil, errors.New("expanded archive exceeds limit")
		}
		src, err := f.Open()
		if err != nil {
			return b, nil, err
		}
		body, err := io.ReadAll(io.LimitReader(src, 25*1024*1024+1))
		src.Close()
		if err != nil || len(body) > 25*1024*1024 {
			return b, nil, errors.New("invalid archive entry")
		}
		out[f.Name] = body
	}
	e = json.Unmarshal(out["backup.json"], &b)
	return b, out, e
}
func (a *App) restoreUser(uid string, b Backup, files map[string][]byte) (int, error) {
	fail := func(s string) (int, error) { return 0, errors.New(s) }
	if b.Format != "oneworkspace-notes" || b.Version != 1 || len(b.Notes) > 5000 || len(b.Notebooks) > 1000 {
		return fail("Unsupported backup format or item count")
	}
	mapping := map[string]string{}
	books := map[string]Notebook{}
	notes := map[string]bool{}
	var size int64
	for _, book := range b.Notebooks {
		if !validID(book.ID) || len(book.Name) > 120 || strings.TrimSpace(book.Name) == "" || mapping[book.ID] != "" {
			return fail("Invalid notebook metadata")
		}
		mapping[book.ID] = ID()
		books[book.ID] = book
	}
	for _, book := range b.Notebooks {
		seen := map[string]bool{book.ID: true}
		p := book.Parent
		for p != "" {
			v, ok := books[p]
			if !ok || seen[p] {
				return fail("Invalid notebook hierarchy")
			}
			seen[p] = true
			p = v.Parent
		}
	}
	for _, n := range b.Notes {
		if !validID(n.ID) || mapping[n.ID] != "" || validation(n) != nil {
			return fail("Invalid note metadata")
		}
		if n.Notebook != "" {
			if _, ok := books[n.Notebook]; !ok {
				return fail("Missing notebook")
			}
		}
		mapping[n.ID] = ID()
		notes[n.ID] = true
		size += int64(len(n.Title) + len(n.Body))
	}
	for _, at := range b.Attachments {
		if !validID(at.ID) || mapping[at.ID] != "" || !notes[at.NoteID] {
			return fail("Invalid attachment metadata")
		}
		v, ok := files["attachments/"+at.ID]
		if !ok || int64(len(v)) != at.Size {
			return fail("Missing attachment data")
		}
		size += at.Size
		mapping[at.ID] = ID()
	}
	if !a.Store.WithinQuota(uid, size) {
		return fail("Storage quota exceeded")
	}
	tx, e := a.Store.DB.Begin()
	if e != nil {
		return 0, e
	}
	defer tx.Rollback()
	written := []string{}
	success := false
	defer func() {
		if !success {
			for _, id := range written {
				a.removeFile(id)
			}
		}
	}()
	for _, book := range b.Notebooks {
		if _, e = tx.Exec("INSERT INTO notebooks VALUES(?,?,?,?)", mapping[book.ID], uid, book.Name, mapping[book.Parent]); e != nil {
			return fail("Could not import notebooks")
		}
	}
	pairs := make([]string, 0, len(mapping)*2)
	for old, id := range mapping {
		pairs = append(pairs, ":/"+old, ":/"+id, "#note="+old, "#note="+id)
	}
	replacer := strings.NewReplacer(pairs...)
	for _, n := range b.Notes {
		n.Body = replacer.Replace(n.Body)
		if n.Tags == nil {
			n.Tags = []string{}
		}
		if n.Created <= 0 {
			n.Created = now()
		}
		if n.Updated <= 0 {
			n.Updated = n.Created
		}
		_, e = tx.Exec(`INSERT INTO notes(id,owner,notebook_id,title,body,tags,starred,pinned,archived,deleted,template,reminder,created,updated,revision)VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,1)`, mapping[n.ID], uid, mapping[n.Notebook], n.Title, n.Body, jsonString(n.Tags), n.Starred, n.Pinned, n.Archived, n.Deleted, n.Template, n.Reminder, n.Created, n.Updated)
		if e != nil {
			return fail("Could not import notes")
		}
	}
	for _, at := range b.Attachments {
		id := mapping[at.ID]
		data := files["attachments/"+at.ID]
		if e = os.WriteFile(a.filePath(id), data, 0600); e != nil {
			return fail("Could not import attachment")
		}
		written = append(written, id)
		_, e = tx.Exec("INSERT INTO attachments VALUES(?,?,?,?,?)", id, mapping[at.NoteID], cleanName(at.Name), http.DetectContentType(data), len(data))
		if e != nil {
			return fail("Could not import attachment metadata")
		}
	}
	if e = tx.Commit(); e != nil {
		return fail("Could not commit import")
	}
	success = true
	a.Store.Audit(uid, "import", fmt.Sprint(len(b.Notes)))
	return len(b.Notes), nil
}
