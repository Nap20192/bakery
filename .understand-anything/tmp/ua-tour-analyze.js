#!/usr/bin/env node
"use strict";
const fs = require("fs");

function main() {
  const inPath = process.argv[2];
  const outPath = process.argv[3];
  if (!inPath || !outPath) {
    console.error("usage: analyze.js <input.json> <output.json>");
    process.exit(1);
  }
  const data = JSON.parse(fs.readFileSync(inPath, "utf8"));
  const nodes = data.nodes || [];
  const edges = data.edges || [];
  const layers = data.layers || [];

  const byId = new Map();
  for (const n of nodes) byId.set(n.id, n);

  // Fan-in / fan-out
  const fanIn = new Map();
  const fanOut = new Map();
  for (const n of nodes) { fanIn.set(n.id, 0); fanOut.set(n.id, 0); }
  for (const e of edges) {
    if (fanOut.has(e.source)) fanOut.set(e.source, fanOut.get(e.source) + 1);
    if (fanIn.has(e.target)) fanIn.set(e.target, fanIn.get(e.target) + 1);
  }

  const nm = (id) => (byId.get(id) || {}).name || id;
  const sm = (id) => (byId.get(id) || {}).summary || "";

  const fanInRanking = [...fanIn.entries()]
    .map(([id, v]) => ({ id, fanIn: v, name: nm(id) }))
    .sort((a, b) => b.fanIn - a.fanIn).slice(0, 20);
  const fanOutRanking = [...fanOut.entries()]
    .map(([id, v]) => ({ id, fanOut: v, name: nm(id) }))
    .sort((a, b) => b.fanOut - a.fanOut).slice(0, 20);

  // percentiles for entry scoring
  const foVals = [...fanOut.values()].sort((a, b) => a - b);
  const fiVals = [...fanIn.values()].sort((a, b) => a - b);
  const pct = (arr, p) => arr.length ? arr[Math.min(arr.length - 1, Math.floor(p * arr.length))] : 0;
  const foTop10 = pct(foVals, 0.9);
  const fiBottom25 = pct(fiVals, 0.25);

  const codeEntryNames = new Set([
    "index.ts","index.js","main.ts","main.js","app.ts","app.js","server.ts","server.js",
    "mod.rs","main.go","main.py","main.rs","manage.py","app.py","wsgi.py","asgi.py","run.py",
    "__main__.py","Application.java","Main.java","Program.cs","config.ru","index.php",
    "App.swift","Application.kt","main.cpp","main.c"
  ]);

  const entryScores = [];
  for (const n of nodes) {
    let score = 0;
    const fp = n.filePath || "";
    const depth = fp ? fp.split("/").length : 99;
    if (n.type === "document") {
      if (n.name === "README.md" && depth === 1) score += 5;
      else if (/\.md$/.test(n.name || "") && depth === 1) score += 2;
    } else if (n.type === "file") {
      if (codeEntryNames.has(n.name)) score += 3;
      if (depth <= 2) score += 1;
      if (fanOut.get(n.id) >= foTop10 && foTop10 > 0) score += 1;
      if (fanIn.get(n.id) <= fiBottom25) score += 1;
    }
    if (score > 0) entryScores.push({ id: n.id, score, name: n.name, summary: sm(n.id), type: n.type });
  }
  entryScores.sort((a, b) => b.score - a.score);
  const entryPointCandidates = entryScores.slice(0, 5);

  // BFS from top code entry point
  const codeEntries = entryScores.filter(e => e.type === "file");
  const startNode = codeEntries.length ? codeEntries[0].id : (nodes[0] && nodes[0].id);
  const adj = new Map();
  for (const n of nodes) adj.set(n.id, []);
  for (const e of edges) {
    if ((e.type === "imports" || e.type === "calls") && adj.has(e.source)) {
      adj.get(e.source).push(e.target);
    }
  }
  const depthMap = {};
  const order = [];
  const q = [startNode];
  depthMap[startNode] = 0;
  while (q.length) {
    const cur = q.shift();
    order.push(cur);
    for (const next of (adj.get(cur) || [])) {
      if (!(next in depthMap)) {
        depthMap[next] = depthMap[cur] + 1;
        q.push(next);
      }
    }
  }
  const byDepth = {};
  for (const id of order) {
    const d = depthMap[id];
    (byDepth[d] = byDepth[d] || []).push(id);
  }

  // Non-code inventory
  const nonCodeFiles = { documentation: [], infrastructure: [], data: [], config: [] };
  for (const n of nodes) {
    const rec = { id: n.id, name: n.name, type: n.type, summary: sm(n.id) };
    if (n.type === "document") nonCodeFiles.documentation.push(rec);
    else if (["service","pipeline","resource"].includes(n.type)) nonCodeFiles.infrastructure.push(rec);
    else if (["table","schema","endpoint"].includes(n.type)) nonCodeFiles.data.push(rec);
    else if (n.type === "config") nonCodeFiles.config.push(rec);
  }

  // Clusters: bidirectional edges + expansion
  const pairKey = (a, b) => [a, b].sort().join("||");
  const dirSet = new Set();
  for (const e of edges) {
    if (e.type === "imports" || e.type === "calls") dirSet.add(e.source + ">>" + e.target);
  }
  const clusterSeeds = [];
  const seen = new Set();
  for (const e of edges) {
    if (e.type !== "imports" && e.type !== "calls") continue;
    if (dirSet.has(e.target + ">>" + e.source)) {
      const k = pairKey(e.source, e.target);
      if (!seen.has(k)) { seen.add(k); clusterSeeds.push(new Set([e.source, e.target])); }
    }
  }
  // adjacency (undirected) for expansion
  const undAdj = new Map();
  for (const n of nodes) undAdj.set(n.id, new Set());
  for (const e of edges) {
    if (e.type === "imports" || e.type === "calls") {
      undAdj.get(e.source).add(e.target);
      undAdj.get(e.target).add(e.source);
    }
  }
  for (const cl of clusterSeeds) {
    let changed = true;
    while (changed && cl.size < 5) {
      changed = false;
      const candCount = new Map();
      for (const m of cl) {
        for (const nb of undAdj.get(m)) {
          if (!cl.has(nb)) candCount.set(nb, (candCount.get(nb) || 0) + 1);
        }
      }
      for (const [c, cnt] of candCount) {
        if (cnt >= 2 && cl.size < 5) { cl.add(c); changed = true; break; }
      }
    }
  }
  // dedupe clusters by member set, count internal edges
  const clusterUniq = [];
  const ckeys = new Set();
  for (const cl of clusterSeeds) {
    const arr = [...cl].sort();
    const key = arr.join("||");
    if (ckeys.has(key)) continue;
    ckeys.add(key);
    let ec = 0;
    for (const e of edges) {
      if (cl.has(e.source) && cl.has(e.target) && (e.type === "imports" || e.type === "calls")) ec++;
    }
    clusterUniq.push({ nodes: arr, edgeCount: ec });
  }
  clusterUniq.sort((a, b) => b.edgeCount - a.edgeCount || b.nodes.length - a.nodes.length);
  const clusters = clusterUniq.slice(0, 10);

  const nodeSummaryIndex = {};
  for (const n of nodes) nodeSummaryIndex[n.id] = { name: n.name, type: n.type, summary: n.summary || "" };

  const out = {
    scriptCompleted: true,
    entryPointCandidates,
    fanInRanking,
    fanOutRanking,
    bfsTraversal: { startNode, order, depthMap, byDepth },
    nonCodeFiles,
    clusters,
    layers: { count: layers.length, list: layers.map(l => ({ id: l.id, name: l.name, description: l.description })) },
    nodeSummaryIndex,
    totalNodes: nodes.length,
    totalEdges: edges.length
  };
  fs.writeFileSync(outPath, JSON.stringify(out, null, 2));
  console.log("done. start=" + startNode + " bfsReached=" + order.length);
}

try { main(); } catch (err) { console.error(err.stack || String(err)); process.exit(1); }
