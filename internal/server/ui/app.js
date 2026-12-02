"use strict";

const $ = (id) => document.getElementById(id);
const token = new URLSearchParams(location.search).get("token") || "";
const withToken = (p) => (token ? p + (p.includes("?") ? "&" : "?") + "token=" + encodeURIComponent(token) : p);
const api = async (p) => {
  const res = await fetch(withToken(p));
  const body = await res.json();
  if (!res.ok) throw new Error(body.error || res.statusText);
  return body;
};
const esc = (s) => String(s ?? "").replace(/[&<>"']/g, (c) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" }[c]));
const svgNS = "http://www.w3.org/2000/svg";
const el = (tag, attrs = {}) => {
  const n = document.createElementNS(svgNS, tag);
  for (const [k, v] of Object.entries(attrs)) n.setAttribute(k, v);
  return n;
};
const reduced = matchMedia("(prefers-reduced-motion: reduce)").matches;
const day = (iso) => new Date(iso).toLocaleDateString(undefined, { year: "numeric", month: "short", day: "numeric" });
const plural = (n, one, many) => n + " " + (n === 1 ? one : many);
const kind = (a) => (a.band === "unlikely" ? "" : a.silent ? "silent" : "announced");
const COLORS = { likely: "#b3263e", possible: "#c98a1b", unlikely: "#b8c2cf" };

let report = null;

async function load(refresh) {
  const btn = $("rescan");
  btn.setAttribute("aria-busy", "true");
  btn.textContent = "Reading history";
  try {
    report = await api("/api/v1/report" + (refresh ? "?refresh=1" : ""));
    const silent = report.silent;
    $("case").textContent = silent ? plural(silent, "silent fix", "silent fixes") + " on file" : "No silent fixes on file";
  } catch (err) {
    $("case").textContent = "Could not read the repository: " + err.message;
    report = null;
  } finally {
    btn.removeAttribute("aria-busy");
    btn.textContent = "Rescan history";
  }
  route();
}

function show(page) {
  document.querySelectorAll(".page").forEach((p) => { p.hidden = p.id !== "page-" + page; });
  const navPage = page === "commit" ? "commits" : page;
  document.querySelectorAll("nav a").forEach((a) => a.classList.toggle("on", a.dataset.page === navPage));
}

function route() {
  const [, page = "overview", arg] = location.hash.split("/");
  const known = ["overview", "commits", "commit", "timeline", "hotspots", "signals", "about"];
  const target = known.includes(page) ? page : "overview";
  show(target);
  $("main").focus({ preventScroll: true });
  window.scrollTo(0, 0);
  if (target === "signals") return renderSignals();
  if (target === "about") return;
  if (!report) return;
  ({ overview: renderOverview, commits: renderCommits, commit: () => renderCommit(arg),
     timeline: renderTimeline, hotspots: renderHotspots })[target]();
}

/* ---------- overview ---------- */

function renderOverview() {
  const flagged = report.assessments.filter((a) => a.band !== "unlikely");
  const announced = flagged.length - report.silent;
  $("h-overview").textContent = report.silent
    ? plural(report.silent, "commit", "commits") + " fixed something without saying so"
    : "Nothing in this history looks like a hidden fix";
  $("lede-sub").textContent = `Read ${plural(report.scanned, "commit", "commits")} in ${fmtDuration(report.elapsed)}. ` +
    (announced ? `${plural(announced, "other fix was", "other fixes were")} announced.` : "None of the fixes were announced.");
  drawLineup();
  renderSuspects(flagged);
  drawDonut();
  renderHotBars($("mini-hot"), report.hotspots.slice(0, 5));
}

function fmtDuration(ns) {
  const ms = ns / 1e6;
  return ms < 1000 ? Math.round(ms) + " ms" : (ms / 1000).toFixed(1) + " s";
}

