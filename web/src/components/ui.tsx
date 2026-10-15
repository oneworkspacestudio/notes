import {
  createElement as h,
  useLayoutEffect,
  useRef,
  type ReactNode,
} from "react";
import {
  Menu,
  NotebookPen,
  Plus,
  Star,
  Bell,
  Users,
  Archive,
  Trash2,
  FolderClosed,
  Ellipsis,
  Hash,
  Layers3,
  Settings2,
  RefreshCw,
  FileSearch,
  ShieldCheck,
  ArrowLeft,
  FileText,
  Pin,
  Maximize2,
  Heading2,
  Bold,
  Italic,
  Strikethrough,
  List,
  ListTodo,
  Quote,
  Code2,
  Table2,
  Link as LinkIcon,
  Paperclip,
  Undo2,
  Redo2,
  Clock3,
  CheckCheck,
  X,
  SunMoon,
  Search,
  Circle,
} from "lucide-react";
import { useNotesContext } from "../context";
import { Dialog } from "../Dialog.generated";
import { EditorView } from "./Editor";
type Props = Record<string, any> & { children?: ReactNode };
const icons = {
  Menu,
  NotebookPen,
  Plus,
  Star,
  Bell,
  Users,
  Archive,
  Trash2,
  FolderClosed,
  Ellipsis,
  Hash,
  Layers3,
  Settings2,
  RefreshCw,
  FileSearch,
  ShieldCheck,
  ArrowLeft,
  FileText,
  Pin,
  Maximize2,
  Heading2,
  Bold,
  Italic,
  Strikethrough,
  List,
  ListTodo,
  Quote,
  Code2,
  Table2,
  Link: LinkIcon,
  Paperclip,
  Undo2,
  Redo2,
  Clock3,
  CheckCheck,
  X,
  SunMoon,
  Search,
  Circle,
};
const cls = (p: Props, base = "") =>
  [base, p.region, p.variant, p.tone && "tone-" + p.tone]
    .filter(Boolean)
    .join(" ");
