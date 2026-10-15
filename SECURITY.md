# Security and preview status

Do not include private notes, tokens, configuration secrets or backup archives
in public issues. Report a security concern privately to repository maintainers
through GitHub's private vulnerability reporting when available.

Security controls include OIDC library validation, state/nonce/PKCE binding,
hashed session tokens, fixed expiry, HttpOnly/SameSite/Secure cookies, server-side
permissions, CSRF tokens and Origin checks, CSP, independent DOMPurify sanitization,
opaque resource paths, bounded uploads/imports, image validation, quotas, rate
limits, Windows data ACLs, process locks and administrative audit records.
Administrators do not acquire API access to private notes. OS administrators and
anyone with access to plaintext storage/backups can still read that storage.

Secrets are accepted only from the server process environment. One Workspace's
documented launcher keeps its OIDC credentials in DPAPI and injects them when
starting the app. This app does not require a FileBrowser administrator secret.
Managed mode fails closed without a valid MyAccounts configuration/session.
Explicit local mode is a different, loopback-only single-user configuration with
a one-time login token. Never deploy local mode through a public reverse proxy.

Preview limitations:

- Database, attachments, ZIP backups and local drafts are not encrypted by Notes.
  Joplin E2EE is unavailable. Protect the host, browser profile and backup storage.
- One unpatched transitive `sprintf-js <=1.1.3` moderate advisory remains:
  [GHSA-hp3w-g68c-fv3c](https://github.com/advisories/GHSA-hp3w-g68c-fv3c).
  npm also counts its two parent dependency chains. Patched Markdown-It and
  Linkify-It overrides remove the initial high-severity findings. The latest
  audit artifact is `docs/npm-audit.json`; do not describe it as a clean audit.
- Go's reachable-vulnerability scan passed during development; this is not an
  independent application penetration test. Scans should be repeated for releases.
- Logout is local to Notes. Upstream logout, account removal or token revocation
  does not automatically end an already established RP session. Fixed lifetime is
  eight hours by default, configurable 1–24 hours. A Notes administrator can disable
  an account to revoke its local sessions immediately.
- Shared notebook members can read that notebook's note history and attachments.
  Removing membership blocks subsequent server requests; previously downloaded
  data cannot be recalled. Sharing is nonrecursive.
- Active SVG/HTML uploads are rejected. Imported archive resources are always served
  as downloads unless their sniffed MIME is PNG, JPEG or GIF. Files are never
  executed on the server. This does not substitute for antivirus scanning.
- Rate limiting is in-memory, per remote socket IP, with a 2,400-request/minute
  ceiling. A proxy combines its clients under that limit; no untrusted forwarded IP
  header is used. Multi-instance rate-limit coordination is not implemented.
- Windows binaries are currently unsigned; independent review, clean Windows
  10/11 verification and service/reboot testing remain open acceptance items.

Keep deployment behind a trusted HTTPS reverse proxy, bind its backend to
loopback, use an ACL-capable data volume, and install updates after reviewing
release notes and making an offline backup. This preview is not declared
production-ready.
