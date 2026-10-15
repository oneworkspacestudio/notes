// Explicitly load upstream's compiled CommonJS artifact. Do not compile their
// source using our application's stricter ES module TypeScript settings.
import MdToHtml from "@joplin/renderer/MdToHtml.js";
export default MdToHtml;
