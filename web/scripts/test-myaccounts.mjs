import assert from "node:assert/strict";
import { randomBytes, randomUUID } from "node:crypto";
import { spawn } from "node:child_process";
import { chromium } from "@playwright/test";
const issuer = process.env.NOTES_TEST_ISSUER,
  origin = process.env.NOTES_TEST_ORIGIN,
  password = process.env.NOTES_TEST_PASSWORD;
assert(
  issuer?.startsWith("http://localhost:") &&
    origin?.startsWith("http://127.0.0.1:") &&
    password,
);
const browser = await chromium.launch({
  executablePath:
    process.env.NOTES_CHROME ||
    "C:/Program Files/Google/Chrome/Application/chrome.exe",
});
let notes;
try {
  const admin = await browser.newContext();
  async function provider(path, body) {
    const r = await admin.request.fetch(issuer + path, {
      method: body ? "POST" : "GET",
      data: body,
      headers: { Origin: issuer },
    });
    assert.equal(r.status(), 200);
    const result = await r.json();
    assert.equal(
      result.status,
      "ok",
      "Provider operation failed: " + path.split("?")[0],
    );
    return result.data;
  }
  const base = {
    organization: "built-in",
    username: "admin@example.com",
    password,
    application: "app-built-in",
    signinMethod: "Password",
    type: "login",
  };
  await provider("/api/login", base);
  const firstPassword = randomBytes(32).toString("hex"),
    otherPassword = randomBytes(32).toString("hex");
  for (const [name, pass] of [
    ["notes-primary", firstPassword],
    ["notes-isolation", otherPassword],
  ])
    await provider("/api/add-user", {
      owner: "myaccounts",
      name,
      displayName: name,
      email: name + "@example.test",
      emailVerified: false,
      type: "normal-user",
      isAdmin: false,
      passwordType: "plain",
      password: pass,
      signupApplication: "myaccounts",
      createdTime: new Date().toISOString(),
      properties: {},
    });
  const relyingUser = {
    ...base,
    organization: "myaccounts",
    username: "notes-primary",
    password: firstPassword,
  };
  const template = await provider("/api/get-application?id=admin/myaccounts");
  const clientId = "notes-test-" + randomBytes(8).toString("hex"),
    clientSecret = randomBytes(32).toString("hex");
  await provider("/api/add-application", {
    ...template,
    name: clientId,
    displayName: "Notes disposable test",
    clientId,
    clientSecret,
    redirectUris: [origin + "/auth/callback"],
    grantTypes: ["authorization_code"],
    enableAutoSignin: false,
    enableSigninSession: true,
    enableSignUp: false,
    promptPage: [],
    providers: [],
    scopes: [],
  });
  notes = spawn(
    process.env.NOTES_TEST_EXE,
    ["--port", new URL(origin).port, "--data", process.env.NOTES_TEST_DATA],
    {
      windowsHide: true,
      env: {
        ...process.env,
        NOTES_OIDC_ISSUER: issuer,
        NOTES_OIDC_CLIENT_ID: clientId,
        NOTES_OIDC_CLIENT_SECRET: clientSecret,
      },
      stdio: "ignore",
    },
  );
  for (let i = 0; i < 100; i++) {
    try {
      if ((await fetch(origin + "/healthz")).ok) break;
    } catch {}
    await new Promise((r) => setTimeout(r, 100));
  }
  async function signIn(context, credentials = relyingUser) {
    const start = await context.request.get(origin + "/auth/login", {
      maxRedirects: 0,
    });
    assert.equal(start.status(), 302);
    const q = new URL(start.headers().location).searchParams;
    assert.equal(q.get("code_challenge_method"), "S256");
    const request = new URLSearchParams({
      clientId,
      responseType: "code",
      redirectUri: origin + "/auth/callback",
      scope: q.get("scope"),
      state: q.get("state"),
      nonce: q.get("nonce"),
      code_challenge: q.get("code_challenge"),
      code_challenge_method: "S256",
    });
    const code = await provider("/api/login?" + request, {
      ...credentials,
      type: "code",
      application: clientId,
    });
    const page = await context.newPage();
    await page.goto(
      origin +
        "/auth/callback?" +
        new URLSearchParams({ code, state: q.get("state") }),
    );
    const me = await (await context.request.get(origin + "/api/me")).json();
    assert(me.user?.id, "Notes rejected the real MyAccounts callback");
    await page.close();
    return me;
  }
  const me = await signIn(admin);
  const created = await admin.request.post(origin + "/api/notes", {
    data: {
      title: "Real provider note",
      body: "OIDC-isolated content",
      tags: [],
    },
    headers: { Origin: origin, "X-CSRF-Token": me.csrf },
  });
  assert.equal(created.status(), 201);
  const n = await created.json();
  const second = await browser.newContext();
  const same = await signIn(second);
  assert.equal(
    me.user.id,
    same.user.id,
    "Subject mapping changed between sessions",
  );
  const other = await browser.newContext();
  const otherMe = await signIn(other, {
    ...relyingUser,
    username: "notes-isolation",
    password: otherPassword,
  });
  assert.notEqual(
    otherMe.user.id,
    me.user.id,
    "Different provider subjects were merged",
  );
  assert.equal(
    (await other.request.get(origin + "/api/notes/" + n.id)).status(),
    404,
    "Cross-user note access was allowed",
  );
  const unauthenticated = await browser.newContext();
  assert.equal(
    (await unauthenticated.request.get(origin + "/api/notes/" + n.id)).status(),
    401,
  );
  assert.equal(
    (
      await admin.request.post(origin + "/api/logout", {
        headers: { Origin: origin, "X-CSRF-Token": me.csrf },
      })
    ).status(),
    200,
  );
  assert.equal(
    (await admin.request.get(origin + "/api/notes/" + n.id)).status(),
    401,
  );
  assert.equal(
    (await second.request.get(origin + "/api/notes/" + n.id)).status(),
    200,
  );
  console.log(
    "PASS: real provider callback, stable identity, distinct-subject note isolation, unauthenticated refusal and local logout",
  );
} finally {
  notes?.kill();
  await browser.close();
}
