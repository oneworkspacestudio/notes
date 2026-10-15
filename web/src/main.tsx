import { createElement as h } from "react";
import { createRoot } from "react-dom/client";
import { NotesContext } from "./context";
import { useNotes } from "./useNotes";
import { Notes } from "./Notes.generated";
import "./styles.css";
function App() {
  const ctx = useNotes();
  return h(
    NotesContext.Provider,
    { value: ctx },
    h(Notes, { data: { app: ctx.model }, onAction: ctx.action }),
  );
}
createRoot(document.getElementById("root")!).render(h(App));
