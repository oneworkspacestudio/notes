import { test, expect, type BrowserContext } from "@playwright/test";
import { spawn, type ChildProcess } from "node:child_process";
import { mkdir, mkdtemp } from "node:fs/promises";
import path from "node:path";
import os from "node:os";
let server: ChildProcess, state: any;
const origin = "http://127.0.0.1:18895";
test.beforeAll(async ({ browser }) => {
  const parent =
    process.env.NOTES_TEST_ROOT || path.join(os.tmpdir(), "notes-tests");
  await mkdir(parent, { recursive: true });
  const dir = await mkdtemp(path.join(parent, "browser-"));
  server = spawn(
    path.resolve("../dist/MyNotes.exe"),
    ["--local", "--data", dir, "--port", "18895"],
    { windowsHide: true, stdio: ["pipe", "pipe", "pipe"] },
  );
  const login = await new Promise<string>((resolve, reject) => {
    const timer = setTimeout(
      () => reject(new Error("Server did not provide sign-in link")),
      20000,
    );
    server.stdout!.on("data", (data) => {
      const match = data
        .toString()
        .match(/http:\/\/127\.0\.0\.1:18895\/auth\/local\?token=[a-f0-9]+/);
      if (match) {
        clearTimeout(timer);
        resolve(match[0]);
      }
    });
    server.on("error", reject);
    server.on("exit", (code) => {
      if (code) reject(new Error("Server exited " + code));
    });
  });
  const context = await browser.newContext();
  const page = await context.newPage();
  await page.goto(login);
  await expect(
    page.getByRole("button", { name: "Compose note" }),
  ).toBeVisible();
  state = await context.storageState();
  await context.close();
});
test.afterAll(() => {
  server?.kill();
});
test.beforeEach(async ({ context, page }) => {
  await context.addCookies(state.cookies);
  await page.goto("/");
  await expect(
    page.getByRole("button", { name: "Compose note" }),
  ).toBeVisible();
});
async function api(
  context: BrowserContext,
  path: string,
  method = "GET",
  data?: any,
) {
  const me = await (await context.request.get(origin + "/api/me")).json();
  const res = await context.request.fetch(origin + path, {
    method,
    data,
    headers: { Origin: origin, "X-CSRF-Token": me.csrf },
  });
  expect(res.ok(), await res.text()).toBeTruthy();
  return res.json();
}

test("editing, autosave, tags, indexed search, archive and restore", async ({
  page,
  context,
}) => {
  await page.getByRole("button", { name: "Compose note" }).click();
  await page.getByPlaceholder("Untitled note").fill("A considered workspace");
  await page.getByRole("button", { name: "Markdown", exact: true }).click();
  await page
    .getByRole("textbox", { name: "Markdown content" })
    .fill(
      "## A clearer day\n\nCapture the sapphire idea before it disappears.\n\n- [ ] Sketch the first version\n- [x] Make room to focus\n\n| Priority | Status |\n| --- | --- |\n| Notes | In progress |",
    );
  await expect(page.locator(".editor-status")).toContainText(
    "All changes saved",
  );
  await page.reload();
  await expect(page.getByPlaceholder("Untitled note")).toHaveValue(
    "A considered workspace",
  );
  await page.getByRole("button", { name: "Note details and history" }).click();
  await page.getByPlaceholder("work, ideas, personal").fill("ideas, product");
  await page
    .getByRole("heading", { name: "Note details", exact: true })
    .click();
  await page.getByRole("button", { name: "Close dialog" }).click();
  await expect(page.locator(".editor-status")).toContainText(
    "All changes saved",
  );
  await page.getByPlaceholder("Search your notes").fill("sapphire");
  await expect(page.locator(".note-card")).toHaveCount(1);
  await page.getByPlaceholder("Search your notes").fill("");
  await page.getByRole("button", { name: "Star note", exact: true }).click();
  await page.getByRole("button", { name: "Starred", exact: true }).click();
  await expect(page.locator(".note-card")).toHaveCount(1);
  await page.getByRole("button", { name: "Archive or restore note" }).click();
  await expect(page.locator(".note-card")).toHaveCount(0);
  await page.getByRole("button", { name: "Archive", exact: true }).click();
  await expect(page.locator(".note-card")).toHaveCount(1);
  await page.getByRole("button", { name: "Archive or restore note" }).click();
  await page
    .getByRole("button", { name: "All notes", exact: false })
    .first()
    .click();
  await expect(page.locator(".note-card")).toHaveCount(1);
  await page.getByRole("button", { name: "Preview", exact: true }).click();
  await expect(page.locator(".markdown-preview")).toContainText(
    "A clearer day",
  );
  await expect(page.locator(".markdown-preview table")).toBeVisible();
  await expect(
    page.locator(".markdown-preview input[type=checkbox]"),
  ).toHaveCount(2);
  await expect(page.locator(".markdown-preview")).not.toContainText(
    "Preview unavailable",
  );
  const data = await api(context, "/api/notes?q=product");
  expect(data.notes).toHaveLength(1);
});

