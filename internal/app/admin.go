package app

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
)

func (a *App) AdminConfig(w http.ResponseWriter, r *http.Request) {
	if !who(r).User.Admin {
		problem(w, 403, "Administrator access required")
		return
	}
	if a.Config.ConfigPath == "" {
		problem(w, 409, "This instance has no writable configuration file")
		return
	}
	var input struct {
		Listen       string `json:"listen"`
		PublicURL    string `json:"public_url"`
		DataDir      string `json:"data_dir"`
		Issuer       string `json:"oidc_issuer"`
		ClientID     string `json:"oidc_client_id"`
		SessionHours int    `json:"session_hours"`
		UploadMB     int64  `json:"upload_mb"`
	}
	if !decode(w, r, &input) {
		return
	}
	c := a.Config
	c.Listen = input.Listen
	c.PublicURL = input.PublicURL
	c.DataDir = input.DataDir
	c.Issuer = input.Issuer
	c.ClientID = input.ClientID
	c.SessionHours = input.SessionHours
	c.UploadMB = input.UploadMB
	if !filepath.IsAbs(c.DataDir) {
		problem(w, 400, "Use an absolute storage path")
		return
	}
	if c.Validate() != nil || CheckDataDirectory(c.DataDir) != nil {
		problem(w, 400, "Invalid configuration or storage directory; use a dedicated empty folder or the existing notes data folder")
		return
	}
	if c.DataDir != a.Config.DataDir {
		problem(w, 409, "Moving storage requires an offline backup and restore into the new folder first. Stop the app, restore, then update config.json.")
		return
	}
	bytes, e := json.MarshalIndent(c, "", "  ")
	if e != nil {
		problem(w, 500, "Could not serialize configuration")
		return
	}
	temp := a.Config.ConfigPath + "." + ID() + ".tmp"
	if e = os.WriteFile(temp, bytes, 0600); e != nil {
		problem(w, 500, "Could not write configuration")
		return
	}
	defer os.Remove(temp)
	if e = os.Rename(temp, a.Config.ConfigPath); e != nil {
		problem(w, 500, "Could not replace configuration")
		return
	}
	a.Store.Audit(who(r).User.ID, "save-configuration", "restart-required")
	respond(w, 200, map[string]string{"message": "Configuration saved. Restart notes to apply it. Launcher environment variables take precedence; no data has been moved."})
}
