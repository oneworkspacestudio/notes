package app

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func (a *App) State(w http.ResponseWriter, r *http.Request) {
	uid := who(r).User.ID
	rows, e := a.Store.DB.Query(`SELECT b.id,b.owner,b.name,b.parent_id,COALESCE(s.role,'owner') FROM notebooks b LEFT JOIN shares s ON s.notebook_id=b.id AND s.user_id=? WHERE b.owner=? OR s.user_id=? ORDER BY b.name COLLATE NOCASE`, uid, uid, uid)
	if e != nil {
		problem(w, 500, "Could not load notebooks")
		return
	}
	books := []Notebook{}
	for rows.Next() {
		var n Notebook
		if e = rows.Scan(&n.ID, &n.Owner, &n.Name, &n.Parent, &n.Permission); e != nil {
			rows.Close()
			problem(w, 500, "Could not load notebooks")
			return
		}
		books = append(books, n)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		problem(w, 500, "Could not load notebooks")
		return
	}
	rows, e = a.Store.DB.Query(`SELECT DISTINCT j.value FROM notes n,json_each(n.tags) j WHERE `+accessible+` AND n.deleted=0 ORDER BY j.value COLLATE NOCASE`, uid, uid)
	if e != nil {
		problem(w, 500, "Could not load tags")
		return
	}
	tags := []string{}
	for rows.Next() {
		var t string
		if e = rows.Scan(&t); e != nil {
			break
		}
		tags = append(tags, t)
	}
	rows.Close()
	var count int
	_ = a.Store.DB.QueryRow("SELECT count(*) FROM notes n WHERE "+accessible+" AND n.deleted=0 AND n.archived=0", uid, uid).Scan(&count)
	respond(w, 200, map[string]any{"notebooks": books, "tags": tags, "count": count})
}
func (a *App) ListNotes(w http.ResponseWriter, r *http.Request) {
	uid := who(r).User.ID
	q := r.URL.Query()
	where := accessible
	args := []any{uid, uid}
	view := q.Get("view")
	if view == "trash" {
		where += " AND n.deleted=1"
	} else {
		where += " AND n.deleted=0"
		if view == "archive" {
			where += " AND n.archived=1"
		} else {
			where += " AND n.archived=0"
		}
	}
	switch view {
	case "starred":
		where += " AND n.starred=1"
	case "reminders":
		where += " AND n.reminder>0"
	case "templates":
		where += " AND n.template=1"
	case "shared":
		where += " AND EXISTS(SELECT 1 FROM shares sh WHERE sh.notebook_id=n.notebook_id)"
	}
	if b := q.Get("notebook"); b != "" {
		where += " AND n.notebook_id=?"
		args = append(args, b)
	}
	if t := q.Get("tag"); t != "" {
		where += " AND EXISTS(SELECT 1 FROM json_each(n.tags) WHERE value=?)"
		args = append(args, t)
	}
	if search := strings.TrimSpace(q.Get("q")); search != "" {
		if len(search) > 200 {
			problem(w, 400, "Search is too long")
			return
		}
		parts := []string{}
		for _, s := range strings.Fields(search) {
			parts = append(parts, `"`+strings.ReplaceAll(s, `"`, `""`)+`"*`)
		}
		where += " AND n.rowid IN(SELECT rowid FROM notes_fts WHERE notes_fts MATCH ?)"
		args = append(args, strings.Join(parts, " AND "))
	}
	order := "n.updated DESC"
	switch q.Get("sort") {
	case "title":
		order = "n.title COLLATE NOCASE"
	case "created":
		order = "n.created DESC"
	}
	offset, _ := strconv.Atoi(q.Get("offset"))
	if offset < 0 {
		offset = 0
	}
	cols := strings.Replace(noteCols, "n.body", "substr(n.body,1,320)", 1)
	rows, e := a.Store.DB.Query("SELECT "+cols+" FROM notes n WHERE "+where+" ORDER BY n.pinned DESC,"+order+" LIMIT 101 OFFSET ?", append(args, offset)...)
	if e != nil {
		problem(w, 500, "Search failed")
		return
	}
	notes := []Note{}
	for rows.Next() {
		n, err := scanNote(rows)
		if err != nil {
			rows.Close()
			problem(w, 500, "Could not read notes")
			return
		}
		notes = append(notes, n)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		problem(w, 500, "Could not read notes")
		return
	}
	more := len(notes) > 100
	if more {
		notes = notes[:100]
	}
	respond(w, 200, map[string]any{"notes": notes, "has_more": more})
}
func (a *App) GetNote(w http.ResponseWriter, r *http.Request) {
	n, e := a.Store.Note(who(r).User.ID, r.PathValue("id"))
	if e != nil {
		problem(w, 404, "Note not found")
		return
	}
	respond(w, 200, n)
}
func (a *App) CreateNote(w http.ResponseWriter, r *http.Request) {
	uid := who(r).User.ID
	var n Note
	if !decode(w, r, &n) {
		return
	}
	if validation(n) != nil {
		problem(w, 400, "Note exceeds limits")
		return
	}
	if !a.Store.CanNotebook(uid, n.Notebook, true) {
		problem(w, 403, "Notebook is not writable")
		return
	}
	n.ID = ID()
	n.Owner = uid
	if n.Notebook != "" {
		if e := a.Store.DB.QueryRow("SELECT owner FROM notebooks WHERE id=?", n.Notebook).Scan(&n.Owner); e != nil {
			problem(w, 404, "Notebook not found")
			return
		}
	}
	if !a.Store.WithinQuota(n.Owner, int64(len(n.Body)+len(n.Title))) {
		problem(w, 413, "Storage quota exceeded")
		return
	}
	n.Created = now()
	n.Updated = n.Created
	n.Revision = 1
	if n.Tags == nil {
		n.Tags = []string{}
	}
	_, e := a.Store.DB.Exec(`INSERT INTO notes(id,owner,notebook_id,title,body,tags,starred,pinned,archived,deleted,template,reminder,created,updated,revision)VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,1)`, n.ID, n.Owner, n.Notebook, n.Title, n.Body, jsonString(n.Tags), n.Starred, n.Pinned, n.Archived, n.Deleted, n.Template, n.Reminder, n.Created, n.Updated)
	if e != nil {
		problem(w, 500, "Could not create note")
		return
	}
	n.Permission = "owner"
	if n.Owner != uid {
		n.Permission = "editor"
	}
	respond(w, 201, n)
}
func (a *App) SaveNote(w http.ResponseWriter, r *http.Request) {
	uid := who(r).User.ID
	old, e := a.Store.Note(uid, r.PathValue("id"))
	if e != nil {
		problem(w, 404, "Note not found")
		return
	}
	if old.Permission == "viewer" {
		problem(w, 403, "This note is read-only")
		return
	}
	var n Note
	if !decode(w, r, &n) {
		return
	}
	if validation(n) != nil {
		problem(w, 400, "Note exceeds limits")
		return
	}
	if n.Revision != old.Revision {
		respond(w, 409, map[string]any{"error": "This note changed in another session. Your draft has been kept.", "current": old})
		return
	}
	if n.Notebook != old.Notebook {
		if old.Owner != uid {
			problem(w, 403, "Only the owner can move notes")
			return
		}
		var owner string
		if n.Notebook != "" {
			e = a.Store.DB.QueryRow("SELECT owner FROM notebooks WHERE id=?", n.Notebook).Scan(&owner)
			if e != nil || owner != uid {
				problem(w, 403, "Move notes only into your own notebooks")
				return
			}
		}
	}
	extra := int64(len(n.Body) + len(n.Title) - len(old.Body) - len(old.Title))
	contentChanged := n.Body != old.Body || n.Title != old.Title
	if contentChanged {
		extra += int64(len(old.Body) + len(old.Title))
	}
	if !a.Store.WithinQuota(old.Owner, extra) {
		problem(w, 413, "Storage quota exceeded")
		return
	}
	tx, e := a.Store.DB.Begin()
	if e != nil {
		problem(w, 500, "Could not save")
		return
	}
	defer tx.Rollback()
	n.Updated = now()
	n.Created = old.Created
	n.Owner = old.Owner
	n.ID = old.ID
	n.Permission = old.Permission
	if n.Tags == nil {
		n.Tags = []string{}
	}
	res, e := tx.Exec(`UPDATE notes SET notebook_id=?,title=?,body=?,tags=?,starred=?,pinned=?,archived=?,deleted=?,template=?,reminder=?,updated=?,revision=revision+1 WHERE id=? AND revision=?`, n.Notebook, n.Title, n.Body, jsonString(n.Tags), n.Starred, n.Pinned, n.Archived, n.Deleted, n.Template, n.Reminder, n.Updated, old.ID, n.Revision)
	if e != nil {
		problem(w, 500, "Could not save")
		return
	}
	changed, _ := res.RowsAffected()
	if changed != 1 {
		problem(w, 409, "Note changed concurrently; reload before saving")
		return
	}
	if contentChanged {
		if _, e = tx.Exec("INSERT INTO versions VALUES(?,?,?,?,?)", old.ID, old.Revision, old.Title, old.Body, old.Updated); e != nil {
			problem(w, 500, "Could not save history")
			return
		}
		if _, e = tx.Exec("DELETE FROM versions WHERE note_id=? AND revision NOT IN(SELECT revision FROM versions WHERE note_id=? ORDER BY revision DESC LIMIT 50)", old.ID, old.ID); e != nil {
			problem(w, 500, "Could not prune history")
			return
		}
	}
	if e = tx.Commit(); e != nil {
		problem(w, 500, "Could not commit note")
		return
	}
	n.Revision++
	respond(w, 200, n)
}
func (a *App) DeleteNote(w http.ResponseWriter, r *http.Request) {
	a.fileMu.Lock()
	defer a.fileMu.Unlock()
	n, e := a.Store.Note(who(r).User.ID, r.PathValue("id"))
	if e != nil {
		problem(w, 404, "Note not found")
		return
	}
	if n.Owner != who(r).User.ID || !n.Deleted {
		problem(w, 403, "Only the owner can permanently delete a trashed note")
		return
	}
	ids := []string{}
	rows, e := a.Store.DB.Query("SELECT id FROM attachments WHERE note_id=?", n.ID)
	if e != nil {
		problem(w, 500, "Could not delete note")
		return
	}
	for rows.Next() {
		var id string
		_ = rows.Scan(&id)
		ids = append(ids, id)
	}
	rows.Close()
	res, e := a.Store.DB.Exec("DELETE FROM notes WHERE id=? AND revision=?", n.ID, n.Revision)
	if e != nil {
		problem(w, 500, "Could not delete note")
		return
	}
	count, _ := res.RowsAffected()
	if count != 1 {
		problem(w, 409, "Note changed concurrently")
		return
	}
	for _, id := range ids {
		a.removeFile(id)
	}
	respond(w, 200, map[string]bool{"ok": true})
}
func (a *App) History(w http.ResponseWriter, r *http.Request) {
	n, e := a.Store.Note(who(r).User.ID, r.PathValue("id"))
	if e != nil {
		problem(w, 404, "Note not found")
		return
	}
	rows, e := a.Store.DB.Query("SELECT revision,title,body,created FROM versions WHERE note_id=? ORDER BY revision DESC LIMIT 50", n.ID)
	if e != nil {
		problem(w, 500, "History unavailable")
		return
	}
	defer rows.Close()
	items := []map[string]any{}
	for rows.Next() {
		var rev int
		var title, body string
		var at int64
		if rows.Scan(&rev, &title, &body, &at) != nil {
			problem(w, 500, "History unavailable")
			return
		}
		items = append(items, map[string]any{"revision": rev, "title": title, "body": body, "created": at})
	}
	respond(w, 200, items)
}
func (a *App) CreateNotebook(w http.ResponseWriter, r *http.Request) {
	var n Notebook
	if !decode(w, r, &n) {
		return
	}
	n.ID = ID()
	n.Owner = who(r).User.ID
	if !a.validNotebook(n) {
		problem(w, 400, "Invalid name or parent notebook")
		return
	}
	_, e := a.Store.DB.Exec("INSERT INTO notebooks(id,owner,name,parent_id)VALUES(?,?,?,?)", n.ID, n.Owner, n.Name, n.Parent)
	if e != nil {
		problem(w, 500, "Could not create notebook")
		return
	}
	n.Permission = "owner"
	respond(w, 201, n)
}
func (a *App) validNotebook(n Notebook) bool {
	if strings.TrimSpace(n.Name) == "" || len(n.Name) > 120 {
		return false
	}
	if n.Parent == "" {
		return true
	}
	seen := map[string]bool{n.ID: true}
	cur := n.Parent
	for cur != "" {
		if seen[cur] {
			return false
		}
		seen[cur] = true
		var owner, next string
		if a.Store.DB.QueryRow("SELECT owner,parent_id FROM notebooks WHERE id=?", cur).Scan(&owner, &next) != nil || owner != n.Owner {
			return false
		}
		cur = next
	}
	return true
}
func (a *App) SaveNotebook(w http.ResponseWriter, r *http.Request) {
	var n Notebook
	if !decode(w, r, &n) {
		return
	}
	n.ID = r.PathValue("id")
	n.Owner = who(r).User.ID
	if !a.validNotebook(n) {
		problem(w, 400, "Invalid name or parent (cycles are not allowed)")
		return
	}
	res, e := a.Store.DB.Exec("UPDATE notebooks SET name=?,parent_id=? WHERE id=? AND owner=?", n.Name, n.Parent, n.ID, n.Owner)
	if e != nil {
		problem(w, 500, "Could not save notebook")
		return
	}
	count, _ := res.RowsAffected()
	if count != 1 {
		problem(w, 404, "Notebook not found")
		return
	}
	respond(w, 200, n)
}
func (a *App) DeleteNotebook(w http.ResponseWriter, r *http.Request) {
	var count int
	id := r.PathValue("id")
	_ = a.Store.DB.QueryRow("SELECT (SELECT count(*) FROM notes WHERE notebook_id=?)+(SELECT count(*) FROM notebooks WHERE parent_id=?)", id, id).Scan(&count)
	if count > 0 {
		problem(w, 409, "Move notes and child notebooks before deleting this notebook")
		return
	}
	res, e := a.Store.DB.Exec("DELETE FROM notebooks WHERE id=? AND owner=?", id, who(r).User.ID)
	if e != nil {
		problem(w, 500, "Could not delete notebook")
		return
	}
	count64, _ := res.RowsAffected()
	if count64 != 1 {
		problem(w, 404, "Notebook not found")
		return
	}
	respond(w, 200, map[string]bool{"ok": true})
}
func (a *App) ownNotebook(uid, id string) bool {
	var owner string
	return a.Store.DB.QueryRow("SELECT owner FROM notebooks WHERE id=?", id).Scan(&owner) == nil && owner == uid
}
func (a *App) Invite(w http.ResponseWriter, r *http.Request) {
	uid := who(r).User.ID
	id := r.PathValue("id")
	if !a.ownNotebook(uid, id) {
		problem(w, 404, "Notebook not found")
		return
	}
	var input struct {
		Role string `json:"role"`
	}
	if !decode(w, r, &input) {
		return
	}
	if input.Role != "viewer" && input.Role != "editor" {
		problem(w, 400, "Role must be viewer or editor")
		return
	}
	token := secret()
	_, e := a.Store.DB.Exec("INSERT INTO invitations VALUES(?,?,?,?)", digest(token), id, input.Role, time.Now().Add(24*time.Hour).UnixMilli())
	if e != nil {
		problem(w, 500, "Could not create invitation")
		return
	}
	a.Store.Audit(uid, "create-invitation", id)
	respond(w, 201, map[string]any{"url": a.Config.PublicURL + "/#invite=" + token, "expires_hours": 24})
}
func (a *App) AcceptInvite(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Token string `json:"token"`
	}
	if !decode(w, r, &input) {
		return
	}
	tx, e := a.Store.DB.Begin()
	if e != nil {
		problem(w, 500, "Invitation failed")
		return
	}
	defer tx.Rollback()
	var book, role string
	var expiry int64
	e = tx.QueryRow("DELETE FROM invitations WHERE hash=? RETURNING notebook_id,role,expires", digest(input.Token)).Scan(&book, &role, &expiry)
	if e != nil || expiry < now() {
		problem(w, 404, "Invitation expired or already used")
		return
	}
	if _, e = tx.Exec("INSERT INTO shares VALUES(?,?,?) ON CONFLICT(notebook_id,user_id)DO UPDATE SET role=excluded.role", book, who(r).User.ID, role); e != nil {
		problem(w, 500, "Could not accept invitation")
		return
	}
	if tx.Commit() != nil {
		problem(w, 500, "Could not commit invitation")
		return
	}
	a.Store.Audit(who(r).User.ID, "accept-invitation", book)
	respond(w, 200, map[string]string{"notebook_id": book})
}
func (a *App) ListShares(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !a.ownNotebook(who(r).User.ID, id) {
		problem(w, 404, "Notebook not found")
		return
	}
	rows, e := a.Store.DB.Query("SELECT u.id,u.name,s.role FROM shares s JOIN users u ON u.id=s.user_id WHERE s.notebook_id=?", id)
	if e != nil {
		problem(w, 500, "Could not load sharing")
		return
	}
	defer rows.Close()
	out := []map[string]string{}
	for rows.Next() {
		var uid, name, role string
		_ = rows.Scan(&uid, &name, &role)
		out = append(out, map[string]string{"id": uid, "name": name, "role": role})
	}
	respond(w, 200, out)
}
func (a *App) RevokeShare(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !a.ownNotebook(who(r).User.ID, id) {
		problem(w, 404, "Notebook not found")
		return
	}
	_, e := a.Store.DB.Exec("DELETE FROM shares WHERE notebook_id=? AND user_id=?", id, r.PathValue("user"))
	if e != nil {
		problem(w, 500, "Could not revoke access")
		return
	}
	_, _ = a.Store.DB.Exec("DELETE FROM invitations WHERE notebook_id=?", id)
	a.Store.Audit(who(r).User.ID, "revoke-share", id)
	respond(w, 200, map[string]bool{"ok": true})
}
func (a *App) Admin(w http.ResponseWriter, r *http.Request) {
	if !who(r).User.Admin {
		problem(w, 403, "Administrator access required")
		return
	}
	rows, e := a.Store.DB.Query("SELECT id,name,email,quota_mb,disabled FROM users")
	if e != nil {
		problem(w, 500, "Could not load users")
		return
	}
	users := []map[string]any{}
	for rows.Next() {
		var id, name, email string
		var q int
		var disabled bool
		_ = rows.Scan(&id, &name, &email, &q, &disabled)
		users = append(users, map[string]any{"id": id, "name": name, "email": email, "quota_mb": q, "disabled": disabled})
	}
	rows.Close()
	rows, e = a.Store.DB.Query("SELECT at,actor,event,target FROM audit ORDER BY id DESC LIMIT 100")
	if e != nil {
		problem(w, 500, "Could not load audit")
		return
	}
	audit := []map[string]any{}
	for rows.Next() {
		var at int64
		var actor, event, target string
		_ = rows.Scan(&at, &actor, &event, &target)
		audit = append(audit, map[string]any{"at": at, "actor": actor, "event": event, "target": target})
	}
	rows.Close()
	respond(w, 200, map[string]any{"users": users, "audit": audit, "version": Version, "config": map[string]any{"listen": a.Config.Listen, "public_url": a.Config.PublicURL, "data_dir": a.Config.DataDir, "oidc_issuer": a.Config.Issuer, "oidc_client_id": a.Config.ClientID, "session_hours": a.Config.SessionHours, "upload_mb": a.Config.UploadMB}, "health": "ok"})
}
func (a *App) AdminUser(w http.ResponseWriter, r *http.Request) {
	if !who(r).User.Admin {
		problem(w, 403, "Administrator access required")
		return
	}
	var in struct {
		Quota    int64 `json:"quota_mb"`
		Disabled bool  `json:"disabled"`
	}
	if !decode(w, r, &in) {
		return
	}
	if in.Quota < 1 || in.Quota > 1048576 || r.PathValue("id") == who(r).User.ID && in.Disabled {
		problem(w, 400, "Invalid quota or self-disable request")
		return
	}
	res, e := a.Store.DB.Exec("UPDATE users SET quota_mb=?,disabled=? WHERE id=?", in.Quota, in.Disabled, r.PathValue("id"))
	if e != nil {
		problem(w, 500, "Could not update user")
		return
	}
	cnt, _ := res.RowsAffected()
	if cnt != 1 {
		problem(w, 404, "User not found")
		return
	}
	if in.Disabled {
		_, _ = a.Store.DB.Exec("DELETE FROM sessions WHERE user_id=?", r.PathValue("id"))
	}
	a.Store.Audit(who(r).User.ID, "update-user", r.PathValue("id"))
	respond(w, 200, map[string]bool{"ok": true})
}
func (a *App) Calendar(w http.ResponseWriter, r *http.Request) {
	n, e := a.Store.Note(who(r).User.ID, r.PathValue("id"))
	if e != nil {
		problem(w, 404, "Note not found")
		return
	}
	if n.Reminder <= 0 {
		problem(w, 400, "Set a reminder first")
		return
	}
	escape := func(s string) string {
		return strings.NewReplacer("\\", "\\\\", "\r", "", "\n", "\\n", ",", "\\,", ";", "\\;").Replace(s)
	}
	w.Header().Set("Content-Type", "text/calendar")
	w.Header().Set("Content-Disposition", `attachment; filename="note-reminder.ics"`)
	fmt.Fprintf(w, "BEGIN:VCALENDAR\r\nVERSION:2.0\r\nPRODID:-//One Workspace//notes//EN\r\nBEGIN:VEVENT\r\nUID:%s@notes\r\nDTSTAMP:%s\r\nDTSTART:%s\r\nSUMMARY:%s\r\nURL:%s/#note=%s\r\nEND:VEVENT\r\nEND:VCALENDAR\r\n", n.ID, time.Now().UTC().Format("20060102T150405Z"), time.UnixMilli(n.Reminder).UTC().Format("20060102T150405Z"), escape(n.Title), a.Config.PublicURL, n.ID)
}
