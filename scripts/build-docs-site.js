// Builds the docs/ GitHub Pages site from the uploaded tutorial documents.
//
//   node scripts/build-docs-site.js
//
// - Converts each .docx to HTML with mammoth (the .pdf tutorial is already
//   covered by the in-repo TUTORIAL.md, so it maps to the existing tutorial.html)
// - Scrubs identifiable / sensitive values, replacing them with neutral samples
// - Wraps each page in a dark theme with a "Home" nav bar and formatted code
// - Generates docs/index.html — a landing page linking to every tutorial page
//
// Requires the `mammoth` dev dependency.

const mammoth = require("mammoth");
const fs = require("fs");
const path = require("path");

const ROOT = path.resolve(__dirname, "..");
const DOCS = path.join(ROOT, "docs");
const UPLOADS = "/Users/ginad/.kiro/crew/uploads";

// ── Source documents → page metadata ───────────────────────────────────────
// order controls the landing-page card order.
const PAGES = [
  {
    slug: "go-basics",
    title: "Go Basics",
    blurb: "The Go language fundamentals — types, variables, functions, structs, slices, maps, and control flow.",
    src: `${UPLOADS}/c3c1af49d041422c908ee8b59dccf753_GO_Basics.docx`,
  },
  {
    slug: "go-stdlib",
    title: "Go Standard Library",
    blurb: "The common standard-library packages you reach for every day: fmt, strings, errors, os, encoding/json, and more.",
    src: `${UPLOADS}/25ae1c889d8d4439bfaff446c52f9de2_Go_Standard_Library__common_.docx`,
  },
  {
    slug: "gin-gonic",
    title: "Gin Gonic",
    blurb: "The Gin web framework — routing, middleware, request binding, and rendering.",
    src: `${UPLOADS}/084fb9cc6bc44fd4adbbb1c7ff752915_Gin-Gonic_Updated.docx`,
  },
  {
    slug: "go-gin-templ",
    title: "Go + Gin + templ",
    blurb: "Putting Gin together with templ for type-safe server-rendered HTML.",
    src: `${UPLOADS}/d1e070915d9d4828b46563072d41260e_Go-Gin-Templ.docx`,
  },
  {
    slug: "ent-sql",
    title: "Ent & SQL",
    blurb: "The Ent ORM and working with SQL databases in Go.",
    src: `${UPLOADS}/21acacba80e1434f99751e293c797db7_ENT-SQL.docx`,
  },
  {
    slug: "tutorial",
    title: "Full-Stack Tutorial (this project)",
    blurb: "The complete gin-templ-datastar walkthrough: the admin panel this repo builds, end to end.",
    // Generated separately from TUTORIAL.md by scripts/build-tutorial.js → tutorial.html
    src: null,
    href: "tutorial.html",
  },
];

