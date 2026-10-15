package app

import (
	"encoding/json"
	"html"
	"io"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"path"
	"strings"
	"sync"
	"time"
)

type App struct {
	Config     Config
	Store      *Store
	Assets     fs.FS
	Logger     *log.Logger
	oidc       authProvider
	localToken string
	localMu    sync.Mutex
	fileMu     sync.Mutex
	writeMu    sync.Mutex
	rateMu     sync.Mutex
	rates      map[string]rate
}
type rate struct {
	count int
	since time.Time
}

func New(c Config, assets fs.FS, logger *log.Logger) (*App, error) {
	if e := c.Validate(); e != nil {
		return nil, e
	}
	s, e := OpenStore(c.DataDir)
	if e != nil {
		return nil, e
	}
	if logger == nil {
		logger = log.New(os.Stderr, "notes ", log.LstdFlags)
	}
	return &App{Config: c, Store: s, Assets: assets, Logger: logger, rates: map[string]rate{}}, nil
}
func (a *App) LocalURL() string {
	a.localMu.Lock()
	defer a.localMu.Unlock()
	token := secret()
	a.localToken = digest(token)
	return a.Config.PublicURL + "/auth/local?token=" + token
}
func respond(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func problem(w http.ResponseWriter, status int, message string) {
	respond(w, status, map[string]string{"error": message})
}
func decode(w http.ResponseWriter, r *http.Request, v any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 3*1024*1024)
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	if e := d.Decode(v); e != nil {
		problem(w, 400, "Invalid JSON request")
		return false
	}
	if d.Decode(new(any)) != io.EOF {
		problem(w, 400, "Expected a single JSON value")
		return false
	}
	return true
}
func (a *App) Handler() http.Handler {
	m := http.NewServeMux()
	m.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		if e := a.Store.DB.PingContext(r.Context()); e != nil {
			problem(w, 503, "Database unavailable")
			return
		}
		respond(w, 200, map[string]any{"app": "notes", "version": Version, "status": "ok"})
	})
	m.HandleFunc("GET /auth/login", a.Login)
	m.HandleFunc("GET /auth/callback", a.Callback)
	m.HandleFunc("GET /auth/local", a.LocalLogin)
	m.HandleFunc("POST /api/logout", a.Require(func(w http.ResponseWriter, r *http.Request) {
		_, e := a.Store.DB.Exec("DELETE FROM sessions WHERE hash=?", who(r).Hash)
		if e != nil {
			problem(w, 500, "Logout failed")
			return
		}
		a.cookie(w, "notes_session", "", -1)
		respond(w, 200, map[string]bool{"ok": true})
	}))
	m.HandleFunc("GET /api/me", a.Require(func(w http.ResponseWriter, r *http.Request) {
		id := who(r)
		respond(w, 200, map[string]any{"user": id.User, "csrf": id.CSRF, "version": Version, "mode": a.Config.Mode, "upload_mb": a.Config.UploadMB})
	}))
	m.HandleFunc("GET /api/state", a.Require(a.State))
	m.HandleFunc("GET /api/notes", a.Require(a.ListNotes))
	m.HandleFunc("POST /api/notes", a.Require(a.CreateNote))
	m.HandleFunc("GET /api/notes/{id}", a.Require(a.GetNote))
	m.HandleFunc("PUT /api/notes/{id}", a.Require(a.SaveNote))
	m.HandleFunc("DELETE /api/notes/{id}", a.Require(a.DeleteNote))
	m.HandleFunc("GET /api/notes/{id}/history", a.Require(a.History))
	m.HandleFunc("POST /api/notes/{id}/attachments", a.Require(a.Upload))
	m.HandleFunc("GET /api/attachments/{id}", a.Require(a.Download))
	m.HandleFunc("GET /api/notes/{id}/attachments", a.Require(a.Attachments))
	m.HandleFunc("DELETE /api/attachments/{id}", a.Require(a.DeleteAttachment))
	m.HandleFunc("POST /api/notebooks", a.Require(a.CreateNotebook))
	m.HandleFunc("PUT /api/notebooks/{id}", a.Require(a.SaveNotebook))
	m.HandleFunc("DELETE /api/notebooks/{id}", a.Require(a.DeleteNotebook))
	m.HandleFunc("GET /api/notebooks/{id}/shares", a.Require(a.ListShares))
	m.HandleFunc("POST /api/notebooks/{id}/invitations", a.Require(a.Invite))
	m.HandleFunc("POST /api/invitations/accept", a.Require(a.AcceptInvite))
	m.HandleFunc("DELETE /api/notebooks/{id}/shares/{user}", a.Require(a.RevokeShare))
	m.HandleFunc("GET /api/export", a.Require(a.Export))
	m.HandleFunc("POST /api/import", a.Require(a.Import))
	m.HandleFunc("GET /api/admin", a.Require(a.Admin))
	m.HandleFunc("PUT /api/admin/users/{id}", a.Require(a.AdminUser))
	m.HandleFunc("PUT /api/admin/config", a.Require(a.AdminConfig))
	m.HandleFunc("GET /api/notes/{id}/reminder.ics", a.Require(a.Calendar))
	m.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) { problem(w, 404, "Unknown API endpoint") })
	m.HandleFunc("/auth/", func(w http.ResponseWriter, r *http.Request) { problem(w, 404, "Unknown authentication endpoint") })
	m.HandleFunc("/", a.Static)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data: blob:; font-src 'self' data:; connect-src 'self'; frame-ancestors 'none'; object-src 'none'; base-uri 'self'; form-action 'self'")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		w.Header().Set("Cache-Control", "no-store")
		if strings.HasPrefix(a.Config.PublicURL, "https://") {
			w.Header().Set("Strict-Transport-Security", "max-age=31536000")
		}
		host, _, _ := net.SplitHostPort(r.RemoteAddr)
		a.rateMu.Lock()
		rt := a.rates[host]
		if time.Since(rt.since) > time.Minute {
			rt = rate{since: time.Now()}
		}
		rt.count++
		a.rates[host] = rt
		limited := rt.count > 2400
		if len(a.rates) > 10000 {
			for k, v := range a.rates {
				if time.Since(v.since) > time.Minute {
					delete(a.rates, k)
				}
			}
		}
		a.rateMu.Unlock()
		if limited {
			w.Header().Set("Retry-After", "60")
			problem(w, 429, "Too many requests")
			return
		}
		if a.Config.BasePath != "" && (r.URL.Path == a.Config.BasePath || strings.HasPrefix(r.URL.Path, a.Config.BasePath+"/")) {
			r.URL.Path = strings.TrimPrefix(r.URL.Path, a.Config.BasePath)
			if r.URL.Path == "" {
				r.URL.Path = "/"
			}
		}
		defer func() {
			if recover() != nil {
				a.Logger.Print("request failed unexpectedly")
				problem(w, 500, "Request failed")
			}
		}()
		m.ServeHTTP(w, r)
	})
}
func (a *App) Static(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		problem(w, 405, "Method not allowed")
		return
	}
	if a.Assets == nil {
		problem(w, 503, "Frontend assets are unavailable")
		return
	}
	p := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
	if p == "." || p == "" {
		p = "index.html"
	}
	b, e := fs.ReadFile(a.Assets, p)
	if e != nil {
		if path.Ext(p) != "" {
			http.NotFound(w, r)
			return
		}
		p = "index.html"
		b, e = fs.ReadFile(a.Assets, p)
	}
	if e != nil {
		http.NotFound(w, r)
		return
	}
	if p == "index.html" {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		b = []byte(strings.ReplaceAll(string(b), "<!--NOTES_BASE-->", `<meta name="notes-base" content="`+html.EscapeString(a.Config.BasePath+"/")+`">`))
		_, _ = w.Write(b)
		return
	}
	http.ServeContent(w, r, p, time.Time{}, strings.NewReader(string(b)))
}
