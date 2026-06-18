import snarkdown from "snarkdown";
import insane from "insane";

export function renderMarkdown(content) {
  return insane(snarkdown(content), {
    allowedTags: [
      "a", "b", "i", "em", "strong", "code", "pre", "blockquote",
      "p", "ul", "ol", "li", "br", "del", "hr",
    ],
    allowedAttributes: {
      a: ["href", "title", "target", "rel"],
    },
  });
}
