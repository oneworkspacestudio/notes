# CODEX PROJECT BRIEF
## One Workspace — My Notes

### 1. Project objective

Build **My Notes**, a professional, self-hosted note-taking application for the One Workspace ecosystem.

The application must use **Joplin's open-source code and functionality as its foundation**, but replace the interface with a modern, Gmail-inspired design.

The application must run as a standalone Windows executable, integrate with our existing **MyAccounts** authentication system, and operate without Docker.

**Primary objective:** Create a polished, fully functional notes application, not merely a visual prototype.

### 2. Reference design

Use the supplied My Notes UI mockup as the primary visual reference.

The interface should resemble Gmail's desktop layout rather than Google Keep.

Design requirements:

- Modern Google Material 3-inspired styling
- Light theme by default, optional dark theme
- Rounded navigation elements and subtle shadows
- Spacious typography and consistent iconography
- Blue accent colours
- Responsive desktop, tablet and mobile interfaces
- Smooth transitions and interactions
- Full-screen and split-pane editing modes

**Three-column desktop layout:**

**Left sidebar**
- My Notes branding
- Compose Note button
- All Notes
- Starred
- Reminders
- Shared
- Notebooks
- Tags
- Archive
- Trash
- Settings

**Middle column**
- Searchable notes list
- Note title and content preview
- Last modified date
- Coloured tags
- Starred indicators
- Sort and filter controls
- Multi-select and bulk operations

**Right column**
- Rich-text note editor
- Editable title
- Formatting toolbar
- Inline images and attachments
- Checklists and tables
- Note metadata
- Links to related notes
- Pin, star, share and delete actions

On mobile, convert the three-column layout into navigation screens.

### 3. Technology stack

Preferred:

- Backend: Go
- Frontend: React + TypeScript
- Styling: Tailwind CSS
- Components: shadcn/ui where suitable
- Icons: Lucide
- Database: SQLite
- Authentication: MyAccounts via OpenID Connect
- Packaging: Go embed, producing a Windows executable

Use established libraries where they reduce implementation time and maintenance.

### 4. Joplin integration

Investigate the official Joplin repositories:

https://github.com/laurent22/joplin

Reuse as much of Joplin's proven functionality as reasonably possible:

- Note creation, editing and deletion
- Notebooks and nested notebooks
- Markdown support
- Rich-text editing
- Attachments and embedded images
- Tags
- Full-text search
- Internal note links
- Import and export
- Version history
- Synchronisation
- End-to-end encryption compatibility

**Important architectural instruction:**

Do not assume Joplin's desktop application can simply be embedded in a Go server.

First investigate Joplin's current Web App, synchronisation protocol, API capabilities and licensing.

Determine which components can realistically be reused and which require an adapter or new implementation.

Do not silently replace Joplin's synchronisation or encryption behaviour with incompatible implementations.

If the original Joplin runtime cannot be embedded into a single Go executable, use a practical alternative such as adapting the web frontend and implementing a compatible backend.

Document any functionality that cannot be retained, including limitations affecting official Joplin client compatibility.

Avoid maintaining an unnecessarily large fork.

### 5. Single executable deployment

The application must ultimately be distributed as:

`MyNotes.exe`

Requirements:

- No Docker
- No external database server
- No separately installed Node.js or Python runtime
- Embedded frontend assets
- SQLite database
- Portable application data
- Automatic database migrations
- Configurable HTTP port
- Runs on Windows 10 and Windows 11
- Option to install as a Windows service
- Automatic launch through One Workspace
- Graceful shutdown and data integrity protection

Expected directory structure:

    OneWorkspace/
      Apps/
        MyNotes/
          MyNotes.exe
          config.json
          data/
            notes.db
            attachments/
            backups/

Application data must never be embedded inside the executable.

Where the single-executable requirement conflicts with Joplin compatibility, explain the technical trade-off and preserve data safety and maintainability.

### 6. MyAccounts integration

MyAccounts is our existing shared identity provider.

Implement OpenID Connect authentication with:

- Authorization Code + PKCE
- OIDC discovery
- Secure token validation
- Automatic user provisioning
- Stable user ID mapping using issuer and subject
- Per-user note isolation
- Secure session cookies
- Logout and session expiry handling
- Account profile and avatar integration

Users must not need to create a separate My Notes password.

Example desired flow:

1. User opens My Notes.
2. Application checks for an authenticated session.
3. Unauthenticated user is redirected to MyAccounts.
4. MyAccounts authenticates the user.
5. My Notes validates the login.
6. User sees only their authorised notes and notebooks.

Use the actual MyAccounts discovery endpoint and documented authentication behaviour. Do not invent undocumented endpoints.

If MyAccounts is unavailable, existing sessions should follow a clearly defined expiry policy. Never bypass authentication.

### 7. Core features

Implement the following:

**Notes**
- Create, rename, edit, duplicate and delete
- Autosave with conflict protection
- Markdown and rich-text editing
- Undo and redo
- Image attachments
- File attachments
- Checklists
- Tables
- Code blocks
- Links and quotations
- Pin and favourite
- Archive and restore
- Soft delete and Trash
- Note templates