test("offline drafts reconnect and conflicts preserve unsaved work", async ({
  page,
  context,
}) => {
  const n = await api(context, "/api/notes", "POST", {
    title: "Draft recovery",
    body: "Server original",
    tags: [],
  });
  await page.reload();
  await page
    .locator(".note-card")
    .filter({ hasText: "Draft recovery" })
    .getByRole("button", { name: /Draft recovery/ })
    .click();
  await page.getByRole("button", { name: "Markdown", exact: true }).click();
  await context.setOffline(true);
  await page
    .getByRole("textbox", { name: "Markdown content" })
    .fill("An offline thought worth keeping");
  await expect(page.locator(".editor-status")).toContainText("Offline");
  await expect
    .poll(() =>
      page.evaluate(() =>
        Object.keys(localStorage).some((k) => k.startsWith("notes-draft:")),
      ),
    )
    .toBeTruthy();
  await context.setOffline(false);
  await expect(page.locator(".editor-status")).toContainText(
    "All changes saved",
  );
  expect((await api(context, `/api/notes/${n.id}`)).body).toBe(
    "An offline thought worth keeping",
  );
  const current = await api(context, `/api/notes/${n.id}`);
  await api(context, `/api/notes/${n.id}`, "PUT", {
    ...current,
    body: "Changed in another session",
  });
  await page
    .getByRole("textbox", { name: "Markdown content" })
    .fill("Keep my conflicting local draft");
  await expect(
    page.getByText("Your draft is safe. Resolve the conflict before saving."),
  ).toBeVisible();
  await page.getByRole("button", { name: "Save draft as a copy" }).click();
  await expect(page.getByPlaceholder("Untitled note")).toHaveValue(
    "Draft recovery (recovered)",
  );
  expect((await api(context, "/api/notes?q=conflicting")).notes).toHaveLength(
    1,
  );
});

test("notebooks, history, trash, attachments and browser error boundaries", async ({
  page,
  context,
}) => {
  const errors: string[] = [];
  page.on("pageerror", (e) => errors.push(e.message));
  await page.getByRole("button", { name: "New notebook", exact: true }).click();
  await page.getByPlaceholder("Notebook name").fill("Research journal");
  await page.getByRole("button", { name: "Save notebook" }).click();
  await page
    .getByRole("button", { name: "Research journal", exact: true })
    .click();
  await page.getByRole("button", { name: "Compose note" }).click();
  await page.getByPlaceholder("Untitled note").fill("Research log");
  await page.getByRole("button", { name: "Markdown", exact: true }).click();
  await page
    .getByRole("textbox", { name: "Markdown content" })
    .fill("First revision");
  await expect(page.locator(".editor-status")).toContainText(
    "All changes saved",
  );
  await page
    .getByRole("textbox", { name: "Markdown content" })
    .fill("Second revision");
  await expect(page.locator(".editor-status")).toContainText(
    "All changes saved",
  );
  await page.getByRole("button", { name: "Note details and history" }).click();
  await page.getByRole("button", { name: "Version history" }).click();
  await expect(page.locator(".history-row")).not.toHaveCount(0);
  await page.getByRole("button", { name: "Close dialog" }).click();
  const chooser = page.waitForEvent("filechooser");
  await page.getByRole("button", { name: "Attach file or image" }).click();
  await (
    await chooser
  ).setFiles({
    name: "research.txt",
    mimeType: "text/plain",
    buffer: Buffer.from("Research attachment"),
  });
  await expect(
    page.getByRole("textbox", { name: "Markdown content" }),
  ).toHaveValue(/research\.txt/);
  await expect(page.locator(".editor-status")).toContainText(
    "All changes saved",
  );
  await page.getByRole("button", { name: "Move to trash or restore" }).click();
  await expect(page.getByText("This note is in Trash.")).toBeVisible();
  await page.getByRole("button", { name: "Restore note", exact: true }).click();
  await expect(page.getByText("This note is in Trash.")).toHaveCount(0);
  expect(errors).toEqual([]);
});

