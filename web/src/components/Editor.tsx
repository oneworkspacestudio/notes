import { createElement as h, useEffect, useState } from "react";
import { useEditor, EditorContent } from "@tiptap/react";
import StarterKit from "@tiptap/starter-kit";
import TaskList from "@tiptap/extension-task-list";
import TaskItem from "@tiptap/extension-task-item";
import Image from "@tiptap/extension-image";
import {
  Table,
  TableCell,
  TableHeader,
  TableRow,
} from "@tiptap/extension-table";
import { Markdown } from "@tiptap/markdown";
import { useNotesContext } from "../context";
import { api, url, type Attachment } from "../types";

const internalToHTTP = (s: string) =>
  s.replace(
    /(!\[[^\]]*\]\():\/([a-f0-9]{32})(\))/g,
    (_all, p, id, suffix) => p + url(`api/attachments/${id}?inline=1`) + suffix,
  );
const httpToInternal = (s: string) =>
  s.replace(
    /(!\[[^\]]*\]\()[^\s)]*\/api\/attachments\/([a-f0-9]{32})\?inline=1(\))/g,
    "$1:/$2$3",
  );
export function EditorView() {
  const ctx = useNotesContext(),
    { note, mode } = ctx.model;
  const readonly = note.permission === "viewer" || note.deleted;
  const [preview, setPreview] = useState(""),
    [renderError, setRenderError] = useState(""),
    [attachments, setAttachments] = useState<Attachment[]>([]);
  const resources = (note.body.match(/:\/[a-f0-9]{32}/g) || []).join(",");
  useEffect(() => {
    let active = true;
    api<Attachment[]>(`api/notes/${note.id}/attachments`)
      .then((items) => {
        if (active) setAttachments(items);
      })
      .catch(() => {});
    return () => {
      active = false;
    };
  }, [note.id, resources]);
  const editor = useEditor(
    {
      extensions: [
        StarterKit.configure({ link: { openOnClick: false } }),
        TaskList,
        TaskItem.configure({ nested: true }),
        Image.configure({ allowBase64: false }),
        Table.configure({ resizable: false }),
        TableRow,
        TableHeader,
        TableCell,
        Markdown,
      ],
      content: internalToHTTP(note.body),
      contentType: "markdown",
      editable: !readonly,
      editorProps: {
        attributes: { class: "rich-editor", "aria-label": "Note content" },
        handleDOMEvents: {
          drop: (_view, e) => {
            if ((e as DragEvent).dataTransfer?.files.length) {
              e.preventDefault();
              return true;
            }
            return false;
          },
          paste: (_view, e) => {
            if ((e as ClipboardEvent).clipboardData?.files.length) {
              e.preventDefault();
              return true;
            }
            return false;
          },
        },
      },
      onUpdate: ({ editor }) =>
        ctx.field("body", httpToInternal(editor.getMarkdown())),
    },
    [note.id],
  );
  useEffect(() => {
    if (!editor) return;
    if (editor.isEditable === readonly) editor.setEditable(!readonly, false);
    if (httpToInternal(editor.getMarkdown()) !== note.body)
      editor.commands.setContent(internalToHTTP(note.body), {
        contentType: "markdown",
        emitUpdate: false,
      });
  }, [note.id, note.body, readonly, editor]);
  useEffect(() => {
    if (!editor) return;
    ctx.registerEditor({
      heading: () => editor.chain().focus().toggleHeading({ level: 2 }).run(),
      bold: () => editor.chain().focus().toggleBold().run(),
      italic: () => editor.chain().focus().toggleItalic().run(),
      strike: () => editor.chain().focus().toggleStrike().run(),
      list: () => editor.chain().focus().toggleBulletList().run(),
      task: () => editor.chain().focus().toggleTaskList().run(),
      quote: () => editor.chain().focus().toggleBlockquote().run(),
      code: () => editor.chain().focus().toggleCodeBlock().run(),
      table: () =>
        editor
          .chain()
          .focus()
          .insertTable({ rows: 3, cols: 3, withHeaderRow: true })
          .run(),
      undo: () => editor.chain().focus().undo().run(),
      redo: () => editor.chain().focus().redo().run(),
    });
    return () => ctx.registerEditor({});
  }, [editor]);
  useEffect(() => {
    let current = true;
    if (mode === "split" || mode === "preview") {
      import("./renderer")
        .then(({ renderMarkdown }) =>
          renderMarkdown(
            note.body,
            attachments.map((a) => a.id),
          ),
        )
        .then((html) => {
          if (current) {
            setPreview(html);
            setRenderError("");
          }
        })
        .catch((e) => {
          if (current) setRenderError("Preview unavailable: " + e.message);
        });
    }
    return () => {
      current = false;
    };
  }, [note.body, mode, attachments]);
  const onPreviewClick = (e: any) => {
    const target = e.target.closest("a");
    const href = target?.getAttribute("href");
    if (href?.startsWith("#note=")) {
      e.preventDefault();
      ctx.action("open-note", href.slice(6));
    }
  };
  const markdown = h("textarea", {
    className: "markdown-editor",
    "data-body": true,
    "aria-label": "Markdown content",
    spellCheck: false,
    value: note.body,
    readOnly: readonly,
    onChange: (e: any) => ctx.field("body", e.target.value),
  });
  const rendered = h(
    "div",
    { className: "markdown-preview", onClick: onPreviewClick },
    renderError
      ? h("p", { role: "alert" }, renderError)
      : h("div", { dangerouslySetInnerHTML: { __html: preview } }),
  );
  if (mode === "rich")
    return h(EditorContent, { editor, className: "editor-content" });
  if (mode === "markdown") return markdown;
  if (mode === "split")
    return h("div", { className: "split-editor" }, markdown, rendered);
  return rendered;
}
