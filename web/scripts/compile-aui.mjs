import { readFile, writeFile } from "node:fs/promises";
import { parse, compileReact } from "@codedia/parser";
for (const name of ["Notes", "Dialog"]) {
  const source = await readFile(
    new URL(`../src/${name}.aui`, import.meta.url),
    "utf8",
  );
  const output = compileReact(parse(source));
  await writeFile(
    new URL(`../src/${name}.generated.tsx`, import.meta.url),
    `// @ts-nocheck -- @codedia/parser emits dynamic data contracts and untyped loop parameters.\n// Generated from ${name}.aui. Edit the source, then npm run generate.\n` +
      output,
  );
}