test("typing during a note action preserves the newer draft", async ({
  page,
  context,
}) => {
  const n = await api(context, "/api/notes", "POST", {
    title: "Concurrent local edits",
    body: "Original content",
    tags: [],
  });
  await page.goto("/#note=" + n.id);
  await page.reload();
  await expect(page.getByPlaceholder("Untitled note")).toHaveValue(n.title);
  await page.getByRole("button", { name: "Markdown", exact: true }).click();
  let began!: () => void;
  const requested = new Promise<void>((resolve) => {
    began = resolve;
  });
  await page.route("**/api/notes/*", async (route) => {
    if (route.request().method() === "PUT") {
      began();
      await new Promise((resolve) => setTimeout(resolve, 250));
    }
    await route.continue();
  });
  await page.getByRole("button", { name: "Star note", exact: true }).click();
  await requested;
  await page
    .getByRole("textbox", { name: "Markdown content" })
    .fill("Keep the text typed while starring");
  await expect(page.locator(".editor-status")).toContainText(
    "All changes saved",
  );
  const saved = await api(context, "/api/notes/" + n.id);
  expect(saved.body).toBe("Keep the text typed while starring");
  expect(saved.starred).toBe(true);
});

test("desktop, mobile and dark mode visual checks", async ({
  page,
  context,
}) => {
  await api(context, "/api/notebooks", "POST", { name: "Personal" });
  const project = await api(context, "/api/notebooks", "POST", {
    name: "Workspace",
  });
  const demo = await api(context, "/api/notes", "POST", {
    notebook_id: project.id,
    title: "A workspace that feels like home",
    body: "A little less noise. A little more room to think.\n\n## The idea\n\nBring the things we use every day into one calm, connected workspace. A place for our notes, files and plans — with the freedom to make it our own.\n\n## What matters\n\n- [x] Keep our files on our own computer\n- [x] One account, wherever we work\n- [ ] Make the small details feel effortless\n\n> Good tools should make space for the work, and then get out of the way.\n\n## Next up\n\nA thoughtful first release, a few good conversations, and a fresh page.",
    tags: ["product", "ideas"],
    starred: true,
    pinned: true,
  });
  for (const [title, body, tags] of [
    [
      "Little things worth remembering",
      "The good ideas often arrive when you least expect them.",
      ["personal"],
    ],
    [
      "Friday · a fresh perspective",
      "A few thoughts from our weekly catch-up, and a clearer plan.",
      ["work"],
    ],
    [
      "Reading list for slower days",
      "Books, essays, and things to return to when there is room.",
      ["reading"],
    ],
    [
      "The shape of the next release",
      "A focused plan for the next small step.",
      ["product"],
    ],
  ] as any[])
    await api(context, "/api/notes", "POST", {
      notebook_id: project.id,
      title,
      body,
      tags,
    });
  await page.goto("/#note=" + demo.id);
  await page.reload();
  await expect(page.getByPlaceholder("Untitled note")).toHaveValue(demo.title);
  await expect(page.locator(".rich-editor")).toContainText("The idea");
  await expect(page.locator(".rich-editor")).toContainText("Next up");
  await page.getByRole("button", { name: "Workspace", exact: true }).click();
  await page.waitForTimeout(500);
  await mkdir("../docs/screenshots", { recursive: true });
  await page.screenshot({ path: "../docs/screenshots/desktop.png" });
  expect(
    await page.evaluate(
      () => document.documentElement.scrollWidth <= innerWidth,
    ),
  ).toBeTruthy();
  await page.getByRole("button", { name: "Toggle dark theme" }).click();
  await expect(page.locator("html")).toHaveAttribute("data-theme", "dark");
  await page.screenshot({ path: "../docs/screenshots/dark.png" });
  await page.setViewportSize({ width: 390, height: 844 });
  await page.getByRole("button", { name: "Back to notes" }).click();
  await expect(page.locator(".note-list")).toBeVisible();
  await page
    .locator(".note-card")
    .filter({ hasText: demo.title })
    .getByRole("button", { name: new RegExp(demo.title) })
    .click();
  await expect(page.locator(".editor")).toBeVisible();
  await expect
    .poll(() =>
      page
        .getByPlaceholder("Untitled note")
        .evaluate((el) => el.scrollHeight <= el.clientHeight + 1),
    )
    .toBeTruthy();
  await page.screenshot({ path: "../docs/screenshots/mobile.png" });
  expect(
    await page.evaluate(
      () => document.documentElement.scrollWidth <= innerWidth,
    ),
  ).toBeTruthy();
});
