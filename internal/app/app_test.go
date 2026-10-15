package app

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"testing/fstest"
)

type testClient struct {
	app    *App
	user   User
	cookie *http.Cookie
	csrf   string
}

func testApp(t *testing.T) *App {
	t.Helper()
	c := Config{Listen: "127.0.0.1:8795", PublicURL: "http://127.0.0.1:8795", Mode: "managed", DataDir: t.TempDir(), UploadMB: 25, QuotaMB: 10, SessionHours: 8, Issuer: "https://accounts.invalid", AdminSubjects: []string{"admin"}}
	a, e := New(c, fstest.MapFS{"index.html": {Data: []byte("<!--NOTES_BASE-->ok")}}, log.New(io.Discard, "", 0))
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { a.Store.Close() })
	return a
}
func client(t *testing.T, a *App, sub string) testClient {
	t.Helper()
	u, e := a.Store.Provision(a.Config.Issuer, sub, sub, sub+"@example.test", "", 10)
	if e != nil {
		t.Fatal(e)
	}
	w := httptest.NewRecorder()
	if e = a.newSession(w, u); e != nil {
		t.Fatal(e)
	}
	cookie := w.Result().Cookies()[0]
	var csrf string
	if e = a.Store.DB.QueryRow("SELECT csrf FROM sessions WHERE hash=?", digest(cookie.Value)).Scan(&csrf); e != nil {
		t.Fatal(e)
	}
	return testClient{a, u, cookie, csrf}
}
func (c testClient) call(method, path string, v any) *httptest.ResponseRecorder {
	var body io.Reader
	if v != nil {
		data, _ := json.Marshal(v)
		body = bytes.NewReader(data)
	}
	r := httptest.NewRequest(method, c.app.Config.PublicURL+path, body)
	r.AddCookie(c.cookie)
	r.Header.Set("Origin", c.app.Config.PublicURL)
	r.Header.Set("X-CSRF-Token", c.csrf)
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	c.app.Handler().ServeHTTP(w, r)
	return w
}
func mustStatus(t *testing.T, r *httptest.ResponseRecorder, status int) {
	t.Helper()
	if r.Code != status {
		t.Fatalf("status %d, want %d: %s", r.Code, status, r.Body.String())
	}
}
func (c testClient) note(t *testing.T, book, title, body string) Note {
	t.Helper()
	r := c.call("POST", "/api/notes", Note{Notebook: book, Title: title, Body: body, Tags: []string{"research"}})
	mustStatus(t, r, 201)
	var n Note
	if e := json.Unmarshal(r.Body.Bytes(), &n); e != nil {
		t.Fatal(e)
	}
	return n
}
func (c testClient) upload(t *testing.T, id, name string, data []byte) *httptest.ResponseRecorder {
	t.Helper()
	var b bytes.Buffer
	mw := multipart.NewWriter(&b)
	part, e := mw.CreateFormFile("file", name)
	if e != nil {
		t.Fatal(e)
	}
	_, _ = part.Write(data)
	_ = mw.Close()
	r := httptest.NewRequest("POST", c.app.Config.PublicURL+"/api/notes/"+id+"/attachments", &b)
	r.AddCookie(c.cookie)
	r.Header.Set("Origin", c.app.Config.PublicURL)
	r.Header.Set("X-CSRF-Token", c.csrf)
	r.Header.Set("Content-Type", mw.FormDataContentType())
	w := httptest.NewRecorder()
	c.app.Handler().ServeHTTP(w, r)
	return w
}
func TestIsolationAndAdminDoesNotReadPrivateNotes(t *testing.T) {
	a := testApp(t)
	owner, admin := client(t, a, "owner"), client(t, a, "admin")
	n := owner.note(t, "", "private", "private unique starlight")
	mustStatus(t, admin.call("GET", "/api/notes/"+n.ID, nil), 404)
	mustStatus(t, admin.call("PUT", "/api/notes/"+n.ID, n), 404)
	r := admin.call("GET", "/api/notes?q=starlight", nil)
	mustStatus(t, r, 200)
	if strings.Contains(r.Body.String(), n.ID) {
		t.Fatal("search leaked note")
	}
	mustStatus(t, admin.call("GET", "/api/admin", nil), 200)
	mustStatus(t, owner.call("GET", "/api/admin", nil), 403)
	r = admin.call("GET", "/api/export", nil)
	mustStatus(t, r, 200)
	z, e := zip.NewReader(bytes.NewReader(r.Body.Bytes()), int64(r.Body.Len()))
	if e != nil {
		t.Fatal(e)
	}
	for _, f := range z.File {
		src, _ := f.Open()
		b, _ := io.ReadAll(src)
		src.Close()
		if strings.Contains(string(b), n.ID) {
			t.Fatal("admin export leaked private note")
		}
	}
}
func TestCSRFAndSessionExpiry(t *testing.T) {
	a := testApp(t)
	c := client(t, a, "alice")
	r := httptest.NewRequest("POST", a.Config.PublicURL+"/api/notes", strings.NewReader(`{"title":"x"}`))
	r.AddCookie(c.cookie)
	w := httptest.NewRecorder()
	a.Handler().ServeHTTP(w, r)
	mustStatus(t, w, 403)
	r.Header.Set("X-CSRF-Token", c.csrf)
	r.Header.Set("Origin", "https://attacker.invalid")
	w = httptest.NewRecorder()
	a.Handler().ServeHTTP(w, r)
	mustStatus(t, w, 403)
	_, e := a.Store.DB.Exec("UPDATE sessions SET expires=0")
	if e != nil {
		t.Fatal(e)
	}
	mustStatus(t, c.call("GET", "/api/me", nil), 401)
}
func TestAutosaveConflictsAndHistory(t *testing.T) {
	a := testApp(t)
	c := client(t, a, "alice")
	n := c.note(t, "", "first", "old body")
	one := n
	one.Body = "new body"
	mustStatus(t, c.call("PUT", "/api/notes/"+n.ID, one), 200)
	n.Body = "stale body"
	mustStatus(t, c.call("PUT", "/api/notes/"+n.ID, n), 409)
	current, e := a.Store.Note(c.user.ID, n.ID)
	if e != nil || current.Body != "new body" || current.Revision != 2 {
		t.Fatal(current, e)
	}
	r := c.call("GET", "/api/notes/"+n.ID+"/history", nil)
	mustStatus(t, r, 200)
	if !strings.Contains(r.Body.String(), "old body") {
		t.Fatal("missing history")
	}
}
func TestConcurrentSavesOnlyOneWins(t *testing.T) {
	a := testApp(t)
	c := client(t, a, "alice")
	n := c.note(t, "", "concurrent", "original")
	results := make(chan int, 2)
	var wg sync.WaitGroup
	for _, body := range []string{"A", "B"} {
		wg.Add(1)
		go func(body string) {
			defer wg.Done()
			copy := n
			copy.Body = body
			results <- c.call("PUT", "/api/notes/"+n.ID, copy).Code
		}(body)
	}
	wg.Wait()
	close(results)
	success, conflicts := 0, 0
	for code := range results {
		if code == 200 {
			success++
		} else if code == 409 {
			conflicts++
		} else {
			t.Fatal(code)
		}
	}
	if success != 1 || conflicts != 1 {
		t.Fatal(success, conflicts)
	}
}
func TestSearchTagsArchiveTrashAndPurge(t *testing.T) {
	a := testApp(t)
	c := client(t, a, "alice")
	n := c.note(t, "", "Market research", "Project sapphire observations")
	for _, q := range []string{"sapphire", "research", "Market"} {
		r := c.call("GET", "/api/notes?q="+q, nil)
		mustStatus(t, r, 200)
		if !strings.Contains(r.Body.String(), n.ID) {
			t.Fatal("search missed " + q)
		}
	}
	n.Archived = true
	r := c.call("PUT", "/api/notes/"+n.ID, n)
	mustStatus(t, r, 200)
	_ = json.Unmarshal(r.Body.Bytes(), &n)
	if strings.Contains(c.call("GET", "/api/notes", nil).Body.String(), n.ID) {
		t.Fatal("archive in main list")
	}
	if !strings.Contains(c.call("GET", "/api/notes?view=archive", nil).Body.String(), n.ID) {
		t.Fatal("archive missing")
	}
	mustStatus(t, c.call("DELETE", "/api/notes/"+n.ID, nil), 403)
	n.Deleted = true
	r = c.call("PUT", "/api/notes/"+n.ID, n)
	mustStatus(t, r, 200)
	if !strings.Contains(c.call("GET", "/api/notes?view=trash", nil).Body.String(), n.ID) {
		t.Fatal("trash missing")
	}
	mustStatus(t, c.call("DELETE", "/api/notes/"+n.ID, nil), 200)
	mustStatus(t, c.call("GET", "/api/notes/"+n.ID, nil), 404)
}
func TestNotebookHierarchyAndSharing(t *testing.T) {
	a := testApp(t)
	owner, viewer := client(t, a, "owner"), client(t, a, "viewer")
	r := owner.call("POST", "/api/notebooks", Notebook{Name: "Projects"})
	mustStatus(t, r, 201)
	var book Notebook
	_ = json.Unmarshal(r.Body.Bytes(), &book)
	r = owner.call("POST", "/api/notebooks", Notebook{Name: "Nested", Parent: book.ID})
	mustStatus(t, r, 201)
	var child Notebook
	_ = json.Unmarshal(r.Body.Bytes(), &child)
	book.Parent = child.ID
	mustStatus(t, owner.call("PUT", "/api/notebooks/"+book.ID, book), 400)
	n := owner.note(t, book.ID, "Shared", "Shared body")
	hidden := owner.note(t, child.ID, "Child private", "Do not inherit sharing")
	mustStatus(t, viewer.call("GET", "/api/notes/"+n.ID, nil), 404)
	for _, role := range []string{"viewer", "editor"} {
		r = owner.call("POST", "/api/notebooks/"+book.ID+"/invitations", map[string]string{"role": role})
		mustStatus(t, r, 201)
		var invite map[string]any
		_ = json.Unmarshal(r.Body.Bytes(), &invite)
		token := strings.Split(invite["url"].(string), "#invite=")[1]
		mustStatus(t, viewer.call("POST", "/api/invitations/accept", map[string]string{"token": token}), 200)
		mustStatus(t, viewer.call("POST", "/api/invitations/accept", map[string]string{"token": token}), 404)
		mustStatus(t, viewer.call("GET", "/api/notes/"+n.ID, nil), 200)
		mustStatus(t, viewer.call("GET", "/api/notes/"+hidden.ID, nil), 404)
		n.Body = "edited"
		expected := 403
		if role == "editor" {
			expected = 200
		}
		mustStatus(t, viewer.call("PUT", "/api/notes/"+n.ID, n), expected)
	}
	mustStatus(t, owner.call("DELETE", "/api/notebooks/"+book.ID+"/shares/"+viewer.user.ID, nil), 200)
	mustStatus(t, viewer.call("GET", "/api/notes/"+n.ID, nil), 404)
	mustStatus(t, owner.call("DELETE", "/api/notebooks/"+book.ID, nil), 409)
}
func TestAttachmentsAndBackupRoundTrip(t *testing.T) {
	a := testApp(t)
	c, other := client(t, a, "owner"), client(t, a, "other")
	n := c.note(t, "", "Documents", "original")
	r := c.upload(t, n.ID, "../safe.txt", []byte("attachment content"))
	mustStatus(t, r, 201)
	var at Attachment
	_ = json.Unmarshal(r.Body.Bytes(), &at)
	if at.Name != "safe.txt" {
		t.Fatal(at.Name)
	}
	mustStatus(t, other.call("GET", "/api/attachments/"+at.ID, nil), 404)
	r = c.call("GET", "/api/attachments/"+at.ID, nil)
	mustStatus(t, r, 200)
	if r.Header().Get("Content-Disposition") != "attachment; filename=safe.txt" {
		t.Fatal(r.Header())
	}
	mustStatus(t, c.upload(t, n.ID, "x.svg", []byte("<svg onload=alert(1)></svg>")), 400)
	n.Body = "![file](:/" + at.ID + ")"
	mustStatus(t, c.call("PUT", "/api/notes/"+n.ID, n), 200)
	r = c.call("GET", "/api/export", nil)
	mustStatus(t, r, 200)
	b, files, e := readBackupZip(r.Body.Bytes())
	if e != nil {
		t.Fatal(e)
	}
	count, e := a.restoreUser(other.user.ID, b, files)
	if e != nil || count != 1 {
		t.Fatal(count, e)
	}
	var id, body string
	e = a.Store.DB.QueryRow("SELECT id,body FROM notes WHERE owner=?", other.user.ID).Scan(&id, &body)
	if e != nil || id == n.ID || strings.Contains(body, at.ID) {
		t.Fatal(id, body, e)
	}
	var newFile string
	_ = a.Store.DB.QueryRow("SELECT id FROM attachments WHERE note_id=?", id).Scan(&newFile)
	r = other.call("GET", "/api/attachments/"+newFile, nil)
	mustStatus(t, r, 200)
	if r.Body.String() != "attachment content" {
		t.Fatal(r.Body.String())
	}
}
func TestImportValidationAndZipBombLimit(t *testing.T) {
	a := testApp(t)
	c := client(t, a, "owner")
	bad := Backup{Format: "oneworkspace-notes", Version: 1, Notebooks: []Notebook{{ID: ID(), Name: "A", Parent: ID()}}, Notes: []Note{{ID: ID(), Title: "will not import"}}}
	if _, e := a.restoreUser(c.user.ID, bad, nil); e == nil {
		t.Fatal("accepted invalid hierarchy")
	}
	var count int
	_ = a.Store.DB.QueryRow("SELECT COUNT(*) FROM notes").Scan(&count)
	if count != 0 {
		t.Fatal("partial import")
	}
	var b bytes.Buffer
	z := zip.NewWriter(&b)
	for i := 0; i < 2; i++ {
		f, _ := z.Create("backup.json")
		_, _ = f.Write([]byte(`{}`))
	}
	z.Close()
	if _, _, e := readBackupZip(b.Bytes()); e == nil {
		t.Fatal("accepted duplicate entries")
	}
}
func TestQuotaAndDisabledUser(t *testing.T) {
	a := testApp(t)
	c, admin := client(t, a, "alice"), client(t, a, "admin")
	mustStatus(t, admin.call("PUT", "/api/admin/users/"+c.user.ID, map[string]any{"quota_mb": 1, "disabled": false}), 200)
	n := Note{Title: "Large", Body: strings.Repeat("x", 1100000)}
	mustStatus(t, c.call("POST", "/api/notes", n), 413)
	mustStatus(t, admin.call("PUT", "/api/admin/users/"+c.user.ID, map[string]any{"quota_mb": 1, "disabled": true}), 200)
	mustStatus(t, c.call("GET", "/api/me", nil), 401)
}
func TestManagedModeFailsClosedAndLocalTokenSingleUse(t *testing.T) {
	a := testApp(t)
	a.Config.Issuer = ""
	r := httptest.NewRequest("GET", a.Config.PublicURL+"/auth/login", nil)
	w := httptest.NewRecorder()
	a.Handler().ServeHTTP(w, r)
	mustStatus(t, w, 503)
	mustStatus(t, testClient{app: a, cookie: &http.Cookie{Name: "notes_session", Value: "fake"}}.call("GET", "/api/notes", nil), 401)
	a.Config.Mode = "local"
	link := a.LocalURL()
	for i, status := range []int{303, 401} {
		r = httptest.NewRequest("GET", link, nil)
		r.RemoteAddr = "127.0.0.1:12345"
		w = httptest.NewRecorder()
		a.Handler().ServeHTTP(w, r)
		if w.Code != status {
			t.Fatal(i, w.Code, w.Body.String())
		}
	}
}
func TestPrefixAssetsAndSecurityHeaders(t *testing.T) {
	a := testApp(t)
	a.Config.PublicURL += "/notes"
	if e := a.Config.Validate(); e != nil {
		t.Fatal(e)
	}
	r := httptest.NewRequest("GET", a.Config.PublicURL+"/", nil)
	w := httptest.NewRecorder()
	a.Handler().ServeHTTP(w, r)
	mustStatus(t, w, 200)
	if !strings.Contains(w.Body.String(), `content="/notes/"`) || !strings.Contains(w.Header().Get("Content-Security-Policy"), "frame-ancestors 'none'") {
		t.Fatal(w.Body.String(), w.Header())
	}
}
func TestOfflineBackupRestoreAndMigrationReopen(t *testing.T) {
	dir := t.TempDir()
	s, e := OpenStore(dir)
	if e != nil {
		t.Fatal(e)
	}
	u, e := s.Provision("issuer", "sub", "Alice", "", "", 10)
	if e != nil {
		t.Fatal(e)
	}
	_, e = s.DB.Exec("INSERT INTO notes(id,owner,title,body,created,updated)VALUES(?,?,?,?,?,?)", ID(), u.ID, "Persistent", "backup body", now(), now())
	if e != nil {
		t.Fatal(e)
	}
	s.Close()
	zipPath := filepath.Join(t.TempDir(), "backup.zip")
	if e = BackupAll(dir, zipPath); e != nil {
		t.Fatal(e)
	}
	dest := t.TempDir()
	if e = RestoreAll(zipPath, dest); e != nil {
		t.Fatal(e)
	}
	restored, e := OpenStore(dest)
	if e != nil {
		t.Fatal(e)
	}
	defer restored.Close()
	var body string
	if e = restored.DB.QueryRow("SELECT body FROM notes").Scan(&body); e != nil || body != "backup body" {
		t.Fatal(body, e)
	}
	if e = RestoreAll(zipPath, dest); e == nil {
		t.Fatal("restored over active database")
	}
	info, e := os.Stat(filepath.Join(dest, "notes.db"))
	if e != nil || info.Size() == 0 {
		t.Fatal(info, e)
	}
}
func TestLocalModeRejectsNonLoopback(t *testing.T) {
	c := Config{Listen: "0.0.0.0:8795", PublicURL: "http://127.0.0.1:8795", Mode: "local", SessionHours: 8, UploadMB: 25, QuotaMB: 10}
	if c.Validate() == nil {
		t.Fatal("accepted LAN local mode")
	}
	c.Listen = "127.0.0.1:8795"
	c.PublicURL = "https://public.example.test"
	if c.Validate() == nil {
		t.Fatal("accepted public local mode")
	}
}
