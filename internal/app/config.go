package app

import (
	"encoding/json"
	"errors"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const Version = "0.1.0-preview.1"

type Config struct {
	Listen        string   `json:"listen"`
	PublicURL     string   `json:"public_url"`
	DataDir       string   `json:"data_dir"`
	Mode          string   `json:"mode"`
	Issuer        string   `json:"oidc_issuer"`
	ClientID      string   `json:"oidc_client_id"`
	ClientSecret  string   `json:"-"`
	RedirectURI   string   `json:"-"`
	AdminSubjects []string `json:"admin_subjects"`
	SessionHours  int      `json:"session_hours"`
	UploadMB      int64    `json:"upload_mb"`
	QuotaMB       int64    `json:"quota_mb"`
	BasePath      string   `json:"-"`
	ConfigPath    string   `json:"-"`
}

func LoadConfig(path string) (Config, error) {
	c := Config{Listen: "127.0.0.1:8795", PublicURL: "http://127.0.0.1:8795", DataDir: "data", Mode: "managed", SessionHours: 8, UploadMB: 25, QuotaMB: 1024}
	b, e := os.ReadFile(path)
	if e == nil {
		if e = json.Unmarshal(b, &c); e != nil {
			return c, e
		}
	} else if !errors.Is(e, os.ErrNotExist) {
		return c, e
	}
	for k, p := range map[string]*string{"NOTES_DATA_DIR": &c.DataDir, "NOTES_PUBLIC_URL": &c.PublicURL, "NOTES_LISTEN": &c.Listen, "WORKSPACE_OIDC_ISSUER": &c.Issuer, "WORKSPACE_OIDC_CLIENT_ID": &c.ClientID, "WORKSPACE_OIDC_CLIENT_SECRET": &c.ClientSecret, "WORKSPACE_OIDC_REDIRECT_URI": &c.RedirectURI, "NOTES_OIDC_ISSUER": &c.Issuer, "NOTES_OIDC_CLIENT_ID": &c.ClientID, "NOTES_OIDC_CLIENT_SECRET": &c.ClientSecret} {
		if v := os.Getenv(k); v != "" {
			*p = v
		}
	}
	if !filepath.IsAbs(c.DataDir) {
		c.DataDir = filepath.Join(filepath.Dir(path), c.DataDir)
	}
	c.ConfigPath = path
	return c, c.Validate()
}
func (c *Config) Validate() error {
	u, e := url.Parse(c.PublicURL)
	if e != nil || u.Host == "" || u.RawQuery != "" || u.Fragment != "" || u.User != nil || (u.Scheme != "https" && u.Scheme != "http") {
		return errors.New("public_url must be an absolute HTTP(S) URL without credentials, query or fragment")
	}
	c.PublicURL = strings.TrimRight(c.PublicURL, "/")
	c.BasePath = strings.TrimRight(u.Path, "/")
	if c.SessionHours < 1 || c.SessionHours > 24 || c.UploadMB < 1 || c.UploadMB > 100 || c.QuotaMB < 1 {
		return errors.New("invalid session, upload or quota limits")
	}
	host, port, e := net.SplitHostPort(c.Listen)
	if e != nil {
		return e
	}
	n, e := strconv.Atoi(port)
	if e != nil || n < 1 || n > 65535 {
		return errors.New("listen port must be between 1 and 65535")
	}
	if c.Mode == "local" {
		ip := net.ParseIP(host)
		if ip == nil || !ip.IsLoopback() || u.Scheme != "http" || net.ParseIP(u.Hostname()) == nil || !net.ParseIP(u.Hostname()).IsLoopback() {
			return errors.New("local mode requires literal loopback listen and public URL addresses")
		}
	} else if c.Mode != "managed" {
		return errors.New("mode must be managed or local")
	}
	if u.Scheme == "http" && c.Mode == "managed" {
		if ip := net.ParseIP(u.Hostname()); ip == nil || !ip.IsLoopback() {
			return errors.New("managed public URLs require HTTPS (HTTP loopback is allowed for development)")
		}
	}
	if c.RedirectURI != "" && c.RedirectURI != c.PublicURL+"/auth/callback" {
		return errors.New("registered redirect URI must match public_url + /auth/callback")
	}
	if c.Issuer != "" {
		issuer, err := url.Parse(c.Issuer)
		if err != nil || issuer.Host == "" || issuer.User != nil || issuer.RawQuery != "" || issuer.Fragment != "" {
			return errors.New("invalid OIDC issuer")
		}
		ip := net.ParseIP(issuer.Hostname())
		if issuer.Scheme != "https" && (issuer.Scheme != "http" || (issuer.Hostname() != "localhost" && (ip == nil || !ip.IsLoopback()))) {
			return errors.New("OIDC issuer must use HTTPS outside loopback development")
		}
	}
	return nil
}

// Avoid applying restrictive data ACLs to an unrelated folder selected by mistake.
func CheckDataDirectory(dir string) error {
	abs, e := filepath.Abs(dir)
	if e != nil {
		return e
	}
	if filepath.Clean(abs) == filepath.Clean(filepath.VolumeName(abs)+string(filepath.Separator)) {
		return errors.New("choose a dedicated notes data folder, not a drive root")
	}
	entries, e := os.ReadDir(abs)
	if os.IsNotExist(e) {
		return nil
	}
	if e != nil {
		return e
	}
	_, dbErr := os.Stat(filepath.Join(abs, "notes.db"))
	allowed := map[string]bool{"notes.db": true, "notes.db-shm": true, "notes.db-wal": true, "notes.lock": true, "attachments": true, "backups": true}
	for _, entry := range entries {
		if !allowed[entry.Name()] || (os.IsNotExist(dbErr) && entry.Name() != "notes.lock") {
			return errors.New("data folder is not empty and does not contain an existing notes installation")
		}
	}
	return nil
}