**Organisation**
- Multiple notebooks
- Nested notebooks
- Coloured tags
- Notebook search
- Sorting by title, creation date and modification date
- Filter by tags, notebook and starred status
- Bulk actions
- Recently edited notes

**Search**
- Full-text indexed search
- Search titles, content and tags
- Search across notebooks
- Highlight matched terms
- Responsive search results

**Collaboration**
- Private notes by default
- Optional shared notebooks
- Viewer and editor permissions
- Secure sharing invitations
- Prevent access to notes through guessed identifiers

Shared notes must enforce permissions server-side.

**Import and export**
- Markdown
- HTML
- Joplin-compatible formats where feasible
- JSON backup
- Attachments
- Preserve tags and notebook hierarchy when supported

### 8. Sync and offline capability

The desktop browser experience must work well over a local network and through a reverse proxy.

Include:

- Reliable autosave
- Offline-aware UI
- Local draft recovery
- Conflict detection
- Reconnection handling
- Optional PWA support

Investigate compatibility with official Joplin desktop and mobile applications.

Do not claim official Joplin sync compatibility until it is implemented and verified with real clients.

### 9. One Workspace integration

My Notes must operate both independently and as part of One Workspace.

Integrations:

**MyAccounts**
- Authentication and account profiles

**My Drive — FileBrowser Quantum**
- Attach existing files to notes
- Save exported notes to Drive
- Secure file selection
- Respect Drive access permissions
- Never expose FileBrowser administrator credentials to the frontend

**My Calendar**
- Optionally create reminders from notes
- Link notes to calendar events

**One Workspace launcher**
- Application discovery
- Launch and stop commands
- Version reporting
- Health-check endpoint
- Installation status
- Configurable port

All integrations must be optional except MyAccounts when operating in centrally managed multi-user mode.

### 10. Administration

Create an admin settings interface supporting:

- Storage location
- Port configuration
- OIDC configuration
- User management
- Storage quotas
- Backup and restore
- Log viewer
- Health status
- Application version
- Update notifications

Administrator privileges must not automatically grant unrestricted access to private note contents.

### 11. Security

Implement production-grade safeguards:

- Authorisation checks on every notes API operation
- CSRF protection where applicable
- Strong session handling
- Content Security Policy
- Safe HTML and Markdown rendering
- Attachment validation
- Upload size limits
- Secure file path handling
- Input validation
- Rate limiting
- Protection against XSS and directory traversal
- Secure secret storage
- No API keys or client secrets in frontend assets
- Audit logging for administrative changes

Never silently disable security to make an integration work.

### 12. Performance

Optimise for low-resource Windows servers.

Targets:

- Fast startup
- Efficient SQLite queries
- Indexed full-text search
- Pagination or virtualisation for large note collections
- Lazy loading of attachments
- Minimal background CPU usage
- No unnecessary services or duplicate dependencies

Treat performance targets as benchmarks to measure, rather than making unsupported claims about memory consumption.

### 13. Development and testing

Build the application in working stages.

**Phase 1 — Architecture**
- Inspect Joplin source and licensing
- Establish reusable components
- Choose the simplest viable implementation
- Document the design

**Phase 2 — Interface**
- Implement the Gmail-inspired layout
- Match the provided reference mockup
- Create functional note navigation and editing

**Phase 3 — Storage**
- Implement SQLite persistence
- Attachments
- Tags and notebooks
- Search
- Backups

**Phase 4 — Authentication**
- Integrate MyAccounts OIDC
- Multi-user isolation
- Secure permissions

**Phase 5 — Joplin compatibility**
- Integrate reusable Joplin functionality
- Implement and test supported import/export
- Validate any sync claims against official clients

**Phase 6 — One Workspace**
- Application launcher integration
- My Drive integration
- Optional Calendar integration

**Phase 7 — Packaging**
- Build MyNotes.exe
- Bundle frontend assets
- Create portable release
- Test fresh Windows installations

**Phase 8 — Quality assurance**
- Unit tests
- API integration tests
- Authentication tests
- Multi-user isolation tests
- Data migration tests
- Browser end-to-end tests
- Backup and restore tests
- Production build validation

### 14. GitHub and releases

Create a dedicated GitHub repository for My Notes.

Include:

- Complete source code
- README with screenshots
- Installation instructions
- Configuration reference
- Architecture documentation
- Joplin compatibility documentation
- Licence and attribution notices
- Automated build workflow
- Versioned Windows releases
- Downloadable Windows executable

If repository creation or publishing is unavailable, prepare the repository locally and document the remaining steps rather than claiming publication succeeded.

### 15. Final acceptance criteria

The project is complete when:

1. A user can download and launch MyNotes.exe on a clean Windows machine.
2. The web interface closely matches the supplied Gmail-inspired mockup.
3. The user can sign in using MyAccounts.
4. Multiple users can securely maintain independent notes.
5. Notes, notebooks, tags and attachments persist across restarts.
6. Editing, autosave, search, archive and restore function correctly.
7. Application data can be backed up and restored.
8. No Docker or external database is required.
9. The application integrates with the