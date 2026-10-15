import { useEffect, useRef, useState } from "react";
import {
  api,
  APIError,
  blankNote,
  setCSRF,
  url,
  type Note,
  type Notebook,
  type User,
} from "./types";

const navItems = [
  ["all", "All notes", "NotebookPen"],
  ["starred", "Starred", "Star"],
  ["reminders", "Reminders", "Bell"],
  ["shared", "Shared", "Users"],
  ["archive", "Archive", "Archive"],
  ["trash", "Trash", "Trash2"],
];
const sortOptions = [
  { value: "updated", label: "Last edited" },
  { value: "created", label: "Date created" },
  { value: "title", label: "Title A–Z" },
];
const color = (s: string) =>
  ["blue", "green", "purple", "amber"][
    Array.from(s).reduce((a, c) => a + c.charCodeAt(0), 0) % 4
  ];
const date = (n: number) =>
  new Date(n).toLocaleDateString(undefined, { month: "short", day: "numeric" });

export function useNotes() {
  const [user, setUser] = useState<User | null>(null),
    [version, setVersion] = useState(""),
    [notes, setNotes] = useState<Note[]>([]),
    [notebooks, setNotebooks] = useState<Notebook[]>([]),
    [tags, setTags] = useState<string[]>([]),
    [count, setCount] = useState(0),
    [hasMore, setHasMore] = useState(false);
  const [note, setNote] = useState<Note | null>(null),
    [view, setView] = useState("all"),
    [search, setSearch] = useState(""),
    [bookSearch, setBookSearch] = useState(""),
    [sort, setSort] = useState("updated"),
    [book, setBook] = useState(""),
    [tag, setTag] = useState(""),
    [selected, setSelected] = useState<Set<string>>(new Set());
  const [mode, setMode] = useState("rich"),
    [fullscreen, setFullscreen] = useState(false),
    [mobileEditor, setMobileEditor] = useState(false),
    [menu, setMenu] = useState(false),
    [theme, setTheme] = useState(
      localStorage.getItem("notes-theme") || "light",
    );
  const [dialog, setDialog] = useState<any>(null),
    [error, setError] = useState(""),
    [toast, setToast] = useState(""),
    [dirty, setDirty] = useState(false),
    [saveStatus, setSaveStatus] = useState("All changes saved"),
    [conflict, setConflict] = useState(false),
    [recovery, setRecovery] = useState<Note | null>(null),
    [tick, setTick] = useState(0);
  const noteRef = useRef<Note | null>(null),
    dirtyRef = useRef(false),
    saveRef = useRef<Promise<boolean> | null>(null),
    generation = useRef(0),
    commands = useRef<Record<string, () => void>>({}),
    request = useRef(0),
    userRef = useRef<User | null>(null),
    conflictRef = useRef(false),
    toastTimer = useRef<number>(0);
  const filters = useRef({ view, search, sort, book, tag });
  filters.current = { view, search, sort, book, tag };
  const actionQueue = useRef<Promise<void>>(Promise.resolve());
  const runLatest = useRef<(name: string, payload?: any) => Promise<void>>(
    async () => {},
  );
  const notify = (s: string) => {
    setToast(s);
    window.clearTimeout(toastTimer.current);
    toastTimer.current = window.setTimeout(() => setToast(""), 4500);
  };
  function assign(n: Note | null) {
    noteRef.current = n;
    setNote(n);
  }
  const key = (id: string) => `notes-draft:${userRef.current?.id}:${id}`;
  function persistDraft(n: Note) {
    try {
      localStorage.setItem(key(n.id), JSON.stringify(n));
    } catch {
      setError(
        "This device has no draft storage space. Keep this tab open until your note saves to the server.",
      );
    }
  }
  function acceptMutation(saved: Note, patch: Partial<Note>) {
    if (noteRef.current?.id !== saved.id) return;
    if (dirtyRef.current) {
      const merged = {
        ...noteRef.current,
        ...patch,
        revision: saved.revision,
        updated: saved.updated,
      };
      assign(merged);
      persistDraft(merged);
      setTick((t) => t + 1);
    } else assign(saved);
  }
  function update(patch: Partial<Note>) {
    const n = noteRef.current;
    if (!n || n.permission === "viewer" || n.deleted) return;
    if (
      Object.entries(patch).every(([key, value]) =>
        Object.is((n as any)[key], value),
      )
    )
      return;
    const updated = { ...n, ...patch };
    assign(updated);
    generation.current++;
    dirtyRef.current = true;
    setDirty(true);
    persistDraft(updated);
    setSaveStatus(
      navigator.onLine ? "Unsaved changes" : "Offline · draft saved here",
    );
    setTick((n) => n + 1);
  }
  async function save(): Promise<boolean> {
    if (saveRef.current) return saveRef.current;
    if (!dirtyRef.current || !noteRef.current) return !conflictRef.current;
    if (conflictRef.current || !navigator.onLine) return false;
    const snapshot = { ...noteRef.current },
      g = generation.current;
    setSaveStatus("Saving…");
    const operation = (async () => {
      try {
        const saved = await api<Note>(
          `api/notes/${snapshot.id}`,
          "PUT",
          snapshot,
        );
        if (noteRef.current?.id === saved.id) {
          if (g === generation.current) {
            assign(saved);
            dirtyRef.current = false;
            setDirty(false);
            localStorage.removeItem(key(saved.id));
            setSaveStatus("All changes saved");
            setError("");
          } else {
            assign({
              ...noteRef.current,
              revision: saved.revision,
              updated: saved.updated,
            });
            persistDraft(noteRef.current);
            setTick((n) => n + 1);
          }
        }
        setNotes((list) => list.map((n) => (n.id === saved.id ? saved : n)));
        return true;
      } catch (e) {
        if (e instanceof APIError && e.status === 409) {
          conflictRef.current = true;
          setConflict(true);
          setSaveStatus("Conflict · draft kept");
        } else {
          setSaveStatus("Not saved · draft kept");
        }
        setError(e instanceof Error ? e.message : "Save failed");
        return false;
      } finally {
        saveRef.current = null;
      }
    })();
    saveRef.current = operation;
    return operation;
  }
  async function flush() {
    if (saveRef.current && !(await saveRef.current)) return false;
    if (dirtyRef.current && !(await save())) return false;
    return !dirtyRef.current && !conflictRef.current;
  }
  async function refresh(more = false) {
    const current = ++request.current;
    const { view, search, sort, book, tag } = filters.current;
    const query = new URLSearchParams({
      view,
      q: search,
      sort,
      notebook: book,
      tag,
      offset: String(more ? notes.length : 0),
    });
    const [list, state] = await Promise.all([
      api(`api/notes?${query}`),
      api("api/state"),
    ]);
    if (current !== request.current) return;
    setNotes((prev) => (more ? [...prev, ...list.notes] : list.notes));
    setHasMore(list.has_more);
    setNotebooks(state.notebooks);
    setTags(state.tags);
    setCount(state.count);
  }
  async function openNote(id: string, skipFlush = false) {
    if (!skipFlush && !(await flush())) return;
    const n = await api<Note>(`api/notes/${id}`);
    if (dirtyRef.current && !(await flush())) return;
    if (/\[\^|^::|^```(?:mermaid|katex)|<\w|==[^=]/m.test(n.body))
      setMode("markdown");
    assign(n);
    dirtyRef.current = false;
    setDirty(false);
    conflictRef.current = false;
    setConflict(false);
    setError("");
    setSaveStatus("All changes saved");
    setMobileEditor(true);
    setDialog(null);
    setRecovery(null);
    const stored = localStorage.getItem(key(id));
    if (stored) {
      try {
        const draft = JSON.parse(stored);
        if (
          draft.id === id &&
          [
            "title",
            "body",
            "notebook_id",
            "tags",
            "starred",
            "pinned",
            "archived",
            "deleted",
            "template",
            "reminder",
          ].some(
            (k) => JSON.stringify(draft[k]) !== JSON.stringify((n as any)[k]),
          )
        )
          setRecovery(draft);
      } catch {
        localStorage.removeItem(key(id));
      }
    }
    history.replaceState(null, "", `#note=${id}`);
  }
  async function create(overrides: Partial<Note> = {}) {
    if (!(await flush())) return;
    const n = await api<Note>("api/notes", "POST", {
      ...blankNote(),
      notebook_id: book,
      ...overrides,
    });
    if (dirtyRef.current && !(await flush())) return;
    assign(n);
    dirtyRef.current = false;
    setDirty(false);
    setMobileEditor(true);
    setDialog(null);
    setRecovery(null);
    conflictRef.current = false;
    setConflict(false);
    await refresh();
    history.replaceState(null, "", `#note=${n.id}`);
  }
  async function mutate(patch: Partial<Note>) {
    if (!noteRef.current || noteRef.current.permission === "viewer") return;
    if (!(await flush())) return;
    const n = await api<Note>(`api/notes/${noteRef.current.id}`, "PUT", {
      ...noteRef.current,
      ...patch,
    });
    acceptMutation(n, patch);
    await refresh();
  }
  const pickFile = (fn: (file: File) => Promise<void>) => {
    const input = document.createElement("input");
    input.type = "file";
    input.addEventListener(
      "change",
      () => {
        const f = input.files?.[0];
        if (f)
          fn(f).catch((e) => {
            setError(e.message);
            notify(e.message);
          });
      },
      { once: true },
    );
    input.click();
  };
  function showDialog(kind: string, extra: any = {}) {
    setDialog({
      title:
        kind === "settings"
          ? "Your workspace"
          : kind === "notebook"
            ? "Notebook"
            : kind === "details"
              ? "Note details"
              : kind === "share"
                ? "Share notebook"
                : kind === "admin"
                  ? "Administration"
                  : "Insert a link",
      isSettings: kind === "settings",
      isNotebook: kind === "notebook",
      isDetails: kind === "details",
      isShare: kind === "share",
      isAdmin: kind === "admin",
      isLink: kind === "link",
      ...extra,
    });
  }
  async function shareMembers(id: string) {
    return id ? await api(`api/notebooks/${id}/shares`) : [];
  }
  async function run(name: string, payload?: any) {
    setError("");
    if (name.startsWith("fmt-")) {
      const kind = name.slice(4);
      if (kind === "link") {
        showDialog("link", { href: "" });
        return;
      }
      if (mode === "rich") {
        commands.current[kind]?.();
      } else {
        const insert: Record<string, string> = {
          heading: "\n## Heading\n",
          bold: "**bold text**",
          italic: "*italic text*",
          strike: "~~text~~",
          list: "\n- Item\n",
          task: "\n- [ ] Task\n",
          quote: "\n> Quote\n",
          code: "\n```\ncode\n```\n",
          table: "\n| Heading | Heading |\n| --- | --- |\n| Cell | Cell |\n",
        };
        if (kind === "undo" || kind === "redo") {
          const textarea = document.querySelector<HTMLTextAreaElement>(
            "textarea[data-body]",
          );
          textarea?.focus();
          document.execCommand(kind);
          return;
        }
        if (insert[kind]) {
          const area = document.querySelector<HTMLTextAreaElement>(
            "textarea[data-body]",
          );
          const n = noteRef.current;
          if (n) {
            const at = area?.selectionStart ?? n.body.length;
            update({
              body:
                n.body.slice(0, at) +
                insert[kind] +
                n.body.slice(area?.selectionEnd ?? at),
            });
          }
        }
      }
      return;
    }
    if (name.startsWith("mode-")) {
      const next = name.slice(5);
      if (
        next === "rich" &&
        mode !== "rich" &&
        /\[\^|^::|^```(?:mermaid|katex)|<\w|==[^=]/m.test(
          noteRef.current?.body || "",
        )
      ) {
        notify(
          "This note uses extended Markdown. Rich text could change it; keep Markdown or split view.",
        );
        return;
      }
      setMode(next);
      return;
    }
    switch (name) {
      case "compose":
        await create();
        break;
      case "open-note":
        await openNote(payload);
        break;
      case "refresh":
        if (await flush()) await refresh();
        break;
      case "load-more":
        await refresh(true);
        break;
      case "menu":
        setMenu(!menu);
        break;
      case "back":
        if (await flush()) setMobileEditor(false);
        break;
      case "fullscreen":
        setFullscreen(!fullscreen);
        break;
      case "theme":
        setTheme(theme === "light" ? "dark" : "light");
        break;
      case "navigate":
        if (!(await flush())) return;
        setView(payload);
        setBook("");
        setTag("");
        setSelected(new Set());
        setMobileEditor(false);
        setMenu(false);
        setDialog(null);
        break;
      case "show-templates":
        if (!(await flush())) return;
        setView("templates");
        setBook("");
        setTag("");
        setDialog(null);
        break;
      case "notebook":
        if (!(await flush())) return;
        setBook(payload);
        setView("all");
        setTag("");
        setMenu(false);
        setSelected(new Set());
        break;
      case "tag":
        if (!(await flush())) return;
        setTag(tag === payload ? "" : payload);
        setBook("");
        setView("all");
        setMenu(false);
        break;
      case "settings":
        showDialog("settings");
        break;
      case "close-dialog":
        setDialog(null);
        break;
      case "new-notebook":
        showDialog("notebook", { name: "", parent: book, editing: false });
        break;
      case "edit-notebook": {
        const b = notebooks.find((n) => n.id === payload);
        if (!b || b.permission !== "owner") {
          notify("Only the owner can edit this notebook.");
          return;
        }
        showDialog("notebook", {
          id: b.id,
          name: b.name,
          parent: b.parent_id,
          editing: true,
        });
        break;
      }
      case "save-notebook":
        await api(
          `api/notebooks${dialog.editing ? "/" + dialog.id : ""}`,
          dialog.editing ? "PUT" : "POST",
          { name: dialog.name, parent_id: dialog.parent || "" },
        );
        setDialog(null);
        await refresh();
        break;
      case "delete-notebook":
        await api(`api/notebooks/${dialog.id}`, "DELETE");
        if (book === dialog.id) setBook("");
        setDialog(null);
        await refresh();
        break;
      case "star":
        await mutate({ starred: !noteRef.current?.starred });
        break;
      case "pin":
        await mutate({ pinned: !noteRef.current?.pinned });
        break;
      case "archive":
        await mutate({ archived: !noteRef.current?.archived });
        notify("Note updated");
        break;
      case "trash":
        await mutate({ deleted: !noteRef.current?.deleted });
        break;
      case "quick-star": {
        if (!(await flush())) return;
        const n = await api<Note>(`api/notes/${payload}`);
        const saved = await api<Note>(`api/notes/${payload}`, "PUT", {
          ...n,
          starred: !n.starred,
        });
        acceptMutation(saved, { starred: saved.starred });
        await refresh();
        break;
      }
      case "bulk-star":
      case "bulk-archive":
      case "bulk-trash": {
        if (!(await flush())) return;
        let done = 0;
        try {
          for (const id of selected) {
            const n = await api<Note>(`api/notes/${id}`);
            const patch =
              name === "bulk-star"
                ? { starred: true }
                : name === "bulk-archive"
                  ? { archived: true }
                  : { deleted: true };
            const saved = await api<Note>(`api/notes/${id}`, "PUT", {
              ...n,
              ...patch,
            });
            acceptMutation(saved, patch);
            done++;
          }
        } finally {
          notify(`${done} notes updated`);
          setSelected(new Set());
          await refresh();
        }
        break;
      }
      case "purge":
        if (
          !noteRef.current ||
          !confirm(
            "Permanently delete this note and its attachments? This cannot be undone.",
          )
        )
          return;
        await api(`api/notes/${noteRef.current.id}`, "DELETE");
        localStorage.removeItem(key(noteRef.current.id));
        assign(null);
        dirtyRef.current = false;
        setDirty(false);
        await refresh();
        break;
      case "details":
        if (noteRef.current)
          showDialog("details", {
            versions: [],
            attachments: (
              await api(`api/notes/${noteRef.current.id}/attachments`)
            ).map((a: any) => ({
              ...a,
              url: url(`api/attachments/${a.id}`),
              sizeLabel: Math.ceil(a.size / 1024) + " KB",
            })),
          });
        break;
      case "delete-attachment":
        if (
          !confirm(
            "Remove this attachment? Existing links to it will no longer work.",
          )
        )
          return;
        await api(`api/attachments/${payload}`, "DELETE");
        setDialog({
          ...dialog,
          attachments: dialog.attachments.filter((a: any) => a.id !== payload),
        });
        notify("Attachment removed");
        break;
      case "duplicate": {
        if (!(await flush()) || !noteRef.current) return;
        const n = noteRef.current;
        await create({
          ...n,
          id: "",
          title: (n.title || "Untitled note") + " (copy)",
          notebook_id: n.owner === user?.id ? n.notebook_id : "",
          deleted: false,
          archived: false,
          template: false,
        });
        notify(
          "Copy created. Attachment references retain their original permissions.",
        );
        break;
      }
      case "attach": {
        if (!(await flush()) || !noteRef.current) return;
        const id = noteRef.current.id;
        pickFile(async (f) => {
          const body = new FormData();
          body.append("file", f);
          const at = await api(`api/notes/${id}/attachments`, "POST", body);
          if (noteRef.current?.id === id)
            update({
              body:
                noteRef.current.body +
                `\n${at.mime.startsWith("image/") ? "!" : ""}[${at.name.replace(/[\[\]]/g, "")}](:/${at.id})\n`,
            });
          notify("File attached");
        });
        break;
      }
      case "import":
        pickFile(async (f) => {
          if (!(await flush())) return;
          const body = new FormData();
          body.append("file", f);
          const result = await api("api/import", "POST", body);
          await refresh();
          notify(`${result.imported} notes imported`);
          setDialog(null);
        });
        break;
      case "history":
        if (!(await flush()) || !noteRef.current) return;
        setDialog({
          ...dialog,
          versions: (await api(`api/notes/${noteRef.current.id}/history`)).map(
            (v: any) => ({
              ...v,
              label: `Revision ${v.revision} · ${new Date(v.created).toLocaleString()}`,
            }),
          ),
        });
        break;
      case "restore-version": {
        const v = dialog.versions.find((v: any) => v.revision === payload);
        if (v) {
          update({ title: v.title, body: v.body });
          if (await flush()) {
            setDialog(null);
            notify("Previous version restored");
          }
        }
        break;
      }
      case "copy-link":
        if (noteRef.current) {
          await navigator.clipboard.writeText(
            `[${noteRef.current.title}](:/${noteRef.current.id})`,
          );
          notify("Internal Markdown link copied");
        }
        break;
      case "share": {
        const b = noteRef.current?.notebook_id || book;
        if (
          !b ||
          !notebooks.some((n) => n.id === b && n.permission === "owner")
        ) {
          notify("Move the note into a notebook you own before sharing.");
          return;
        }
        showDialog("share", {
          book: b,
          role: "viewer",
          members: await shareMembers(b),
        });
        break;
      }
      case "create-invite":
        if (!dialog.book) throw new Error("Choose a notebook");
        setDialog({
          ...dialog,
          invite: (
            await api(`api/notebooks/${dialog.book}/invitations`, "POST", {
              role: dialog.role,
            })
          ).url,
          members: await shareMembers(dialog.book),
        });
        break;
      case "copy-invite":
        await navigator.clipboard.writeText(dialog.invite);
        notify("Invitation copied");
        break;
      case "revoke-share":
        await api(`api/notebooks/${dialog.book}/shares/${payload}`, "DELETE");
        setDialog({ ...dialog, members: await shareMembers(dialog.book) });
        notify("Access revoked; outstanding invitations cancelled");
        break;
      case "insert-link": {
        const href = String(dialog.href || "");
        if (
          !/^(https?:\/\/|mailto:|:\/[a-f0-9]{32}$|#note=[a-f0-9]{32}$)/i.test(
            href,
          )
        )
          throw new Error("Use HTTPS, mailto, or an internal note link");
        if (mode === "rich") commands.current["link:" + href]?.();
        const n = noteRef.current;
        if (n)
          update({
            body:
              n.body +
              `\n[Link](${href.replace(/[()\s]/g, encodeURIComponent)})\n`,
          });
        setDialog(null);
        break;
      }
      case "recover":
        if (recovery && noteRef.current) {
          const server = noteRef.current;
          if (recovery.revision !== server.revision) {
            conflictRef.current = true;
            setConflict(true);
          }
          assign({
            ...recovery,
            revision: server.revision,
            permission: server.permission,
          });
          dirtyRef.current = true;
          setDirty(true);
          generation.current++;
          setRecovery(null);
          setTick((n) => n + 1);
        }
        break;
      case "discard-draft":
        if (noteRef.current) localStorage.removeItem(key(noteRef.current.id));
        setRecovery(null);
        break;
      case "conflict-copy": {
        const draft = noteRef.current;
        if (!draft) return;
        const oldID = draft.id;
        dirtyRef.current = false;
        setDirty(false);
        conflictRef.current = false;
        setConflict(false);
        try {
          await create({
            ...draft,
            id: "",
            title: (draft.title || "Untitled note") + " (recovered)",
            notebook_id: notebooks.some(
              (b) =>
                b.id === draft.notebook_id && b.owner === userRef.current?.id,
            )
              ? draft.notebook_id
              : "",
            deleted: false,
            archived: false,
          });
          localStorage.removeItem(key(oldID));
        } catch (e) {
          assign(draft);
          dirtyRef.current = true;
          setDirty(true);
          conflictRef.current = true;
          setConflict(true);
          throw e;
        }
        break;
      }
      case "reload-note":
        if (
          noteRef.current &&
          confirm("Discard the local draft and load the server version?")
        ) {
          const id = noteRef.current.id;
          dirtyRef.current = false;
          setDirty(false);
          conflictRef.current = false;
          localStorage.removeItem(key(id));
          await openNote(id, true);
        }
        break;
      case "admin": {
        const data = await api("api/admin");
        showDialog("admin", {
          ...data,
          system: `notes ${data.version} · ${data.health} · ${data.config.listen} · ${data.config.data_dir} · ${data.config.oidc_issuer || "Local mode"}`,
          audit: data.audit.map((v: any) => ({
            ...v,
            label: `${new Date(v.at).toLocaleString()} · ${v.event} · ${v.target}`,
          })),
        });
        break;
      }
      case "save-user": {
        const u = dialog.users.find((u: any) => u.id === payload);
        await api(`api/admin/users/${payload}`, "PUT", {
          quota_mb: Number(u.quota_mb),
          disabled: u.disabled,
        });
        notify("User settings saved");
        break;
      }
      case "save-config": {
        const c = dialog.config;
        const result = await api("api/admin/config", "PUT", {
          ...c,
          session_hours: Number(c.session_hours),
          upload_mb: Number(c.upload_mb),
        });
        notify(result.message);
        break;
      }
      case "logout":
        if (!(await flush())) return;
        await api("api/logout", "POST");
        for (const k of Object.keys(localStorage)) {
          if (k.startsWith(`notes-draft:${userRef.current?.id}:`))
            localStorage.removeItem(k);
        }
        location.assign(url(""));
        break;
    }
  }
  runLatest.current = run;
  const action = (name: string, payload?: any) => {
    actionQueue.current = actionQueue.current
      .then(() => new Promise<void>((resolve) => setTimeout(resolve, 0)))
      .then(() => runLatest.current(name, payload))
      .catch((e) => {
        setError(e.message);
        notify(e.message);
      });
  };
  function field(name: string, value: any, payload?: any) {
    if (name === "title") update({ title: value });
    else if (name === "body") update({ body: value });
    else if (name === "search") setSearch(value);
    else if (name === "sort") setSort(value);
    else if (name === "book-search") setBookSearch(value);
    else if (name === "select-all")
      setSelected(value ? new Set(notes.map((n) => n.id)) : new Set());
    else if (name === "select-note")
      setSelected((prev) => {
        const next = new Set(prev);
        value ? next.add(payload) : next.delete(payload);
        return next;
      });
    else if (name === "note-notebook") update({ notebook_id: value });
    else if (name === "note-tags")
      update({
        tags: Array.from(
          new Set(
            String(value)
              .split(",")
              .map((t) => t.trim())
              .filter(Boolean),
          ),
        ),
      });
    else if (name === "note-template") update({ template: value });
    else if (name === "note-reminder")
      update({ reminder: value ? new Date(value).getTime() : 0 });
    else if (name.startsWith("dialog-")) {
      const key = name.slice(7);
      setDialog((d: any) => ({ ...d, [key]: value }));
      if (key === "book")
        shareMembers(value)
          .then((members) => setDialog((d: any) => ({ ...d, members })))
          .catch((e) => notify(e.message));
    } else if (name === "user-quota" || name === "user-disabled")
      setDialog((d: any) => ({
        ...d,
        users: d.users.map((u: any) =>
          u.id === payload
            ? { ...u, [name === "user-quota" ? "quota_mb" : "disabled"]: value }
            : u,
        ),
      }));
    else if (name.startsWith("config-"))
      setDialog((d: any) => ({
        ...d,
        config: { ...d.config, [name.slice(7)]: value },
      }));
  }
  useEffect(() => {
    document.documentElement.dataset.theme = theme;
    localStorage.setItem("notes-theme", theme);
  }, [theme]);
  useEffect(() => {
    const expired = () => {
      setUser(null);
      userRef.current = null;
      assign(null);
      setDialog(null);
      dirtyRef.current = false;
      setDirty(false);
      setError(
        "Session expired. Sign in again; local drafts can be recovered after authentication.",
      );
    };
    window.addEventListener("notes-session-expired", expired);
    api("api/me")
      .then(async (info) => {
        setUser(info.user);
        userRef.current = info.user;
        setVersion(info.version);
        setCSRF(info.csrf);
        const hash = location.hash;
        if (hash.startsWith("#invite=")) {
          sessionStorage.setItem("notes-invite", hash.slice(8));
          history.replaceState(null, "", url(""));
        }
        const invite = sessionStorage.getItem("notes-invite");
        if (invite) {
          await api("api/invitations/accept", "POST", { token: invite });
          sessionStorage.removeItem("notes-invite");
          notify("Notebook invitation accepted");
        }
        if (hash.startsWith("#note=")) await openNote(hash.slice(6));
      })
      .catch((e) => {
        if (location.hash.startsWith("#invite="))
          sessionStorage.setItem("notes-invite", location.hash.slice(8));
        if (!(e instanceof APIError && e.status === 401)) setError(e.message);
      });
    return () => window.removeEventListener("notes-session-expired", expired);
  }, []);
  useEffect(() => {
    if (!user) return;
    const timer = window.setTimeout(
      () => refresh().catch((e) => setError(e.message)),
      search ? 220 : 0,
    );
    return () => clearTimeout(timer);
  }, [user, view, book, tag, search, sort]);
  useEffect(() => {
    if (!dirty || conflict || recovery) return;
    const timer = window.setTimeout(() => {
      void save();
    }, 800);
    return () => clearTimeout(timer);
  }, [dirty, tick, conflict, recovery]);
  useEffect(() => {
    const reconnect = () => {
      setSaveStatus(
        dirtyRef.current ? "Reconnected · saving draft" : "All changes saved",
      );
      save()
        .then(() => refresh())
        .catch((e) => setError(e.message));
    };
    const offline = () => setSaveStatus("Offline · drafts stay on this device");
    const before = (e: BeforeUnloadEvent) => {
      if (dirtyRef.current) {
        e.preventDefault();
        e.returnValue = "";
      }
    };
    const keys = (e: KeyboardEvent) => {
      if ((e.ctrlKey || e.metaKey) && e.key === "s") {
        e.preventDefault();
        void save();
      }
    };
    const hash = () => {
      if (location.hash.startsWith("#note="))
        void openNote(location.hash.slice(6)).catch((e) => setError(e.message));
    };
    window.addEventListener("online", reconnect);
    window.addEventListener("offline", offline);
    window.addEventListener("beforeunload", before);
    window.addEventListener("keydown", keys);
    window.addEventListener("hashchange", hash);
    return () => {
      window.removeEventListener("online", reconnect);
      window.removeEventListener("offline", offline);
      window.removeEventListener("beforeunload", before);
      window.removeEventListener("keydown", keys);
      window.removeEventListener("hashchange", hash);
    };
  }, [user, view, book, tag, search, sort]);
  const ownOptions = [
    { value: "", label: "No notebook" },
    ...notebooks
      .filter((b) => b.owner === user?.id)
      .map((b) => ({ value: b.id, label: b.name })),
  ];
  const displayNotes = notes.map((n) => ({
    ...n,
    displayTitle: n.title || "Untitled note",
    date: date(n.updated),
    preview:
      n.body
        .replace(/[#*`>\[\]]/g, "")
        .replace(/\n/g, " ")
        .slice(0, 150) || "A fresh page. Start writing.",
    badges: [
      ...n.tags.slice(0, 2).map((t) => ({ label: t, color: color(t) })),
      ...(n.pinned ? [{ label: "Pinned", color: "muted" }] : []),
    ],
    cardRegion: n.id === note?.id ? "note-card active" : "note-card",
    selected: selected.has(n.id),
    selectLabel: `Select ${n.title || "Untitled note"}`,
    starVariant: n.starred ? "icon starred" : "icon note-star",
  }));
  return {
    model: {
      user,
      versionLabel: `notes ${version} · One Workspace`,
      notes: displayNotes,
      note,
      notebooks,
      books: notebooks
        .filter((b) => b.name.toLowerCase().includes(bookSearch.toLowerCase()))
        .map((b) => ({
          ...b,
          displayName: (b.parent_id ? "↳ " : "") + b.name,
          variant: book === b.id ? "nav active" : "nav",
        })),
      tags: tags.map((name) => ({
        name,
        color: color(name),
        variant: tag === name ? "nav active" : "nav",
      })),
      navigation: navItems.map(([key, label, icon]) => ({
        key,
        label,
        icon,
        variant: view === key && !book && !tag ? "nav active" : "nav",
        count: key === "all" ? String(count) : "",
      })),
      viewTitle: book
        ? notebooks.find((b) => b.id === book)?.name || "Notebook"
        : tag
          ? "#" + tag
          : navItems.find((n) => n[0] === view)?.[1] || "Templates",
      search,
      bookSearch,
      sort,
      sortOptions,
      empty: notes.length === 0,
      hasMore,
      noteCount: String(notes.length) + (hasMore ? "+" : ""),
      allSelected: notes.length > 0 && selected.size === notes.length,
      hasSelected: selected.size > 0,
      selectedLabel: `${selected.size} selected`,
      shellRegion: `workspace${fullscreen ? " fullscreen" : ""}${mobileEditor ? " mobile-editor" : ""}${menu ? " menu-open" : ""}`,
      saveStatus,
      saveTone: conflict ? "error" : dirty ? "amber" : "muted",
      error,
      toast,
      conflict,
      recovery,
      initials:
        user?.name
          .split(" ")
          .slice(0, 2)
          .map((n) => n[0])
          .join("") || "N",
      notebookLabel:
        notebooks.find((b) => b.id === note?.notebook_id)?.name ||
        "Personal notes",
      pinVariant: note?.pinned ? "icon active" : "icon",
      starVariant: note?.starred ? "icon starred" : "icon",
      richVariant: mode === "rich" ? "tab active" : "tab",
      markdownVariant: mode === "markdown" ? "tab active" : "tab",
      splitVariant: mode === "split" ? "tab active" : "tab",
      previewVariant: mode === "preview" ? "tab active" : "tab",
      mode,
      permissionLabel:
        note?.permission === "viewer"
          ? "View only"
          : note?.permission === "editor"
            ? "Shared · can edit"
            : "Only you",
      modifiedLabel: note
        ? "Edited " +
          new Date(note.updated).toLocaleString(undefined, {
            month: "short",
            day: "numeric",
            hour: "2-digit",
            minute: "2-digit",
          })
        : "",
      wordCount: `${note?.body.trim().split(/\s+/).filter(Boolean).length || 0} words`,
      dialog,
      ownNotebookOptions: ownOptions,
      detailNotebookOptions: [
        ...ownOptions,
        ...notebooks
          .filter((b) => b.id === note?.notebook_id && b.owner !== user?.id)
          .map((b) => ({ value: b.id, label: b.name + " (shared)" })),
      ],
      parentOptions: ownOptions.filter((o) => o.value !== dialog?.id),
      roleOptions: [
        { value: "viewer", label: "Can view" },
        { value: "editor", label: "Can edit" },
      ],
      tagText: note?.tags.join(", ") || "",
      reminderValue: note?.reminder
        ? new Date(
            note.reminder - new Date(note.reminder).getTimezoneOffset() * 60000,
          )
            .toISOString()
            .slice(0, 16)
        : "",
      exportJSON: url("api/export?format=json"),
      exportMD: url("api/export?format=markdown"),
      exportHTML: url("api/export?format=html"),
      hasReminder: !!note?.reminder,
      calendarURL: url(`api/notes/${note?.id}/reminder.ics`),
      healthURL: url("healthz"),
      loginURL: url("auth/login"),
    },
    action,
    field,
    registerEditor: (value: Record<string, () => void>) => {
      commands.current = value;
    },
  };
}
