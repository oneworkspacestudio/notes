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

Published public source at `fcd186ebf0fb1f9b7b114c3dd09cb99d865d6b0a` and a labelled
prerelease `v0.1.0-preview.1`. GitHub verified the tag targets that commit, the
release is public (draft=false, prerelease=true), and all seven assets are present.
Release: https://github.com/oneworkspacestudio/notes/releases/tag/v0.1.0-preview.1
Hosted Windows CI passed: https://github.com/oneworkspacestudio/notes/actions/runs/38758523188

Release files are the downloaded, successful hosted Windows CI artifact, not the
earlier local build. The exact CI executable was additionally verified against
the real disposable two-user MyAccounts fixture. Published MyNotes.exe is
16,515,072 bytes, SHA256
`5d81ba3e1caef127d8be67da7bc2b77c93f587d7b305c45579c278a1b1548b4a`.
Portable ZIP is 7,221,128 bytes, SHA256
`4c83374bba18aebc1611b79c8b4236633731e52934d25632975039017631f0f7`.
GitHub's asset digests match these values. CI artifact retained locally at
`H:/Projects/tooling/notes/release-ci-fcd186e`; development build remains in dist/.

The HTTPS OAuth credential lacks workflow write scope, so its initial push was
rejected. Existing authorized SSH access succeeded; origin now uses
`git@github.com:oneworkspacestudio/notes.git`. Use SSH for workflow-bearing pushes.
No extra authorization or credential changes were required.

`docs/release-acceptance.json` remains stable_ready=false. Open gates: live
MyAccounts issuer/client registration, live Workspace add-app, clean Windows 10/11,
service/reboot and upgrade matrix, code signing, independent security review,
residual npm advisory, Drive picker/export adapter, direct Calendar APIs, JEX and
official Joplin sync/E2EE. Calendar ICS, local resources/exports, and already-open
offline drafts work. No fully cached offline PWA is claimed. User was asked for
the live issuer/client ID; no answer was supplied during implementation.

## Workspace installer integration follow-up

During the user's all-organization-apps integration, preview1 became available.
Source review found that LoadConfig validated the default standalone URL before
main could apply WORKSPACE_OIDC_REDIRECT_URI, and managed public localhost URLs
were rejected although Workspace's local origin uses localhost. Preview2 derives
the callback public prefix before validation and permits managed loopback
localhost (public HTTP remains rejected). The manifest supplies NOTES_PUBLIC_URL
and records the exact organization source release. Managed OIDC is retained.

Added regression test covers callback-only configuration, /notes prefix and
rejection of localhost.evil.example HTTP. All Go tests and vet pass, existing
frontend regenerated/built, Windows CGO-free executable and portable package
built locally. No GitHub Actions used for preview2; source commits use [skip ci].
Earlier preview1 CI evidence remains historical, not evidence for this build.
Browser sign-in was not rerun because this task's browser policy service was
unavailable. Existing frontend and OIDC verification code were unchanged.

One Workspace19 packages this corrected release, provisions its own OIDC client,
passes exact public/data/listen settings and creates notes-settings.json outside
the dedicated Notes data directory. The Workspace owner is an explicit Notes
admin; unrelated users receive no additional access. Record actual suite startup
verification after deployment in Workspace's organization-apps handoff.
Preview2 publication completed: https://github.com/oneworkspacestudio/notes/releases/tag/v0.1.0-preview.2
Tag source0fecd40; all seven hosted asset digests/sizes checked before publishing.
Latest Actions run remains the earlier preview1 fcd186e run, confirming no new
hosted build was used. Workspace19 downloaded the published2 portable ZIP and
checked its aggregate digest plus inner EXE against the organization release.

The actual authorized Workspace installation now starts Notes2 automatically:
health200, private unauthenticated API401, managed mode, explicit Workspace-owner
admin configuration and all app processes healthy. OIDC client provisioned and
protected by the suite; config/data locations verified. This establishes live
installation/startup, not a new browser sign-in or cross-user UI acceptance run.
No manual issuer/client/secret entry was required. Existing mail/files preserved.