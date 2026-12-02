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
function renderCommit() {}
function renderTimeline() {}
function renderHotspots() {}
function renderSignals() {}

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
  window.addEventListener("resize", () => { if (report && !$("page-overview").hidden) drawLineup(); });
  try {
    const st = await api("/api/v1/status");
    $("build").textContent = `Culprit ${st.version} on ${st.platform}`;
  } catch { /* the report call surfaces errors */ }
  load(false);
}

init();