function drawLineup() {
  const svg = $("lineup");
  svg.replaceChildren();
  const W = svg.clientWidth || 900, H = 150, mid = 78, pad = 22;
  svg.setAttribute("viewBox", `0 0 ${W} ${H}`);
  const commits = [...report.assessments].sort((a, b) => new Date(a.commit.when) - new Date(b.commit.when));
  if (!commits.length) return;
  const t0 = new Date(commits[0].commit.when).getTime();
  const t1 = new Date(commits[commits.length - 1].commit.when).getTime() || t0 + 1;
  const x = (a) => pad + ((new Date(a.commit.when).getTime() - t0) / Math.max(1, t1 - t0)) * (W - pad * 2);

  svg.append(el("line", { x1: pad, x2: W - pad, y1: mid, y2: mid, stroke: "#3a4659", "stroke-width": 2, "stroke-linecap": "round" }));
  // month ticks
  const start = new Date(t0); start.setUTCDate(1);
  for (let d = new Date(start); d.getTime() <= t1; d.setUTCMonth(d.getUTCMonth() + 1)) {
    if (d.getTime() < t0) continue;
    const tx = pad + ((d.getTime() - t0) / Math.max(1, t1 - t0)) * (W - pad * 2);
    svg.append(el("line", { x1: tx, x2: tx, y1: mid + 8, y2: mid + 14, stroke: "#4a5668" }));
    const label = el("text", { x: tx, y: H - 8, "text-anchor": "middle", fill: "#7d8999", "font-size": 11 });
    label.textContent = d.toLocaleDateString(undefined, { month: "short" });
    svg.append(label);
  }

  let flip = false;
  commits.forEach((a, i) => {
    const cx = x(a);
    const k = kind(a);
    const r = k ? 5 + a.score / 14 : 3.5;
    const g = el("g", { class: "bead-group" });
    if (k === "silent") {
      flip = !flip;
      const ly = flip ? 22 : 128;
      g.append(el("line", { x1: cx, x2: cx, y1: mid + (flip ? -r - 3 : r + 3), y2: flip ? ly + 6 : ly - 14, stroke: "#f5d547", "stroke-width": 1, "stroke-dasharray": "2 3" }));
      const t = el("text", { x: cx, y: flip ? ly : ly, "text-anchor": "middle", fill: "#f5d547", "font-size": 12, "font-family": "Charter, Georgia, serif" });
      t.textContent = a.commit.subject.length > 24 ? a.commit.subject.slice(0, 23) + "…" : a.commit.subject;
      g.append(t);
      g.append(el("circle", { cx, cy: mid, r: r + 4, fill: "none", stroke: "#f5d547", "stroke-width": 2 }));
    }
    const fill = k === "silent" ? "#b3263e" : k === "announced" ? "#2f7d8c" : "#4a5668";
    const bead = el("circle", { cx, cy: mid, r: reduced ? r : 0, fill, class: "bead", tabindex: 0, role: "link",
      "aria-label": `${a.commit.subject}, score ${a.score}` });
    const tip = el("title"); tip.textContent = `${a.commit.subject} (${day(a.commit.when)}), score ${a.score}`;
    bead.append(tip);
    const open = () => { location.hash = "#/commit/" + a.commit.hash; };
    bead.addEventListener("click", open);
    bead.addEventListener("keydown", (e) => { if (e.key === "Enter") open(); });
    g.append(bead);
    svg.append(g);
    if (!reduced) setTimeout(() => bead.setAttribute("r", r), 30 * i);
  });
}

function renderSuspects(flagged) {
  const list = $("suspects");
  const top = flagged.slice(0, 5);
  list.innerHTML = top.length ? top.map((a) => `
    <li><a class="suspect" href="#/commit/${a.commit.hash}">
      <span class="score-chip ${a.band}">${a.score}</span>
      <span><span class="s">${esc(a.commit.subject)}</span><span class="tag ${kind(a)}">${kind(a)}</span><br>
      <span class="m">${esc(a.commit.author)} on ${day(a.commit.when)}, ${esc(a.hits.map((h) => h.signal.label).slice(0, 2).join(" and "))}</span></span>
    </a></li>`).join("") : `<li class="empty">No commit carries enough evidence to flag.</li>`;
}

