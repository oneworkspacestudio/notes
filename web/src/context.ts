import { createContext, useContext } from "react";
export interface ViewContext {
  model: any;
  action: (name: string, payload?: any) => void;
  field: (name: string, value: any, payload?: any) => void;
  registerEditor: (commands: Record<string, () => void>) => void;
}
export const NotesContext = createContext<ViewContext>(null!);
export const useNotesContext = () => useContext(NotesContext);
