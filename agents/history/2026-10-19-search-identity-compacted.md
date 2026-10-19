# Search identity contract

Authorized with the Four Apps build/release task. Added authenticated GET /api/identity returning only the current session issuer/subject, allowing Search to reject account mismatch without administrator credentials. Alice/Bob/anonymous test passes. Bumped native version to 0.1.0-preview.5. Local frontend build, full Go test/vet and Windows package passed; build.ps1 uses buildvcs=false for portable checkouts.

No source ACL or note content access changed. Runtime npm audit reports three moderate findings through existing Joplin renderer -> utils -> sprintf-js <=1.1.3 (unbounded precision denial of service); this is inherited from preview.4. Do not force the suggested renderer downgrade without compatibility review. Not a zero-vulnerability or production certification claim. Packages built locally on I:, no Actions. Final source push/release/suite pin pending.