function drawDonut() {
  const svg = $("donut");
  svg.replaceChildren();
  const bands = ["likely", "possible", "unlikely"];
  const total = bands.reduce((n, b) => n + (report.counts[b] || 0), 0) || 1;
  const R = 46, C = 2 * Math.PI * R;
  let offset = 0;
  svg.append(el("circle", { cx: 60, cy: 60, r: R, fill: "none", stroke: "#e8ecf1", "stroke-width": 16 }));
  for (const b of bands) {
    const n = report.counts[b] || 0;
    if (!n) continue;
    const len = (n / total) * C;
    svg.append(el("circle", { cx: 60, cy: 60, r: R, fill: "none", stroke: COLORS[b], "stroke-width": 16,
      "stroke-dasharray": `${len} ${C - len}`, "stroke-dashoffset": -offset, transform: "rotate(-90 60 60)" }));
    offset += len;
  }
  const t = el("text", { x: 60, y: 58, "text-anchor": "middle", "font-size": 26, "font-weight": 700, "font-family": "Charter, Georgia, serif", fill: "#1b2433" });
  t.textContent = (report.counts.likely || 0) + (report.counts.possible || 0);
  const s = el("text", { x: 60, y: 76, "text-anchor": "middle", "font-size": 11, fill: "#5b6678" });
  s.textContent = "flagged";
  svg.append(t, s);
  svg.setAttribute("aria-label", `${t.textContent} of ${total} commits flagged`);
  $("donut-legend").innerHTML = bands.map((b) => `<li><i style="background:${COLORS[b]}"></i>${report.counts[b] || 0} ${b}</li>`).join("");
}

function renderHotBars(ul, spots) {
  const max = Math.max(1, ...spots.map((h) => h.flagged));
  ul.innerHTML = spots.length ? spots.map((h) => `
    <li><div class="p"><span>${esc(h.path)}</span><span>${h.flagged}</span></div>
    <div class="track" title="${h.silent} silent of ${h.flagged} flagged">
      <i class="fill-s" style="width:${(h.silent / max) * 100}%"></i><i class="fill-o" style="width:${((h.flagged - h.silent) / max) * 100}%"></i>
    </div></li>`).join("") : `<li class="empty">No file has been touched by a flagged commit.</li>`;
}

/* placeholders filled in by later views */
let filter = { band: "flagged", q: "" };

function renderCommits() {
  const q = filter.q.trim().toLowerCase();
  const rows = report.assessments.filter((a) => {
    if (filter.band === "flagged" && a.band === "unlikely") return false;
    if (filter.band === "silent" && !a.silent) return false;
    if (!q) return true;
    return [a.commit.subject, a.commit.author, a.commit.hash].some((v) => v.toLowerCase().includes(q));
  });
  $("rows").innerHTML = rows.length ? rows.map((a) => `
    <tr data-hash="${a.commit.hash}" tabindex="0">
      <td><span class="meter ${a.band}">${a.score}<span><b style="width:${a.score}%"></b></span></span></td>
      <td><span class="subj">${esc(a.commit.subject)}</span>${kind(a) ? `<span class="tag ${kind(a)}">${kind(a)}</span>` : ""}
        <div class="hashcell">${a.commit.hash.slice(0, 10)}</div></td>
      <td><div class="ev">${a.hits.map((h) => `<span>${esc(h.signal.label)}</span>`).join("") || '<span>none</span>'}</div></td>
      <td>${esc(a.commit.author)}</td>
      <td>${day(a.commit.when)}</td>
    </tr>`).join("") : `<tr><td colspan="5" class="empty">${filter.q ? "No commit matches that search." : "No commit fits this filter. Try All."}</td></tr>`;
}
async function renderCommit(hash) {
  $("h-commit").textContent = "Opening commit";
  $("c-hash").textContent = hash || "";
  $("c-meta").textContent = "";
  $("evidence").innerHTML = "";
  $("diffs").innerHTML = "";
  $("dial").replaceChildren();
  let detail;
  try {
    detail = await api("/api/v1/commits/" + encodeURIComponent(hash || ""));
  } catch (err) {
    $("h-commit").textContent = "This commit is not in the repository";
    $("c-meta").textContent = err.message + ". Pick one from the commit list.";
    return;
  }
  const a = detail.assessment;
  $("c-hash").textContent = a.commit.hash;
  $("h-commit").textContent = a.commit.subject;
  const k = kind(a);
  $("c-meta").textContent = `${a.commit.author} committed this on ${day(a.commit.when)}, touching ` +
    `${plural(a.files, "file", "files")} and ${plural(a.churn, "line", "lines")}.` +
    (k === "silent" ? " It fixes something and never says so." : k === "announced" ? " The fix was announced." : "");
  drawDial(a);
  $("evidence").innerHTML = a.hits.length ? a.hits.map((h) => `
    <li><span class="w">+${h.signal.weight}</span><span class="fam">${esc(h.signal.family)} signal</span>
      <span><span class="what">${esc(h.signal.label)}</span><code>${esc(h.evidence)}</code></span></li>`).join("") +
    (a.targeted ? `<li><span class="w">+8</span><span class="fam">size</span><span><span class="what">Small, targeted change</span><code>${a.churn} lines</code></span></li>` : "")
    : `<li class="empty">No security signals in this commit.</li>`;
  renderDiffs(detail.files, a.hits);
}

