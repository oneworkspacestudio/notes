import assert from "node:assert/strict";
import { spawn, spawnSync } from "node:child_process";
import { mkdtemp, mkdir, stat, writeFile } from "node:fs/promises";
import path from "node:path";
import os from "node:os";
import { chromium } from "@playwright/test";
const root =
  process.env.NOTES_TEST_ROOT || path.join(os.tmpdir(), "notes-tests");
await mkdir(root, { recursive: true });
const temp = await mkdtemp(path.join(root, "portable-"));
const exe = path.resolve("../dist/MyNotes.exe"),
  origin = "http://127.0.0.1:18898";
const browser = await chromium.launch({
  executablePath:
    process.env.NOTES_CHROME ||
    "C:/Program Files/Google/Chrome/Application/chrome.exe",
});
let processHandle;
const measurements = {};
async function start(data, prefix = "") {
  const started = performance.now();
  processHandle = spawn(
    exe,
    [
      "--local",
      "--port",
      "18898",
      "--data",
      data,
      "--public-url",
      origin + prefix,
    ],
    { windowsHide: true },
  );
  const login = await new Promise((resolve, reject) => {
    const timer = setTimeout(
      () => reject(new Error("Portable server timeout")),
      15000,
    );
    processHandle.on("error", reject);
    processHandle.stdout.on("data", (b) => {
      const m = b
        .toString()
        .match(/http:\/\/127\.0\.0\.1:18898[^\s]*token=[a-f0-9]+/);
      if (m) {
        clearTimeout(timer);
        resolve(m[0]);
      }
    });
  });
  measurements.startup_ms = Math.round(performance.now() - started);
  return login;
}
async function stop() {
  const child = processHandle;
  processHandle = null;
  if (!child || child.exitCode !== null) return;
  const exited = new Promise((resolve) => child.once("exit", resolve));
  child.kill();
  await exited;
}
function command(args) {
  const result = spawnSync(exe, args, { windowsHide: true, encoding: "utf8" });
  assert.equal(result.status, 0, result.stderr || "Portable command failed");
}
try {
  const data = path.join(temp, "data");
  const context = await browser.newContext();
  const page = await context.newPage();
  await page.goto(await start(data));
  let me = await (await context.request.get(origin + "/api/me")).json();
  const headers = { Origin: origin, "X-CSRF-Token": me.csrf };
  let response = await context.request.post(origin + "/api/notes", {
    data: {
      title: "Survives a restart",
      body: "Persistent portable note",
      tags: ["portable"],
    },
    headers,
  });
  assert.equal(response.status(), 201);
  const note = await response.json();
  response = await context.request.post(
    origin + "/api/notes/" + note.id + "/attachments",
    {
      multipart: {
        file: {
          name: "proof.txt",
          mimeType: "text/plain",
          buffer: Buffer.from("Persistent attachment"),
        },
      },
      headers,
    },
  );
  assert.equal(response.status(), 201);
  const attachment = await response.json();
  await stop();
  await page.goto(await start(data));
  response = await context.request.get(origin + "/api/notes/" + note.id);
  assert.equal(response.status(), 200);
  assert.equal((await response.json()).body, "Persistent portable note");
  assert.equal(
    await (
      await context.request.get(origin + "/api/attachments/" + attachment.id)
    ).text(),
    "Persistent attachment",
  );
  await stop();
  const archive = path.join(temp, "backup.zip"),
    restored = path.join(temp, "restored");
  command(["--local", "--data", data, "--file", archive, "backup"]);
  command(["--local", "--data", restored, "--file", archive, "restore"]);
  const staleCookies = await context.cookies();
  const login = await start(restored, "/notes");
  const old = await browser.newContext();
  await old.addCookies(staleCookies);
  assert.equal(
    (await old.request.get(origin + "/notes/api/me")).status(),
    401,
    "Restored sessions were not invalidated",
  );
  const errors = [];
  const prefixed = await browser.newContext();
  const p = await prefixed.newPage();
  p.on("pageerror", (e) => errors.push(e.message));
  await p.goto(login);
  await p.getByRole("button", { name: "Compose note" }).waitFor();
  assert.equal(
    (
      await prefixed.request.get(origin + "/notes/api/notes/" + note.id)
    ).status(),
    200,
  );
  assert.equal(
    await (
      await prefixed.request.get(
        origin + "/notes/api/attachments/" + attachment.id,
      )
    ).text(),
    "Persistent attachment",
  );
  assert.deepEqual(errors, []);
  me = await (await prefixed.request.get(origin + "/notes/api/me")).json();
  const malicious =
    '<h1>Imported HTML</h1><p><strong>Keep this</strong></p><script>alert(1)</script><a href="javascript:alert(2)">Unsafe link</a>';
  response = await prefixed.request.post(origin + "/notes/api/import", {
    multipart: {
      file: {
        name: "safe.html",
        mimeType: "text/html",
        buffer: Buffer.from(malicious),
      },
    },
    headers: { Origin: origin, "X-CSRF-Token": me.csrf },
  });
  assert.equal(response.status(), 201);
  const imported = await (
    await prefixed.request.get(origin + "/notes/api/notes?q=Imported")
  ).json();
  assert.equal(imported.notes.length, 1);
  assert(
    !/javascript:|<script|alert\(1\)/.test(imported.notes[0].body),
    "HTML import retained active content",
  );
  measurements.executable_bytes = (await stat(exe)).size;
  measurements.backup_bytes = (await stat(archive)).size;
  await mkdir("../test-results", { recursive: true });
  await writeFile(
    "../test-results/portable.json",
    JSON.stringify({ passed: true, ...measurements }, null, 2),
  );
  console.log(
    "PASS: native process restart, persisted attachments, offline CLI backup/restore, session invalidation, /notes proxy prefix, safe HTML import",
  );
  console.log(JSON.stringify(measurements));
} finally {
  await stop();
  await browser.close();
}
