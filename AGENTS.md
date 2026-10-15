# Notes

Use repository-history before substantive work and update agents/history after material work.
The product and repository are named `notes`; the portable binary is `MyNotes.exe`.
Write UI structure in .aui and regenerate with `npm run generate` in web/.
Build web/ before Go embeds it. Use disposable data for all tests.
Never bypass OIDC in managed mode. Local mode requires explicit configuration,
loopback-only binding and a one-time login token. Do not claim Joplin sync/E2EE,
clean-machine acceptance or live integrations without recorded evidence.
Do not publish a stable release while acceptance gates remain unverified.