function drawDial(a) {
  const svg = $("dial");
  const R = 56, C = 2 * Math.PI * R, arc = C * 0.75;
  svg.append(el("circle", { cx: 70, cy: 70, r: R, fill: "none", stroke: "#dfe5ec", "stroke-width": 12,
    "stroke-dasharray": `${arc} ${C}`, transform: "rotate(135 70 70)", "stroke-linecap": "round" }));
  const fill = el("circle", { cx: 70, cy: 70, r: R, fill: "none", stroke: COLORS[a.band], "stroke-width": 12,
    "stroke-dasharray": `${reduced ? arc * a.score / 100 : 0} ${C}`, transform: "rotate(135 70 70)", "stroke-linecap": "round" });
  fill.style.transition = "stroke-dasharray .7s ease-out";
  svg.append(fill);
  if (!reduced) requestAnimationFrame(() => requestAnimationFrame(() => fill.setAttribute("stroke-dasharray", `${arc * a.score / 100} ${C}`)));
  const n = el("text", { x: 70, y: 76, "text-anchor": "middle", "font-size": 36, "font-weight": 700, "font-family": "Charter, Georgia, serif", fill: "#1b2433" });
  n.textContent = a.score;
  const b = el("text", { x: 70, y: 98, "text-anchor": "middle", "font-size": 13, fill: "#5b6678" });
  b.textContent = a.band;
  svg.append(n, b);
  svg.setAttribute("aria-label", `Score ${a.score} of 100, ${a.band}`);
}

function renderDiffs(files, hits) {
  const evidence = hits.filter((h) => h.signal.family === "diff")
    .flatMap((h) => h.evidence.split("  →  ").map((e) => ({ text: e.trim(), file: h.file })));
  $("diffs").innerHTML = files.map((f) => {
    const marks = evidence.filter((e) => e.file === f.path).map((e) => e.text);
    const line = (cls, text) => {
      const t = text.trim();
      const hit = t && marks.includes(t);
      const body = hit ? esc(text.slice(0, text.indexOf(t))) + "<mark>" + esc(t) + "</mark>" : esc(text);
      return `<div class="${cls}">${body || " "}</div>`;
    };
    const note = f.new ? "created in this commit" : `${f.added.length} added, ${f.removed.length} removed`;
    return `<article class="file"><h3>${esc(f.path)}<small>${note}</small></h3><div class="diff">` +
      f.removed.map((l) => line("del", l)).join("") + f.added.map((l) => line("add", l)).join("") +
      `</div></article>`;
  }).join("");
}
function weekKey(iso) {
  const d = new Date(iso);
  const t = Date.UTC(d.getUTCFullYear(), d.getUTCMonth(), d.getUTCDate());
  const offset = (new Date(t).getUTCDay() + 6) % 7;
  return new Date(t - offset * 864e5).toISOString().slice(0, 10);
}

