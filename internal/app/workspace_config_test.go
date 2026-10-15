package app

import (
	"path/filepath"
	"testing"
)

func TestWorkspaceCallbackDefinesPublicPrefixBeforeValidation(t *testing.T) {
	t.Setenv("NOTES_PUBLIC_URL", "")
	t.Setenv("WORKSPACE_OIDC_REDIRECT_URI", "http://localhost:18890/notes/auth/callback")
	t.Setenv("WORKSPACE_OIDC_ISSUER", "http://localhost:18891")
	c, err := LoadConfig(filepath.Join(t.TempDir(), "missing.json"))
	if err != nil {
		t.Fatal(err)
	}
	if c.PublicURL != "http://localhost:18890/notes" || c.BasePath != "/notes" || c.Mode != "managed" {
		t.Fatalf("unexpected Workspace configuration: %+v", c)
	}
	c.PublicURL = "http://localhost.evil.example/notes"
	c.RedirectURI = ""
	if err = c.Validate(); err == nil {
		t.Fatal("public HTTP origin was allowed")
	}
}
