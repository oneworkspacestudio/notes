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
