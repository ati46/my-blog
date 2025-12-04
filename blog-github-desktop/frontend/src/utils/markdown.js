import MarkdownIt from "markdown-it";
import { applyHighlight } from "./highlight";

// 关闭原生 HTML 以避免潜在 XSS，保持链接自动识别和排版优化
const md = new MarkdownIt({
  html: false,
  linkify: true,
  typographer: true,
});

export function renderMarkdown(content) {
  const html = md.render(content || "");
  // 等 DOM 出来后自动做高亮
  setTimeout(applyHighlight, 10);
  return html;
}
