# Notes rich editor lifecycle hotfix

User reported a Workspace Notes crash: Cannot read properties of null (reading
commands), and explicitly requested fix, main push and release. Prior branding
preview3/public source1cfa5f1 preserved. No live Notes store, identity or active
Workspace process was changed. Disposable NTFS data only; no GitHub Actions.

## Cause and change

EditorView uses useEditor with note.id dependencies to keep document/undo history
separate. Tiptap destroys the old instance during the commit that changes note.id,
while that render's content-sync effect still holds the old truthy editor. Native
Editor.destroy clears commandManager, so editor.commands throws and React unmounts
the app. Guard synchronization and toolbar registration with editor.isDestroyed;
also guard update events and the lifetime of registered toolbar callbacks. Keep
per-note editor recreation so Undo cannot copy the preceding note into another.
No store/schema migration, auth change or dependency upgrade.

## Verification

New Playwright regression runs against a selectable NOTES_TEST_EXE. It reproduces
the exact commands-null error against the published preview3 MyNotes.exe. The
corrected executable passes repeated two-note switching, rich edit persistence,
Markdown-to-rich synchronization, formatting, independent Undo and reload checks,
with no page errors. Browser data fixture must select its first note explicitly.
An initial pair of test invocations collided on the same fixture port; reran
sequentially. This was test orchestration, not a second product fault.

Go API/store/auth/ACL tests and vet pass. Full browser/portable results and exact
publication receipt will be appended after completion. Sources/payloads/logs stay
on project/build drives; small disposable NTFS fixtures under a task-owned cache
must be stopped/removed after verification. Existing public preview limitations
and stable_ready=false remain; no clean-host/service/reboot claim.

## Distribution

Planned Notes v0.1.0-preview.4 and new complete Workspace bundle pinning it. Suite
work uses an isolated workspace-notes-fix worktree to preserve the primary release
artifacts and unrelated work. Retain the earlier published Music pin and withheld
hash guard; do not include the new flagged Music binary. Do not overwrite existing
tags/assets or alter the user's active installed suite as part of publication.

Full current hotfix acceptance completed: all6 Playwright workflows pass; native
portable process restart, attachments, CLI backup/restore, restored-session
invalidation, /notes prefix and HTML import pass. TypeScript/production web build,
Go tests and vet pass. Final Windows executable reports0.1.0-preview.4. No new
real provider or clean-host run; historical acceptance fields remain labelled.

## Notes publication receipt

Published v0.1.0-preview.4 from4c6e422; all9 hosted asset sizes/digests matched
local files before publishing. Public portable ZIP HEAD returned200. Source/tag,
standalone EXE, portable package, launcher manifest, licenses, provenance and
SHA256 sums are available at:
https://github.com/oneworkspacestudio/notes/releases/tag/v0.1.0-preview.4

MyNotes.exe SHA2560d7caf9d714b92e58b176123c94e553a0b2ac5b56de5d54ea2fb93f9dd0b53db
Portable ZIP SHA256e7ec8c75801d5ba0a8cc9c459bcb523783f671451519fcd44953bc3b1a81584c
The suite hotfix sourcec70d9e7 pins this exact ZIP. Full11-app/43-package bundle
was locally rebuilt/verified, including the exact published Notes4 EXE; suite
preview21 publication was being finalized. Its handoff records the final receipt.
No active user process/store changed. Task-created test processes ended and the
checked disposable NTFS test root was removed. Release artifacts/logs are under
the projects tooling notes-editor-hotfix folder. Later history-only commits do
not alter the immutable release source tag. GitHub Actions remains disabled.
