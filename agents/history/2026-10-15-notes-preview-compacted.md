# notes implementation handoff — 2026-10-15

## Purpose and source

User requested implementing the supplied One Workspace My Notes brief in a new
public `oneworkspacestudio/notes` repository, named `notes`. The original text is
preserved at `docs/PROJECT-BRIEF.md`; it ends mid-item 9 of the final acceptance
list and contained no image mockup. The brief authorizes source publication and
versioned Windows releases. Repository creation succeeded.

Authoritative checkout: `H:/Projects/notes`. It was created on C: and moved to H:
after a native build ran out of space. The old C: task directory was removed only
after checking that all source had moved and only redundant dependency files
remained. H: is exFAT and suitable for source/disposable local-mode tests; managed
runtime data requires NTFS/persistent ACLs. An exact Git safe.directory exception
was added for this checkout. No unrelated repositories were edited.

## Implementation and decisions

Native Go executable `MyNotes.exe`, pure-Go modernc SQLite/FTS5, embedded React UI,
canonical Markdown and direct reuse of `@joplin/renderer` 3.7.1. Full Joplin web,
Electron, sync and encryption runtimes were investigated and are not embedded.
AGPL-3.0-or-later source/notices retained; no Joplin logo/icon reuse. Full boundary
and primary sources are in `docs/JOPLIN.md`.

UI trees are `web/src/Notes.aui` and `Dialog.aui`, generated with @codedia/parser.
Do not hand-edit generated TSX. Adapter components wire Tiptap and browser inputs.
Go backend supplies notes/notebooks/tags, revisions, authorized resources, shares,
OIDC, administration and backup/import/export. App ID/name is lowercase `notes`;
the binary name follows the supplied brief. Relative assets and prefix metadata
support the real launcher `/notes` contract and its supplied OIDC environment.

Security boundaries: managed mode never bypasses auth; explicit local mode is
loopback-only and requires a one-time URL. OIDC uses coreos/go-oidc and oauth2 with
state/browser binding, nonce, PKCE and signed token verification. Fixed sessions
default to eight hours, keyed to issuer/sub, and local logout is separate from
IdP logout. Admin subjects do not gain private-note access. NTFS ACL replacement,
dedicated data-directory validation, opaque filenames, server permissions,
CSRF/Origin checks, upload/import limits, quotas, CSP and final DOMPurify rendering
are implemented. Backups and drafts are plaintext; no E2EE claim.

Important frontend fixes: call Tiptap setEditable with emitUpdate=false to avoid
an update/autosave loop. Queue user actions, use current filter refs for refresh,
and merge a mutation response with any newer local edits. Regression coverage
types while a star request is delayed. Title autosizing observes width changes so
mobile navigation does not clip long titles. Extended Markdown opens in source
mode instead of silently converting through the rich editor.

Pinned Markdown dependency overrides remove initial high-severity advisories.
Markdown-It 15 requires a small compatibility rule restoring md.utils.assign for
older upstream plugins. One unpatched sprintf-js moderate advisory remains, with
two parent chains in npm audit. Go govulncheck passed. Do not claim a clean npm
audit or production security approval.

## Verification and build

17 Go API/store/auth/migration tests passed, including a native Windows ACL test
that removes an existing Everyone grant. Five Chrome workflows passed, including
Joplin preview, offline recovery/conflicts, pending-action edits, history/resources,
and desktop/mobile/dark screenshots. Native process kill/restart and CLI backup/
restore preserve content/resources and invalidate restored sessions. `/notes`
assets/APIs and safe HTML import are verified. Go vet, TypeScript, production
frontend build and CGO-free Windows build passed.

Real MyAccounts preview executable 20261014.2 was tested on a fresh disposable NTFS
installation. Two normal-realm users completed discovery/code/PKCE callbacks;
stable mapping, cross-subject isolation, unauthenticated rejection and logout pass.
No live installation, account database, or existing credentials were changed.
Fixture script is `scripts/test-myaccounts.ps1`; it creates its own DPAPI bootstrap
and stops only the process it starts. MyAccounts permits HTTP localhost fixtures;
Notes supports that exact issuer and still requires HTTPS away from loopback.

Build tool on this host: `I:/Projects/tooling/go1.27.2/go/bin/go.exe`. Go temp/cache
can use `H:/Projects/tooling/notes/`; set NOTES_ACL_TEST_ROOT to an NTFS temp root
when GOTMPDIR is exFAT. Go 1.27 testing.TempDir follows GOTMPDIR. Node 26/npm build
the frontend; no Node/Python runtime is required by the executable.

`scripts/build.ps1 -Go <path> -BrowserTests` builds/tests/packages.
`scripts/licenses.mjs` collects installed Go/npm notices into the release.
`scripts/package.ps1` emits the actual SHA256/size launcher manifest and portable
ZIP. ZIP contents and embedded executable SHA were checked. Version is
`0.1.0-preview.1`; latest local executable checkpoint is 16,515,584 bytes, with one
observed local startup of 91 ms. These are observations, not load/memory promises.

## Publication and remaining work

Publish only a labelled prerelease `v0.1.0-preview.1`, with `docs/RELEASE.md` and
`docs/ACCEPTANCE.md` limitations. The repository and expected release location are
`https://github.com/oneworkspacestudio/notes` and its releases page. Reconcile final
tag, CI run and uploaded assets with GitHub before claiming publication completed.

`docs/release-acceptance.json` remains stable_ready=false. Open gates: live
MyAccounts issuer/client registration, live Workspace add-app, clean Windows 10/11,
service/reboot and upgrade matrix, code signing, independent security review,
residual npm advisory, Drive picker/export adapter, direct Calendar APIs, JEX and
official Joplin sync/E2EE. Calendar ICS, local resources/exports, and already-open
offline drafts work. No fully cached offline PWA is claimed. User was asked for
the live issuer/client ID; no answer was supplied during implementation.
