# Joplin foundation and compatibility

## Investigation and decision

Reviewed the official Joplin repository (`dev` at
`6753a76b99c48a14f12eb20f93bfda34f3cc489e` on 2026-10-15), renderer source, raw export
implementation, current Web App architecture, synchronization specification and
E2EE specification before implementation. The exact renderer actually distributed
here is the published `@joplin/renderer` **3.7.1** artifact pinned by npm integrity;
the investigation commit is not a claim that this npm artifact was built from it.

Joplin's Web App adapts the mobile client, with a virtual filesystem, browser
SQLite WASM, cross-origin isolation and single-instance coordination. Its complete
client stack and desktop Electron runtime are not a drop-in Go server library.
Adopting the entire monorepo would create a large fork and a different browser
storage/security model from this centrally authenticated multi-user service.

This application directly reuses Joplin's maintained Markdown renderer, including
its Markdown extensions, checkbox processing, link behavior and sanitization.
The application layer is a Go/SQLite adapter with a new Gmail-style React UI and
Tiptap rich editor. Canonical note bodies use Markdown and Joplin-style `:/ID`
resource/internal links. This is a deliberately narrower foundation than reusing
Joplin's whole note model or synchronization engine.

| Capability | Status and boundary |
| --- | --- |
| Joplin Markdown renderer | Reused directly from the published AGPL package |
| Markdown notes, notebooks, tags, resources | Go service implementation; not the Joplin profile schema |
| Joplin-style internal/resource links | Adapted to authorized app routes; not proof of sync compatibility |
| Rich text | Tiptap common Markdown subset; extended syntax should stay in Markdown mode |
| Checklists, tables, code, quotations, emphasis | Implemented and browser-tested |
| Markdown import | Plain `.md` / `.markdown` files |
| Markdown export | ZIP containing Markdown files and attachment resources, plus native backup metadata |
| HTML import/export | Safe HTML-to-Markdown conversion; sanitized standalone HTML export ZIP |
| Tags/hierarchy/attachments backup | Versioned native ZIP/JSON; not JEX |
| JEX / raw Joplin item import/export | Not implemented; rejected rather than misread |
| Official Joplin desktop/mobile synchronization | Not implemented and not claimed |
| Joplin E2EE and master-key exchange | Not implemented; encrypted items are not interpreted |
| Plugins, Web Clipper and Data API | Not exposed by this service |
| Mermaid, KaTeX, ABC, Fountain, embeds and media player plugins | Disabled in this preview; keep source Markdown intact |

The official sync engine coordinates item metadata, deletions, locks and sync
target state. E2EE additionally requires master keys, key distribution and the
JED serialization/encryption behavior. Notes autosave is an independent HTTP
protocol; neither a JSON backup nor an attachment directory is a Joplin sync target.
Do not configure official clients to synchronize to this service.

## Security and version adaptations

The renderer's older transitive Markdown dependencies had known advisories. The
lockfile overrides Markdown-It to 15.0.2, Linkify-It to 6.1.0, Moment to 2.31.0,
KaTeX to 0.19.0 and sprintf-js to 1.1.3. An explicit tiny renderer rule restores
the legacy `md.utils.assign` helper needed by older Joplin Markdown plugins when
running on Markdown-It 15. Checklists/tables/links preview is exercised in Chrome.
The upstream renderer itself is not edited.

DOMPurify sanitizes the final renderer output independently of upstream behavior.
Inline event handlers, scripts, styles, frames and active HTML are removed.
Attachment image URLs resolve through authorization-checked endpoints. Remote
images are disabled, so opening a note does not silently call tracking hosts.
Joplin plugin JavaScript assets are not executed. The renderer is loaded lazily
when preview is requested.

One unpatched transitive `sprintf-js` moderate advisory remains (npm reports its
two parent dependency chains as well). No claim of a clean npm vulnerability
audit or production security certification is made; see [SECURITY.md](../SECURITY.md).

## Licensing and sources

The distributed renderer is AGPL-3.0-or-later. This application is distributed
under AGPL-3.0-or-later too; source, build inputs, notices and exact versioned
dependencies are available with release tags. Joplin logos/icons are not reused.
Joplin Server has separate directory-level licensing and is not included.

- [Joplin repository and licensing](https://github.com/laurent22/joplin)
- [Renderer usage and source](https://github.com/laurent22/joplin/tree/dev/packages/renderer)
- [Web App architecture](https://joplinapp.org/help/dev/spec/web_app/)
- [Synchronization specification](https://joplinapp.org/help/dev/spec/sync/)
- [Encryption specification](https://joplinapp.org/help/dev/spec/e2ee/)

Any future claim of official client interoperability must name the real client
versions and record round-trip, deletion, conflict and encrypted-resource tests.
