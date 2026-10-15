import DOMPurify from "dompurify";
// Import the maintained upstream renderer directly, rather than recreating Joplin Markdown rules.
import MdToHtml from "./joplin-bridge.mjs";
import { url } from "../types";
const renderer = new MdToHtml({
  ResourceModel: { isResourceUrl: () => false },
  extraRendererRules: [
    {
      id: "notes_markdown15_compat",
      pluginId: "notes",
      assetPath: "",
      module: {
        plugin: (md: any) => {
          md.utils = { ...md.utils, assign: Object.assign };
        },
      },
    },
  ],
  pluginOptions: Object.fromEntries(
    [
      "mermaid",
      "katex",
      "abc",
      "fountain",
      "externalEmbed",
      "audioPlayer",
      "videoPlayer",
      "pdfViewer",
    ].map((key) => [key, { enabled: false }]),
  ),
});
DOMPurify.addHook("afterSanitizeAttributes", (node) => {
  if (node instanceof HTMLAnchorElement) {
    node.setAttribute("rel", "noopener noreferrer");
    if (node.getAttribute("href")?.startsWith("http"))
      node.setAttribute("target", "_blank");
  }
  if (node instanceof HTMLImageElement) {
    const src = node.getAttribute("src") || "";
    if (
      !src.startsWith(url("api/attachments/")) &&
      !src.startsWith("data:image/")
    )
      node.removeAttribute("src");
    node.setAttribute("loading", "lazy");
  }
});
export async function renderMarkdown(
  body: string,
  attachmentIDs: string[] = [],
) {
  // Resolve resource links through our authorized attachment route. Joplin note links
  // remain internal note IDs; this is presentation adaptation, not a sync implementation.
  body = body
    .replace(
      /(!\[[^\]]*\]\():\/([a-f0-9]{32})(\))/g,
      (_all, p, id, s) => p + url(`api/attachments/${id}?inline=1`) + s,
    )
    .replace(
      /(\[[^\]]*\]\():\/([a-f0-9]{32})(\))/g,
      (_all, p, id, s) =>
        p +
        (attachmentIDs.includes(id)
          ? url(`api/attachments/${id}`)
          : "#note=" + id) +
        s,
    );
  const result = await renderer.render(
    body,
    {},
    {
      splitted: true,
      checkboxDisabled: true,
      linkRenderingType: 2,
      resources: {},
    },
  );
  return DOMPurify.sanitize(result.html, {
    FORBID_TAGS: ["style", "script", "iframe", "object", "embed", "form"],
    FORBID_ATTR: ["style"],
    ADD_ATTR: ["target"],
    ALLOW_DATA_ATTR: false,
  });
}
