# Acceptance ledger — 2026-10-15

This preview implements a working application, not the entire original brief.
The supplied brief ends midway through acceptance item 9 and had no attached
mockup. The written three-column Gmail-style requirements guided the interface.

## Recorded validation

- Go API/store/authentication tests exercise private-note isolation including admin
  boundaries, CSRF and expiry, revision conflicts and concurrent saves, FTS across
  titles/content/tags, archive/Trash, notebook cycles, viewer/editor sharing and
  revocation, invitation replay, resource permissions, quota/disabled accounts,
  archive validation, transactional import and offline backup/restore.
- Signed OIDC fixture exercises discovery, PKCE verification, state binding,
  nonce/audience rejection, replay refusal and stable issuer/subject mapping.
- Real MyAccounts preview executable (20261014.2) was run against a fresh disposable
  NTFS installation. Two synthetic normal-user identities completed discovery and
  code/PKCE callbacks; independent sessions retained stable identity and the second
  identity could not read the first identity's private note. Logout was verified.
  No live MyAccounts installation or credentials were modified.
- Windows ACL test replaces an explicit Everyone grant on a pre-existing private
  file and verifies that the new protected ACL removes it.
- Native portable process test verifies saved content and binary attachments after
  forced termination/restart, CLI backup/restore, invalidated restored sessions,
  `/notes` routing and assets, and sanitized HTML import.
- Chrome workflows exercise editing/autosave/reload, tags/search/star/archive,
  Joplin preview checkboxes/tables, offline drafts/reconnection, concurrent-session
  conflict copies, notebooks/history/attachments/Trash, mobile and dark/light layouts.
- Frontend production build, TypeScript, native CGO-free Windows build, Go vet and
  Go vulnerability scan pass. npm has one remaining moderate transitive advisory
  plus two parent chains; see SECURITY.md. Browser assets have a large lazily loaded
  Joplin renderer chunk, explicitly recorded rather than hidden.
- Screenshot review identified and fixed grid alignment, mobile title wrapping
  and rich checklist spacing. Current screenshots are stored under `screenshots/`.

Performance is evidence, not a promise. One disposable local native startup
measurement was 213 ms; the executable at that checkpoint was 16,513,536 bytes.
Final packaging records actual file size and SHA256. These measurements are not
peak-load, memory or large-collection benchmarks.

## Remaining acceptance gates

| Original requirement | Current status |
| --- | --- |
| Clean Windows 10 and 11, no developer tools present | Not yet tested on fresh machines |
| Match supplied UI mockup | No mockup was supplied; written layout implemented |
| MyAccounts sign-in | Real isolated provider tested; live deployment registration/configuration pending |
| Multiple isolated users | API tests and two real MyAccounts subjects tested |
| Persistence, notes, notebooks, tags, attachments | Implemented and verified across native process restart |
| Editing/autosave/search/archive/restore | Implemented and browser/API-tested |
| Backup and restore | Personal ZIP/JSON and offline full installation round trips tested |
| No Docker or external database | Native embedded executable built; clean-host acceptance still pending |
| One Workspace launcher | Actual manifest contract and `/notes` prefix tested; no live suite add-app operation performed |
| Drive file picker / save exports to Drive | Adapter not implemented; local upload/download/export works |
| Calendar | `.ics` export implemented; direct API connection not implemented |
| Official Joplin sync / E2EE | Not implemented; no compatibility claim |
| JEX import/export | Not implemented; Markdown/HTML/native ZIP supported |
| Offline PWA | Draft recovery/reconnection implemented; complete offline application caching not implemented |
| Windows service/reboot and upgrade rollback | Lifecycle code included; OS install/reboot and upgrade matrix not verified |
| Admin storage changes | Safe offline migration documented; no automatic data move |
| Admin log viewer / update notifications | Audit viewer and manual release link; no automatic update polling or raw log console |
| Production security approval | Pending independent review, residual npm advisory and host acceptance |
| Windows code signing | Not performed |

Release policy: publish an explicitly labelled prerelease with these limitations.
Do not label v0.1.0 stable or mark the entire brief complete while these gates
remain open. The machine-readable status is `release-acceptance.json`.
