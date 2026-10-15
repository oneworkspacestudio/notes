# Configuration and deployment

The default `config.json` is beside MyNotes.exe. Use `--config` to select another
file. A relative `data_dir` resolves against that file's directory. No note data
or credentials are embedded in the executable.

| Setting | Default / meaning |
| --- | --- |
| `listen` | `127.0.0.1:8795`; configure the HTTP listener |
| `public_url` | `http://127.0.0.1:8795`; exact browser-facing URL, including `/notes` if proxied |
| `data_dir` | `data`; dedicated storage folder |
| `mode` | `managed`; `local` is explicitly enabled with `--local` or configuration |
| `oidc_issuer` | Exact public MyAccounts issuer, without invented endpoint suffixes |
| `oidc_client_id` | Registered client ID |
| `admin_subjects` | Explicit allowed `sub` values at this configured issuer; never email-based |
| `session_hours` | 8, configurable 1–24; fixed lifetime, no silent refresh |
| `upload_mb` | 25, configurable 1–100; per-upload limit |
| `quota_mb` | 1024; initial logical-content quota per provisioned user |

Environment overrides: `NOTES_DATA_DIR`, `NOTES_PUBLIC_URL`, `NOTES_LISTEN`,
`NOTES_OIDC_ISSUER`, `NOTES_OIDC_CLIENT_ID`, `NOTES_OIDC_CLIENT_SECRET`.
One Workspace supplies `WORKSPACE_OIDC_ISSUER`, `WORKSPACE_OIDC_CLIENT_ID`,
`WORKSPACE_OIDC_CLIENT_SECRET`, `WORKSPACE_OIDC_REDIRECT_URI`; Notes accepts them
directly. The registered redirect determines the public URL before configuration validation when launched by the
workspace. Explicit NOTES OIDC variables take precedence. Keep secrets out of
config JSON, manifest fields, browser code and command-line arguments.

MyAccounts documents discovery at `ISSUER/.well-known/openid-configuration`.
Notes uses discovery-advertised authorization, token and JWKS endpoints through
coreos/go-oidc and oauth2. Configure authorization code flow and the exact callback
`PUBLIC_URL/auth/callback`, with `openid profile email`. MyAccounts is Casdoor-based;
choose the intended normal-user organization when registering the client.
The integration fixture validates this against the real MyAccounts executable.
See [MyAccounts integration documentation](https://github.com/oneworkspacestudio/myaccounts/blob/main/docs/integrations.md).

Managed mode requires HTTPS outside loopback development (`localhost` or a loopback IP). Issuer HTTP is
allowed only for loopback or `localhost` fixtures. TLS verification is never
disabled. Existing Notes sessions remain valid until their original expiry while
MyAccounts is down. New sign-ins fail. Notes logout invalidates its own server
session; it does not invent an IdP logout endpoint or promise upstream global
logout/revocation. Disabling a user in Notes revokes that user's local sessions.

Administrator configuration edits persist to `config.json` for the next restart;
they never restart the server or move data automatically. Storage moves require
an offline backup/restore and an operator change to `data_dir`. Launcher-provided
environment settings take precedence over saved configuration.

## Windows storage and service

Managed data must live on a filesystem with persistent ACLs such as NTFS. Startup
replaces the data tree's ACL with current-account, SYSTEM and Administrators access,
rejects symlinks and unrelated/non-dedicated folders, and refuses a second process
using the same database. Do not use the Joplin desktop profile as `data_dir`.
Local single-user mode permits removable storage without ACLs; physical access to
such storage provides access to its plaintext data.

Run these service commands from an elevated terminal, with an absolute config
path and a stable binary installation location:

```powershell
.\MyNotes.exe --config C:\OneWorkspace\Apps\MyNotes\config.json install
.\MyNotes.exe start
.\MyNotes.exe stop
.\MyNotes.exe uninstall
```

Service name: `OneWorkspaceNotes`. The service runs managed mode; it cannot use a
one-time interactive local login. Supply its client secret to the service process
through your protected service environment or let One Workspace handle launch
and DPAPI-protected OIDC credentials. Environment variables set only in your
terminal do not automatically propagate to an installed service. Service install,
reboot and clean-host behavior are acceptance items still awaiting verification.

Backups are plaintext ZIPs. Full backup/restore commands acquire the same instance
lock, so the service must be stopped. Restore keeps account mappings and note
history and invalidates sessions, pending logins and outstanding invitations.
For an interrupted or failed update, retain the previous executable and an offline
backup; future-schema databases are refused by older binaries.

## One Workspace launcher

`scripts/package.ps1` emits `dist/notes.workspace.json` using the executable's real
size and SHA256, following the existing
[extension contract](https://github.com/oneworkspacestudio/one-workspace/blob/main/docs/ADDING-APPS.md).
It declares ID/name `notes`, `/notes`, port 8795, `/healthz`, `accounts` dependency,
native OIDC callback `/notes/auth/callback`, and data `${DATA}/notes`.

```powershell
Workspace.exe --data C:\WorkspaceData stop
Workspace.exe --data C:\WorkspaceData add-app --manifest notes.workspace.json --package MyNotes.exe
Workspace.exe --data C:\WorkspaceData run
```

The suite provisions OIDC credentials; Notes derives its external prefix from the
callback environment and serves relative assets and API routes under it. Both
prefix-preserving proxies and the suite's `stripPrefix: true` routing are supported.
The native portable test verifies the `/notes` route and its authenticated APIs.
No live Workspace installation was modified during development.

Independent proxy deployment: set `public_url` to the full HTTPS URL, proxy to the
configured loopback listener, and forward all paths under that prefix. Keep the
HTTP port inaccessible from untrusted networks. Notes uses its configured public
origin rather than trusting arbitrary `X-Forwarded-*` headers.

## Optional connections

Calendar reminders export an interoperable `.ics` event containing the note link.
Direct Calendar API writes and event synchronization are not implemented.
The Drive picker and save-to-Drive adapter are not yet implemented. Local uploads,
downloads and export ZIPs work. No shared FileBrowser administrator token or
password is accepted or exposed; a future adapter must use each user's actual
Drive permissions. Automatic update polling is not implemented; Settings links
to the repository's releases for manual checks.
