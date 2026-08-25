// Renders TUTORIAL.md -> TUTORIAL.html with a dark theme and clickable,
// scroll-to-section table of contents.
//
// Usage: node scripts/build-tutorial.js
//
// Requires the `marked` dev dependency (npm install -D marked).

const { marked } = require("marked");
const fs = require("fs");
const path = require("path");

const ROOT = path.resolve(__dirname, "..");
const SRC = path.join(ROOT, "TUTORIAL.md");
const OUT = path.join(ROOT, "TUTORIAL.html");

// GitHub-style heading slug so the table-of-contents #anchors resolve:
// lowercase, strip punctuation in place (keeping the surrounding spaces), then
// turn EACH remaining space into one hyphen. A " — " therefore yields "--",
// which matches how GitHub/marked build the TOC links in the markdown.
function slug(text) {
  return text
    .toLowerCase()
    .replace(/[^\w\s-]/g, "")
    .replace(/ /g, "-");
}

// Give every heading an id derived from its text so in-page links scroll to it.
marked.use({
  renderer: {
    heading(token) {
      const level = token.depth;
      const html = this.parser.parseInline(token.tokens);
      const id = slug(token.text.trim());
      return `<h${level} id="${id}">${html}</h${level}>\n`;
    },
  },
});

const style = `<style>
:root{--bg:#0f1117;--surface:#1a1d27;--border:#2d3148;--text:#e2e4f0;--muted:#8b8fa8;--accent:#7c87ff;--code-bg:#1e2130;}
*{box-sizing:border-box;margin:0;padding:0;}
html{scroll-behavior:smooth;}
body{background:var(--bg);color:var(--text);font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",sans-serif;font-size:15px;line-height:1.75;}
.container{max-width:920px;margin:0 auto;padding:56px 36px 96px;}
h1{font-size:2.1rem;font-weight:700;color:#fff;margin-bottom:10px;line-height:1.2;scroll-margin-top:24px;}
h2{font-size:1.35rem;font-weight:600;color:var(--accent);margin:52px 0 16px;padding-bottom:10px;border-bottom:1px solid var(--border);scroll-margin-top:24px;}
h3{font-size:1.05rem;font-weight:600;color:#c5caff;margin:36px 0 12px;scroll-margin-top:24px;}
h4{font-size:0.85rem;font-weight:600;color:var(--muted);margin:24px 0 8px;text-transform:uppercase;letter-spacing:.07em;scroll-margin-top:24px;}
p{margin-bottom:16px;color:#cdd0e0;}
a{color:var(--accent);text-decoration:none;}a:hover{text-decoration:underline;}
code{background:#252836;color:#c5caff;padding:2px 7px;border-radius:4px;font-family:"JetBrains Mono","Fira Code",monospace;font-size:0.85em;}
pre{background:var(--code-bg);border:1px solid var(--border);border-radius:10px;padding:22px 26px;overflow-x:auto;margin:20px 0;}
pre code{background:none;padding:0;color:#dde1f7;font-size:0.87em;line-height:1.65;}
table{width:100%;border-collapse:collapse;margin:24px 0;font-size:0.9em;}
th{background:#252836;color:var(--accent);text-align:left;padding:11px 15px;font-weight:600;font-size:0.78em;text-transform:uppercase;letter-spacing:.06em;border-bottom:2px solid var(--border);}
td{padding:10px 15px;border-bottom:1px solid var(--border);color:#cdd0e0;vertical-align:top;}
tr:hover td{background:rgba(124,135,255,.05);}
ul,ol{padding-left:26px;margin-bottom:16px;}
li{margin-bottom:7px;color:#cdd0e0;}
blockquote{border-left:3px solid var(--accent);margin:20px 0;padding:14px 22px;background:rgba(124,135,255,.07);color:var(--muted);border-radius:0 7px 7px 0;}
hr{border:none;border-top:1px solid var(--border);margin:52px 0;}
.toc-back{position:fixed;bottom:24px;right:24px;background:var(--accent);color:#fff;padding:10px 14px;border-radius:8px;font-size:0.8rem;box-shadow:0 4px 12px rgba(0,0,0,.4);opacity:.85;}
.toc-back:hover{opacity:1;text-decoration:none;}
</style>`;

const body = marked.parse(fs.readFileSync(SRC, "utf8"));

const html = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8"/>
<meta name="viewport" content="width=device-width, initial-scale=1"/>
<title>admin-v2 Tutorial — Go · Gin · templ · sqlx · Datastar · Tailwind · Playwright</title>
${style}
</head>
<body>
<div class="container">
${body}
</div>
<a href="#table-of-contents" class="toc-back">↑ Contents</a>
</body>
</html>`;

fs.writeFileSync(OUT, html);
console.log(`Wrote ${OUT} (${body.length} bytes of body HTML)`);

// The docs/ copy gets a "Home" nav bar linking back to the site landing page.
const navBar = `<nav style="position:sticky;top:0;z-index:10;background:rgba(15,17,23,.9);backdrop-filter:blur(8px);border-bottom:1px solid #2d3148;padding:12px 24px;display:flex;align-items:center;gap:16px;">
  <a href="index.html" style="color:#7c87ff;text-decoration:none;font-weight:600;font-size:0.9rem;">&#8592; Home</a>
  <span style="color:#8b8fa8;font-size:0.85rem;">Full-Stack Tutorial</span>
</nav>`;
const docsHtml = html.replace(/<body>/, `<body>\n${navBar}`);

// Also emit a GitHub Pages copy at docs/tutorial.html so the full project
// tutorial is reachable from the docs landing page (docs/index.html, built by
// scripts/build-docs-site.js). .nojekyll tells Pages to serve files as-is.
const docsDir = path.join(ROOT, "docs");
fs.mkdirSync(docsDir, { recursive: true });
fs.writeFileSync(path.join(docsDir, "tutorial.html"), docsHtml);
fs.writeFileSync(path.join(docsDir, ".nojekyll"), "");
console.log(`Wrote ${path.join(docsDir, "tutorial.html")} and docs/.nojekyll for GitHub Pages`);
