Working first preview of **notes** for One Workspace.

Includes the three-column React interface, rich text and Joplin-powered Markdown
preview, SQLite/FTS storage, note history, attachments, nested notebooks/tags,
conflict-protected autosave and local draft recovery, notebook sharing, MyAccounts
OIDC with PKCE, administration, portable backups and launcher integration manifest.

Download `MyNotes.exe` or the portable ZIP. Check `SHA256SUMS.txt`. No Docker,
Node.js, Python or external database is needed at runtime. Extract into a dedicated
folder; `Start-local.cmd` starts an explicit loopback-only personal workspace and
prints a one-time sign-in link. Managed deployment uses MyAccounts configuration
and an NTFS data folder. `notes.workspace.json` follows the launcher add-app contract.

Validated with Go tests/vet, Chrome workflows, native restart and CLI backup/restore,
and two identities on a disposable real MyAccounts instance. Source and exact build
inputs are available in this tag under AGPL-3.0-or-later.

**Preview limitations:** live MyAccounts/Workspace registration, clean Windows 10/11
and service/reboot acceptance remain pending. Binaries are unsigned. Official
Joplin sync/E2EE, JEX and Drive picker/save integration are not implemented; Calendar
uses `.ics` export. One moderate transitive npm advisory remains. This release is
not declared production-ready or complete against the full brief.

See `docs/ACCEPTANCE.md`, `docs/JOPLIN.md` and `SECURITY.md` before deployment.
