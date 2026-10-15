export interface Note {
  id: string;
  owner: string;
  notebook_id: string;
  title: string;
  body: string;
  tags: string[];
  starred: boolean;
  pinned: boolean;
  archived: boolean;
  deleted: boolean;
  template: boolean;
  reminder: number;
  created: number;
  updated: number;
  revision: number;
  permission?: string;
}
export interface Notebook {
  id: string;
  owner: string;
  name: string;
  parent_id: string;
  permission: string;
}
export interface User {
  id: string;
  name: string;
  email: string;
  avatar: string;
  admin: boolean;
}
export interface Attachment {
  id: string;
  note_id: string;
  name: string;
  mime: string;
  size: number;
}
export const base =
  document.querySelector<HTMLMetaElement>('meta[name="notes-base"]')?.content ||
  "/";
export const url = (path: string) => base + path.replace(/^\//, "");
export const blankNote = (): Note => ({
  id: "",
  owner: "",
  notebook_id: "",
  title: "",
  body: "",
  tags: [],
  starred: false,
  pinned: false,
  archived: false,
  deleted: false,
  template: false,
  reminder: 0,
  created: 0,
  updated: 0,
  revision: 0,
});
export class APIError extends Error {
  constructor(
    public status: number,
    message: string,
    public data: any,
  ) {
    super(message);
  }
}
let csrf = "";
export function setCSRF(s: string) {
  csrf = s;
}
export async function api<T = any>(
  path: string,
  method = "GET",
  body?: any,
): Promise<T> {
  const multipart = body instanceof FormData;
  const res = await fetch(url(path), {
    method,
    credentials: "same-origin",
    headers: {
      ...(method !== "GET" ? { "X-CSRF-Token": csrf } : {}),
      ...(!multipart && body !== undefined
        ? { "Content-Type": "application/json" }
        : {}),
    },
    body:
      body === undefined ? undefined : multipart ? body : JSON.stringify(body),
  });
  const data = await res.json();
  if (!res.ok) {
    if (res.status === 401)
      window.dispatchEvent(new Event("notes-session-expired"));
    throw new APIError(res.status, data.error || "Request failed", data);
  }
  return data;
}