// ── Code block extraction ───────────────────────────────────────────────────
// Word stores code samples inside single-column tables ("code boxes"). mammoth
// renders those as <table>…<br/>…</table>. Detect those and turn them into
// real <pre><code> blocks so they render as monospaced, non-wrapped code.
function decodeEntities(s) {
  return s
    .replace(/<br\s*\/?>/gi, "\n")
    .replace(/<\/p>\s*<p>/gi, "\n")
    .replace(/<[^>]+>/g, "")      // strip any remaining inline tags (<strong>, etc.)
    .replace(/&nbsp;/g, " ")
    .replace(/&amp;/g, "&")
    .replace(/&lt;/g, "<")
    .replace(/&gt;/g, ">")
    .replace(/&quot;/g, '"')
    .replace(/&#39;/g, "'");
}

function formatCodeBlocks(html) {
  return html.replace(/<table>[\s\S]*?<\/table>/gi, (table) => {
    // Heuristic: a "code box" is a table with exactly one column and content
    // that contains line breaks (multi-line code) or Go/HTML code tokens.
    const cellCount = (table.match(/<t[dh]\b/gi) || []).length;
    const looksLikeCode =
      /<br\s*\/?>/i.test(table) &&
      /(func |package |import |:=|fmt\.|<-|templ |go\.mod|SELECT |INSERT |type \w+ struct|router\.|c\.\w+\(|DATABASE_URL)/i.test(table);
    // Only convert single-column code boxes; leave real data tables alone.
    if (cellCount <= 2 && looksLikeCode) {
      const code = decodeEntities(table).replace(/\n{3,}/g, "\n\n").trim();
      const escaped = code
        .replace(/&/g, "&amp;")
        .replace(/</g, "&lt;")
        .replace(/>/g, "&gt;");
      return `<pre><code>${escaped}</code></pre>`;
    }
    return table; // genuine data table — keep as-is
  });
}

// Some docs store code as a RUN of consecutive <p> lines (one <p> per line),
// often introduced by a lone language-fence paragraph like <p>go</p>. Collapse
// those runs into a single <pre><code> block.
const FENCE = /^(go|bash|sh|html|sql|json|yaml|toml|dockerfile|makefile|js|ts|templ|text)$/i;
function stripTags(s) {
  return s.replace(/<[^>]+>/g, "")
    .replace(/&nbsp;/g, " ").replace(/&amp;/g, "&")
    .replace(/&lt;/g, "<").replace(/&gt;/g, ">")
    .replace(/&quot;/g, '"').replace(/&#39;/g, "'");
}
function isCodeLine(text) {
  const t = text.trim();
  if (t === "") return false;
  if (/^\s{2,}\S/.test(text)) return true;                 // indented
  // Standalone structural lines (closing braces/parens, opening a block).
  if (/^[\})\];]+[,;]?$/.test(t)) return true;
  if (/^(\{|\(|\[)$/.test(t)) return true;
  // KEY=VALUE env / config lines (all-caps key).
  if (/^[A-Z][A-Z0-9_]*=/.test(t)) return true;
  // Shell commands.
  if (/^(curl|go|npm|npx|air|templ|make|cd|export|git|docker|sqlite3|psql|mysql)\b/.test(t)) return true;
  // Go / templ / SQL code tokens.
  if (/(func |package |import |:=|fmt\.|gin\.|router\.|c\.\w+\(|templ\.|http\.\w|log\.\w|<-|return |var \w+ |const \w+ |type \w+ (struct|interface)|=\s*&?\w+\{|\}\s*\{|SELECT |INSERT |UPDATE |DELETE )/.test(t)) {
    // …but not a prose sentence that merely mentions a token: prose ends in a
    // period/colon and contains several spaces of ordinary words.
    const looksProse = /[.:]$/.test(t) && t.split(" ").length > 8 && !/[{};=]/.test(t);
    return !looksProse;
  }
  return false;
}
function collapseCodeParagraphs(html) {
  // Split into a flat list of top-level <p>…</p> and everything-else chunks.
  const parts = html.split(/(<p>[\s\S]*?<\/p>)/g).filter((s) => s !== "");
  const out = [];
  let i = 0;
  while (i < parts.length) {
    const part = parts[i];
    const pm = part.match(/^<p>([\s\S]*?)<\/p>$/);
    if (pm) {
      const text = stripTags(pm[1]);
      // A lone fence label starts a code run.
      const fenceStarts = FENCE.test(text.trim());
      if (fenceStarts || isCodeLine(text)) {
        const lines = [];
        if (!fenceStarts) lines.push(text); // the current line is code too
        let j = i + 1;
        // consume following <p> lines that still look like code
        while (j < parts.length) {
          const nm = parts[j].match(/^<p>([\s\S]*?)<\/p>$/);
          if (!nm) break;
          const t = stripTags(nm[1]);
          if (t.trim() === "" || isCodeLine(t) || /^[\})\];]/.test(t.trim())) {
            lines.push(t);
            j++;
          } else break;
        }
        if (lines.length >= 2 || (fenceStarts && lines.length >= 1)) {
          const code = lines.join("\n").replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;");
          out.push(`<pre><code>${code.trim()}</code></pre>`);
          i = j;
          continue;
        }
      }
    }
    out.push(part);
    i++;
  }
  return out.join("")
    // Drop any orphaned lone fence-label paragraphs (e.g. a stray <p>go</p>).
    .replace(/<p>\s*(go|bash|sh|html|sql|json|yaml|toml|dockerfile|makefile|js|ts|templ|text)\s*<\/p>/gi, "");
}

// ── Sensitive / identifiable value scrubbing ────────────────────────────────
// Replace real names, project names, and connection details with neutral samples.
function scrub(html) {
  return html
    // Real project/db name -> generic
    .replace(/openlisting/gi, "myapp")
    // Personal name used in code samples -> neutral sample name
    .replace(/Ginad@example\.com/gi, "sam@example.com")
    .replace(/\bGinad\b/g, "Alex")
    // Any DSN with embedded credentials -> obvious placeholder
    .replace(/([a-z]+):\/\/[^:@\/\s"<]+:[^@\/\s"<]+@[^\/\s"<]+/gi, "$1://user:password@localhost")
    .replace(/[a-z0-9_]+:[^@\s"<]+@tcp\([^)]+\)/gi, "user:password@tcp(localhost:3306)")
    // Any private/real-looking IP in examples -> documentation address
    .replace(/\b(?!127\.0\.0\.1)(?!0\.0\.0\.0)(?:\d{1,3}\.){3}\d{1,3}\b/g, "203.0.113.10");
}

// ── Page shell ──────────────────────────────────────────────────────────────
const STYLE = `<style>
:root{--bg:#0f1117;--surface:#1a1d27;--border:#2d3148;--text:#e2e4f0;--muted:#8b8fa8;--accent:#7c87ff;--code-bg:#1e2130;}
*{box-sizing:border-box;margin:0;padding:0;}
html{scroll-behavior:smooth;}
body{background:var(--bg);color:var(--text);font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",sans-serif;font-size:15px;line-height:1.75;}
.nav{position:sticky;top:0;z-index:10;background:rgba(15,17,23,.9);backdrop-filter:blur(8px);border-bottom:1px solid var(--border);padding:12px 24px;display:flex;align-items:center;gap:16px;}
.nav a{color:var(--accent);text-decoration:none;font-weight:600;font-size:0.9rem;}
.nav a:hover{text-decoration:underline;}
.nav .crumb{color:var(--muted);font-size:0.85rem;}
.container{max-width:920px;margin:0 auto;padding:40px 36px 96px;}
h1{font-size:2.5rem;font-weight:800;color:#fff;margin:28px 0 16px;line-height:1.2;letter-spacing:-0.01em;}
h2{font-size:1.45rem;font-weight:600;color:var(--accent);margin:44px 0 14px;padding-bottom:8px;border-bottom:1px solid var(--border);}
h3{font-size:1.15rem;font-weight:600;color:#c5caff;margin:30px 0 10px;}
h4,h5,h6{font-size:1rem;font-weight:600;color:var(--muted);margin:22px 0 8px;}
p{margin-bottom:14px;color:#cdd0e0;}
a{color:var(--accent);}
strong{color:#eef0fb;}
code{background:#252836;color:#c5caff;padding:2px 6px;border-radius:4px;font-family:"JetBrains Mono","Fira Code",ui-monospace,SFMono-Regular,Menlo,monospace;font-size:0.9em;font-weight:400;}
pre{background:var(--code-bg);border:1px solid var(--border);border-radius:10px;padding:18px 22px;overflow-x:auto;margin:18px 0;}
pre code{background:none;padding:0;color:#e6e9f5;font-size:0.95rem;line-height:1.7;white-space:pre;font-weight:400;}
pre code strong, pre code b{font-weight:400;color:inherit;}
table{width:100%;border-collapse:collapse;margin:20px 0;font-size:0.9em;}
th{background:#252836;color:var(--accent);text-align:left;padding:10px 14px;font-weight:600;font-size:0.8em;border-bottom:2px solid var(--border);}
td{padding:9px 14px;border-bottom:1px solid var(--border);color:#cdd0e0;vertical-align:top;}
ul,ol{padding-left:26px;margin-bottom:14px;}
li{margin-bottom:6px;color:#cdd0e0;}
blockquote{border-left:3px solid var(--accent);margin:18px 0;padding:12px 18px;background:rgba(124,135,255,.07);color:var(--muted);border-radius:0 6px 6px 0;}
img{max-width:100%;border-radius:8px;}
hr{border:none;border-top:1px solid var(--border);margin:36px 0;}
</style>`;

function pageShell(title, bodyHtml) {
  return `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8"/>
<meta name="viewport" content="width=device-width, initial-scale=1"/>
<title>${title} — gin-templ-datastar</title>
${STYLE}
</head>
<body>
<nav class="nav">
  <a href="index.html">&#8592; Home</a>
  <span class="crumb">${title}</span>
</nav>
<div class="container">
<h1 class="page-title">${title}</h1>
${bodyHtml}
</div>
</body>
</html>`;
}

// ── Convert + write each doc page ───────────────────────────────────────────
(async () => {
  fs.mkdirSync(DOCS, { recursive: true });

  for (const p of PAGES) {
    if (!p.src) continue; // tutorial.html is produced by build-tutorial.js
    const { value } = await mammoth.convertToHtml({ path: p.src });
    // Demote any document-internal <h1> to <h2> so the injected page title is
    // the single, largest heading; the doc's own top-level headings sit under it.
    const demoted = value.replace(/<(\/?)h1(\s[^>]*)?>/gi, "<$1h2$2>");
    const body = scrub(collapseCodeParagraphs(formatCodeBlocks(demoted)));
    fs.writeFileSync(path.join(DOCS, `${p.slug}.html`), pageShell(p.title, body));
    console.log(`Wrote docs/${p.slug}.html`);
  }

  // ── Landing page ──────────────────────────────────────────────────────────
  const cards = PAGES.map((p) => {
    const href = p.href || `${p.slug}.html`;
    return `    <a class="card" href="${href}">
      <h3>${p.title}</h3>
      <p>${p.blurb}</p>
      <span class="go">Open &#8594;</span>
    </a>`;
  }).join("\n");

  const indexBody = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8"/>
<meta name="viewport" content="width=device-width, initial-scale=1"/>
<title>Golang Tutorial</title>
${STYLE}
<style>
.hero{padding:56px 0 28px;text-align:center;}
.hero h1{font-size:2.4rem;margin-bottom:10px;}
.hero p{color:var(--muted);font-size:1.05rem;max-width:640px;margin:0 auto;}
.grid{display:grid;grid-template-columns:repeat(auto-fill,minmax(280px,1fr));gap:18px;margin-top:36px;}
.card{display:flex;flex-direction:column;gap:8px;background:var(--surface);border:1px solid var(--border);border-radius:12px;padding:22px;text-decoration:none;transition:border-color .15s,transform .15s;}
.card:hover{border-color:var(--accent);transform:translateY(-2px);text-decoration:none;}
.card h3{color:#fff;margin:0;font-size:1.1rem;}
.card p{color:var(--muted);font-size:0.9rem;margin:0;flex:1;}
.card .go{color:var(--accent);font-size:0.85rem;font-weight:600;margin-top:6px;}
.footer{text-align:center;color:var(--muted);font-size:0.8rem;margin-top:56px;}
.footer a{color:var(--accent);}
</style>
</head>
<body>
<div class="container">
  <div class="hero">
    <h1>Golang Tutorial</h1>
    <p>A hands-on guide to server-rendered Go — the language basics, the standard library, Gin, templ, Datastar, and Ent/SQL. Pick a topic to begin.</p>
  </div>
  <div class="grid">
${cards}
  </div>
  <div class="footer">
    Source: <a href="https://github.com/danegigi/gin-templ-datastar">github.com/danegigi/gin-templ-datastar</a>
  </div>
</div>
</body>
</html>`;

  fs.writeFileSync(path.join(DOCS, "index.html"), indexBody);
  console.log("Wrote docs/index.html (landing page)");
})();
