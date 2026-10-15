package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDedicatedDataFolderAndAdminConfiguration(t *testing.T) {
	a := testApp(t)
	admin, other := client(t, a, "admin"), client(t, a, "other")
	a.Config.ConfigPath = filepath.Join(t.TempDir(), "config.json")
	a.Config.ClientSecret = "never-persist-this-secret"
	in := map[string]any{"listen": "127.0.0.1:8796", "public_url": "http://127.0.0.1:8796", "data_dir": a.Config.DataDir, "oidc_issuer": a.Config.Issuer, "oidc_client_id": "notes", "session_hours": 4, "upload_mb": 20}
	mustStatus(t, other.call("PUT", "/api/admin/config", in), 403)
	mustStatus(t, admin.call("PUT", "/api/admin/config", in), 200)
	b, e := os.ReadFile(a.Config.ConfigPath)
	if e != nil {
		t.Fatal(e)
	}
	if strings.Contains(string(b), a.Config.ClientSecret) {
		t.Fatal("secret persisted")
	}
	if a.Config.Listen != "127.0.0.1:8795" {
		t.Fatal("runtime configuration changed before restart")
	}
	in["data_dir"] = t.TempDir()
	mustStatus(t, admin.call("PUT", "/api/admin/config", in), 409)
	unrelated := t.TempDir()
	if e = os.WriteFile(filepath.Join(unrelated, "important.txt"), []byte("keep"), 0600); e != nil {
		t.Fatal(e)
	}
	if CheckDataDirectory(unrelated) == nil {
		t.Fatal("accepted unrelated directory")
	}
}