function renderTimeline() {
  const svg = $("weeks");
  svg.replaceChildren();
  const weeks = report.timeline;
  const W = svg.clientWidth || 900, H = 300, left = 34, bottom = 30, top = 22;
  svg.setAttribute("viewBox", `0 0 ${W} ${H}`);
  if (!weeks.length) return;
  const max = Math.max(1, ...weeks.map((w) => w.commits));
  const step = (W - left - 10) / weeks.length;
  const bw = Math.max(4, Math.min(28, step * 0.62));
  const y = (n) => H - bottom - (n / max) * (H - bottom - top);
  for (let g = 0; g <= max; g += Math.max(1, Math.ceil(max / 4))) {
    svg.append(el("line", { x1: left, x2: W - 6, y1: y(g), y2: y(g), stroke: "#dfe5ec" }));
    const t = el("text", { x: left - 8, y: y(g) + 4, "text-anchor": "end", "font-size": 11, fill: "#5b6678" });
    t.textContent = g; svg.append(t);
  }
  let lastMonth = "";
  weeks.forEach((w, i) => {
    const cx = left + step * i + step / 2, x0 = cx - bw / 2;
    const other = w.commits - w.likely - w.possible;
    let base = 0;
    for (const [n, color] of [[w.likely, COLORS.likely], [w.possible, COLORS.possible], [other, "#b8c2cf"]]) {
      if (!n) continue;
      svg.append(el("rect", { x: x0, y: y(base + n), width: bw, height: y(base) - y(base + n), fill: color, rx: 3 }));
      base += n;
    }
    const tip = el("title");
    tip.textContent = `Week of ${day(w.start)}: ${plural(w.commits, "commit", "commits")}, ${w.silent} silent`;
    const hit = el("rect", { x: cx - step / 2, y: top, width: step, height: H - bottom - top, fill: "transparent" });
    hit.append(tip); svg.append(hit);
    if (w.silent) svg.append(el("circle", { cx, cy: y(w.commits) - 10, r: 4.5, fill: "#b3263e", stroke: "#f5d547", "stroke-width": 2 }));
    const month = new Date(w.start).toLocaleDateString(undefined, { month: "short", timeZone: "UTC" });
    if (month !== lastMonth) {
      const t = el("text", { x: cx, y: H - 10, "text-anchor": "middle", "font-size": 11, fill: "#5b6678" });
      t.textContent = month; svg.append(t); lastMonth = month;
    }
  });
  const silentWeeks = weeks.filter((w) => w.silent);
  $("week-list").innerHTML = silentWeeks.length ? silentWeeks.map((w) => {
    const key = w.start.slice(0, 10);
    const commits = report.assessments.filter((a) => a.silent && weekKey(a.commit.when) === key);
    return `<li><div class="d">Week of ${day(w.start)}</div>` +
      commits.map((a) => `<a href="#/commit/${a.commit.hash}">${esc(a.commit.subject)}</a>`).join("") + `</li>`;
  }).join("") : `<li class="empty">No week contains a silent fix.</li>`;
}

function renderHotspots() {
  renderHotBars($("hot"), report.hotspots);
  $("authors").innerHTML = report.authors.map((au) => `
    <tr><td>${esc(au.name)}</td><td>${au.commits}</td><td>${au.flagged ? `<span class="tag silent">${au.flagged}</span>` : "0"}</td></tr>`).join("");
}

let catalog = null;
async function renderSignals() {
  if (!catalog) {
    try { catalog = await api("/api/v1/signals"); } catch (err) { $("families").innerHTML = `<p class="empty">${esc(err.message)}</p>`; return; }
  }
  const names = { message: "What the message says", diff: "What the diff adds", path: "Which files it touches" };
  $("families").innerHTML = Object.keys(names).map((fam) => `
    <section class="panel"><h2>${names[fam]}</h2>` +
    catalog.filter((s) => s.family === fam).map((s) => `
      <div class="sig${s.announced ? " ann" : ""}"><div class="top"><span>${esc(s.label)}</span><b>+${s.weight}</b></div>
      <p>${esc(s.description)}${s.announced ? ". Marks the fix as announced." : ""}</p></div>`).join("") +
    `</section>`).join("");
}

async function init() {
  $("rescan").addEventListener("click", () => load(true));
  $("q").addEventListener("input", (e) => { filter.q = e.target.value; if (report) renderCommits(); });
  document.querySelectorAll(".seg button").forEach((b) => b.addEventListener("click", () => {
    filter.band = b.dataset.band;
    document.querySelectorAll(".seg button").forEach((x) => x.classList.toggle("on", x === b));
    if (report) renderCommits();
  }));
  const openRow = (e) => {
    const tr = e.target.closest("tr[data-hash]");
    if (tr && (e.type === "click" || e.key === "Enter")) location.hash = "#/commit/" + tr.dataset.hash;
  };
  $("rows").addEventListener("click", openRow);
  $("rows").addEventListener("keydown", openRow);
  window.addEventListener("hashchange", route);
  window.addEventListener("resize", () => {
    if (!report) return;
    if (!$("page-overview").hidden) drawLineup();
    if (!$("page-timeline").hidden) renderTimeline();
  });
  try {
    const st = await api("/api/v1/status");
    $("build").textContent = `Culprit ${st.version} on ${st.platform}`;
  } catch { /* the report call surfaces errors */ }
  load(false);
}

init();