export function Page(p: Props) {
  return h("div", { className: "page" }, p.children);
}
export function Stack(p: Props) {
  return h("div", { className: cls(p, "stack") }, p.children);
}
export function Row(p: Props) {
  return h("div", { className: cls(p, "row") }, p.children);
}
export function Section(p: Props) {
  return h("section", { className: cls(p) }, p.children);
}
export function Spacer() {
  return h("span", { className: "spacer" });
}
export function Heading(p: Props) {
  return h("h" + (p.level || 2), { className: cls(p) }, p.children);
}
export function Text(p: Props) {
  const { model } = useNotesContext();
  let children = p.children;
  if (p.tone === "preview" && model.search && typeof children === "string") {
    const escaped = model.search.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
    const terms = new RegExp("(" + escaped + ")", "ig");
    children = children
      .split(terms)
      .map((s: string, i: number) => (i % 2 ? h("mark", { key: i }, s) : s));
  }
  return h("span", { className: cls(p, "text") }, children);
}
export function Badge(p: Props) {
  if (p.children === undefined || p.children === "") return null;
  return h("span", { className: cls(p, "badge") }, p.children);
}
export function Icon(p: Props) {
  const Comp = (icons as any)[p.name] || icons.Circle;
  return h(Comp, {
    size: p.size || 19,
    strokeWidth: 1.7,
    className: cls(p, "icon-glyph"),
    "aria-hidden": true,
  });
}
export function Divider() {
  return h("span", { className: "divider" });
}
export function Button(p: Props) {
  const ctx = useNotesContext();
  return h(
    "button",
    {
      type: "button",
      className: cls(p, "button"),
      title: p.title,
      "aria-label": p.title,
      disabled:
        (ctx.model.note?.permission === "viewer" &&
          (p.intent?.startsWith("fmt-") ||
            [
              "pin",
              "star",
              "archive",
              "trash",
              "attach",
              "purge",
              "restore-version",
              "delete-attachment",
            ].includes(p.intent))) ||
        (ctx.model.note?.deleted &&
          (p.intent?.startsWith("fmt-") ||
            ["attach", "restore-version"].includes(p.intent))),
      onClick: p.intent ? () => ctx.action(p.intent, p.payload) : p.onClick,
    },
    p.children,
  );
}
export function Link(p: Props) {
  return h(
    "a",
    {
      className: cls(p, "link"),
      href: p.href,
      rel: "noopener noreferrer",
      target: String(p.href).startsWith("https://") ? "_blank" : undefined,
    },
    p.children,
  );
}
export function Checkbox(p: Props) {
  const ctx = useNotesContext();
  return h(
    "label",
    { className: "checkbox-control", title: p.label },
    h("input", {
      type: "checkbox",
      checked: !!p.checked,
      disabled:
        p.field === "note-template" &&
        (ctx.model.note?.permission === "viewer" || ctx.model.note?.deleted),
      "aria-label": p.label,
      onChange: (e: any) => ctx.field(p.field, e.target.checked, p.payload),
    }),
    p.label && ["note-template", "user-disabled"].includes(p.field)
      ? h("span", null, p.label)
      : null,
  );
}
export function Select(p: Props) {
  const ctx = useNotesContext();
  return h(
    "select",
    {
      className: cls(p, "select"),
      value: p.value || "",
      "aria-label": p.label || p.field,
      disabled:
        p.field === "note-notebook" && ctx.model.note?.permission !== "owner",
      onChange: (e: any) => ctx.field(p.field, e.target.value),
    },
    (p.options || []).map((o: any) =>
      h("option", { key: o.value, value: o.value }, o.label),
    ),
  );
}
export function Field(p: Props) {
  return h(
    "label",
    { className: "field" },
    h("span", null, p.label),
    p.children,
  );
}
export function Alert(p: Props) {
  return h(
    "div",
    {
      className: cls(p, "alert"),
      role: p.tone === "error" ? "alert" : "status",
    },
    p.children,
  );
}
export function Input(p: Props) {
  const ctx = useNotesContext();
  if (p.type === "editor") return h(EditorView);
  if (p.type === "dialog")
    return h(Dialog, { data: { app: ctx.model }, onAction: ctx.action });
  const disabled =
    (p.field === "title" || p.field?.startsWith("note-")) &&
    (ctx.model.note?.permission === "viewer" || ctx.model.note?.deleted);
  const props: any = {
    className: cls(p, "input"),
    type: p.type || "text",
    placeholder: p.placeholder,
    "aria-label": p.label || p.placeholder || p.field,
    readOnly: p.readOnly || disabled,
    onChange: (e: any) => ctx.field(p.field, e.target.value, p.payload),
    value: p.value ?? "",
  };
  if (p.field === "note-tags") {
    delete props.value;
    delete props.onChange;
    props.defaultValue = p.value;
    props.key = ctx.model.note?.id;
    props.onBlur = (e: any) => ctx.field(p.field, e.target.value);
  }
  if (p.variant === "title") return h(TitleInput, props);
  if (p.variant === "search")
    return h(
      "label",
      { className: "global-search" },
      h(icons.Search, { size: 21 }),
      h("input", props),
      h("span", { className: "search-hint" }, "Search"),
    );
  return h("input", props);
}
function TitleInput(props: Props) {
  const ref = useRef<HTMLTextAreaElement>(null);
  useLayoutEffect(() => {
    const el = ref.current;
    if (el) {
      const resize = () => {
        el.style.height = "auto";
        el.style.height = el.scrollHeight + "px";
      };
      resize();
      let width = el.clientWidth;
      const observer = new ResizeObserver(() => {
        if (el.clientWidth !== width) {
          width = el.clientWidth;
          resize();
        }
      });
      observer.observe(el);
      return () => observer.disconnect();
    }
  }, [props.value]);
  const { type, ...rest } = props;
  return h("textarea", { ...rest, ref, rows: 1 });
}
