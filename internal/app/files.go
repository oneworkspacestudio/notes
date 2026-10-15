package app

import (
	"bytes"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type Attachment struct {
	ID     string `json:"id"`
	NoteID string `json:"note_id"`
	Name   string `json:"name"`
	MIME   string `json:"mime"`
	Size   int64  `json:"size"`
}

func (a *App) filePath(id string) string {
	if !validID(id) {
		return ""
	}
	return filepath.Join(a.Store.Dir, "attachments", id)
}
func (a *App) removeFile(id string) {
	if p := a.filePath(id); p != "" {
		if e := os.Remove(p); e != nil && !os.IsNotExist(e) {
			a.Logger.Print("attachment cleanup failed")
		}
	}
}
func cleanName(s string) string {
	s = filepath.Base(strings.ReplaceAll(s, "\\", "/"))
	s = strings.Map(func(r rune) rune {
		if r < 32 || strings.ContainsRune(`<>:"/\|?*`, r) {
			return '_'
		}
		return r
	}, s)
	if len(s) > 180 {
		s = s[:180]
	}
	if s == "" || s == "." {
		return "attachment"
	}
	return s
}
func (a *App) Upload(w http.ResponseWriter, r *http.Request) {
	a.fileMu.Lock()
	defer a.fileMu.Unlock()
	n, e := a.Store.Note(who(r).User.ID, r.PathValue("id"))
	if e != nil {
		problem(w, 404, "Note not found")
		return
	}
	if n.Permission == "viewer" || n.Deleted {
		problem(w, 403, "Note is not writable")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, a.Config.UploadMB*1024*1024+1024*1024)
	reader, e := r.MultipartReader()
	if e != nil {
		problem(w, 400, "Expected a file upload")
		return
	}
	part, e := reader.NextPart()
	if e != nil || part.FormName() != "file" {
		problem(w, 400, "Missing file")
		return
	}
	defer part.Close()
	b, e := io.ReadAll(io.LimitReader(part, a.Config.UploadMB*1024*1024+1))
	if e != nil || int64(len(b)) > a.Config.UploadMB*1024*1024 || len(b) == 0 {
		problem(w, 413, "Upload is empty or exceeds the size limit")
		return
	}
	kind := http.DetectContentType(b)
	if strings.HasPrefix(kind, "image/") {
		cfg, _, err := image.DecodeConfig(bytes.NewReader(b))
		if err != nil || int64(cfg.Width)*int64(cfg.Height) > 50000000 {
			problem(w, 400, "Invalid or oversized image; use PNG, JPEG or GIF")
			return
		}
	}
	if kind == "text/html; charset=utf-8" || kind == "image/svg+xml" || strings.HasSuffix(strings.ToLower(part.FileName()), ".svg") {
		problem(w, 400, "Active HTML and SVG attachments are not supported")
		return
	}
	if !a.Store.WithinQuota(n.Owner, int64(len(b))) {
		problem(w, 413, "Storage quota exceeded")
		return
	}
	at := Attachment{ID: ID(), NoteID: n.ID, Name: cleanName(part.FileName()), MIME: kind, Size: int64(len(b))}
	if e = os.WriteFile(a.filePath(at.ID), b, 0600); e != nil {
		problem(w, 500, "Could not store attachment")
		return
	}
	if _, e = a.Store.DB.Exec("INSERT INTO attachments VALUES(?,?,?,?,?)", at.ID, at.NoteID, at.Name, at.MIME, at.Size); e != nil {
		a.removeFile(at.ID)
		problem(w, 500, "Could not save attachment")
		return
	}
	respond(w, 201, at)
}
func (a *App) attachment(uid, id string) (Attachment, error) {
	var at Attachment
	e := a.Store.DB.QueryRow("SELECT id,note_id,name,mime,size FROM attachments WHERE id=?", id).Scan(&at.ID, &at.NoteID, &at.Name, &at.MIME, &at.Size)
	if e == nil {
		_, e = a.Store.Note(uid, at.NoteID)
	}
	return at, e
}
func (a *App) Download(w http.ResponseWriter, r *http.Request) {
	at, e := a.attachment(who(r).User.ID, r.PathValue("id"))
	if e != nil {
		problem(w, 404, "Attachment not found")
		return
	}
	kind := "attachment"
	if r.URL.Query().Get("inline") == "1" && (at.MIME == "image/png" || at.MIME == "image/jpeg" || at.MIME == "image/gif") {
		kind = "inline"
	}
	w.Header().Set("Content-Type", at.MIME)
	w.Header().Set("Content-Disposition", mime.FormatMediaType(kind, map[string]string{"filename": at.Name}))
	http.ServeFile(w, r, a.filePath(at.ID))
}
func (a *App) Attachments(w http.ResponseWriter, r *http.Request) {
	n, e := a.Store.Note(who(r).User.ID, r.PathValue("id"))
	if e != nil {
		problem(w, 404, "Note not found")
		return
	}
	rows, e := a.Store.DB.Query("SELECT id,note_id,name,mime,size FROM attachments WHERE note_id=? ORDER BY name", n.ID)
	if e != nil {
		problem(w, 500, "Could not load attachments")
		return
	}
	defer rows.Close()
	out := []Attachment{}
	for rows.Next() {
		var at Attachment
		_ = rows.Scan(&at.ID, &at.NoteID, &at.Name, &at.MIME, &at.Size)
		out = append(out, at)
	}
	respond(w, 200, out)
}
func (a *App) DeleteAttachment(w http.ResponseWriter, r *http.Request) {
	a.fileMu.Lock()
	defer a.fileMu.Unlock()
	at, e := a.attachment(who(r).User.ID, r.PathValue("id"))
	if e != nil {
		problem(w, 404, "Attachment not found")
		return
	}
	n, e := a.Store.Note(who(r).User.ID, at.NoteID)
	if e != nil || n.Permission == "viewer" {
		problem(w, 403, "Note is not writable")
		return
	}
	if _, e = a.Store.DB.Exec("DELETE FROM attachments WHERE id=?", at.ID); e != nil {
		problem(w, 500, "Could not delete attachment")
		return
	}
	a.removeFile(at.ID)
	respond(w, 200, map[string]bool{"ok": true})
}
