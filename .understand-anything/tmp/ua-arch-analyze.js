#!/usr/bin/env node
'use strict';

const fs = require('fs');

function fail(msg) { process.stderr.write(String(msg) + '\n'); process.exit(1); }

const inPath = process.argv[2];
const outPath = process.argv[3];
if (!inPath || !outPath) fail('usage: analyze.js <input.json> <output.json>');

let data;
try { data = JSON.parse(fs.readFileSync(inPath, 'utf8')); }
catch (e) { fail('cannot read/parse input: ' + e.message); }

const fileNodes = data.fileNodes || [];
const importEdges = data.importEdges || [];
const allEdges = data.allEdges || [];

// Derive filePath for nodes missing it (e.g. compose service nodes id like service:docker-compose.yml:db)
function pathOf(n) {
  if (n.filePath) return n.filePath;
  // id form type:path[:sub]
  const id = n.id || '';
  const rest = id.indexOf(':') >= 0 ? id.slice(id.indexOf(':') + 1) : id;
  return rest; // best effort
}

const idToNode = new Map();
fileNodes.forEach(n => idToNode.set(n.id, n));
const idToPath = new Map();
fileNodes.forEach(n => idToPath.set(n.id, pathOf(n)));

// ---- Common prefix ----
const paths = fileNodes.map(n => idToPath.get(n.id)).filter(Boolean);
function commonPrefixDir(paths) {
  if (!paths.length) return '';
  const split = paths.map(p => p.split('/'));
  const first = split[0];
  let prefix = [];
  for (let i = 0; i < first.length - 1; i++) {
    const seg = first[i];
    if (split.every(s => s.length > i + 1 && s[i] === seg)) prefix.push(seg);
    else break;
  }
  return prefix.length ? prefix.join('/') + '/' : '';
}
const prefix = commonPrefixDir(paths);

function groupKey(p) {
  let rel = p;
  if (prefix && rel.startsWith(prefix)) rel = rel.slice(prefix.length);
  const parts = rel.split('/');
  if (parts.length <= 1) return '(root)';
  return parts[0];
}

// ---- A. Directory grouping ----
const directoryGroups = {};
fileNodes.forEach(n => {
  const g = groupKey(idToPath.get(n.id));
  (directoryGroups[g] = directoryGroups[g] || []).push(n.id);
});

// ---- B. Node type grouping ----
const nodeTypeGroups = {};
fileNodes.forEach(n => {
  (nodeTypeGroups[n.type] = nodeTypeGroups[n.type] || []).push(n.id);
});

// ---- C. Adjacency / fan-in fan-out ----
const fileFanOut = {}, fileFanIn = {};
fileNodes.forEach(n => { fileFanOut[n.id] = 0; fileFanIn[n.id] = 0; });
importEdges.forEach(e => {
  if (fileFanOut[e.source] === undefined) fileFanOut[e.source] = 0;
  if (fileFanIn[e.target] === undefined) fileFanIn[e.target] = 0;
  fileFanOut[e.source]++; fileFanIn[e.target]++;
});

// ---- D. Cross-category edges (by node type) ----
const ccMap = {};
allEdges.forEach(e => {
  const s = idToNode.get(e.source), t = idToNode.get(e.target);
  if (!s || !t) return;
  if (s.type === t.type && s.type === 'file') return; // skip plain file->file noise for this view
  const key = s.type + '|' + t.type + '|' + e.type;
  ccMap[key] = (ccMap[key] || 0) + 1;
});
const crossCategoryEdges = Object.entries(ccMap).map(([k, count]) => {
  const [fromType, toType, edgeType] = k.split('|');
  return { fromType, toType, edgeType, count };
}).sort((a, b) => b.count - a.count);

// ---- E. Inter-group import frequency ----
const groupOf = {};
fileNodes.forEach(n => { groupOf[n.id] = groupKey(idToPath.get(n.id)); });
const interMap = {};
importEdges.forEach(e => {
  const a = groupOf[e.source], b = groupOf[e.target];
  if (a === undefined || b === undefined) return;
  if (a === b) return;
  const k = a + '|' + b;
  interMap[k] = (interMap[k] || 0) + 1;
});
const interGroupImports = Object.entries(interMap).map(([k, count]) => {
  const [from, to] = k.split('|');
  return { from, to, count };
}).sort((a, b) => b.count - a.count);

// ---- F. Intra-group density ----
const intraGroupDensity = {};
Object.keys(directoryGroups).forEach(g => { intraGroupDensity[g] = { internalEdges: 0, totalEdges: 0, density: 0 }; });
importEdges.forEach(e => {
  const a = groupOf[e.source], b = groupOf[e.target];
  if (a !== undefined) { intraGroupDensity[a].totalEdges++; if (a === b) intraGroupDensity[a].internalEdges++; }
  if (b !== undefined && b !== a) { intraGroupDensity[b].totalEdges++; }
});
Object.keys(intraGroupDensity).forEach(g => {
  const d = intraGroupDensity[g];
  d.density = d.totalEdges ? +(d.internalEdges / d.totalEdges).toFixed(3) : 0;
});

// ---- G. Pattern matching ----
const dirPatterns = [
  [['routes','api','controllers','endpoints','handlers','controller','routers','serializers','blueprints'], 'api'],
  [['services','core','lib','domain','logic','signals','composables','mailers','jobs','channels','internal'], 'service'],
  [['models','db','data','persistence','repository','repo','entities','entity','migrations','sql','database'], 'data'],
  [['components','views','pages','ui','layouts','screens'], 'ui'],
  [['middleware','plugins','interceptors','guards'], 'middleware'],
  [['utils','helpers','common','shared','tools','pkg','templatetags'], 'utility'],
  [['config','constants','env','settings','management','commands'], 'config'],
  [['__tests__','test','tests','spec','specs'], 'test'],
  [['types','interfaces','schemas','contracts','dtos','dto','request','response'], 'types'],
  [['hooks'], 'hooks'],
  [['store','state','reducers','actions','slices'], 'state'],
  [['assets','static','public'], 'assets'],
  [['cmd','bin'], 'entry'],
  [['docs','documentation','wiki'], 'documentation'],
  [['deploy','deployment','infra','infrastructure','docker','k8s','kubernetes','helm','charts','terraform','tf'], 'infrastructure'],
  [['.github','.gitlab','.circleci'], 'ci-cd'],
];
function classifyDir(name) {
  const n = name.toLowerCase();
  for (const [keys, label] of dirPatterns) if (keys.includes(n)) return label;
  return null;
}
const patternMatches = {};
Object.keys(directoryGroups).forEach(g => {
  const m = classifyDir(g);
  if (m) patternMatches[g] = m;
});

// file-level patterns
function fileLabel(p, name) {
  const lp = p.toLowerCase();
  if (/(\.test\.|\.spec\.|_test\.go$|_test\.|test_.*\.py$|test\.java$|_spec\.rb$)/.test(lp)) return 'test';
  if (lp.endsWith('.d.ts')) return 'types';
  if (lp.endsWith('.sql')) return 'data';
  if (/\.(graphql|gql|proto)$/.test(lp)) return 'types';
  if (/\.(md|rst)$/.test(lp)) return 'documentation';
  if (/dockerfile/.test(lp) || /docker-compose/.test(lp) || lp.endsWith('.tf') || lp.endsWith('.tfvars') || /^makefile$/.test(name.toLowerCase())) return 'infrastructure';
  return null;
}

// ---- H. Deployment topology ----
const infraFiles = [];
let hasDockerfile = false, hasCompose = false, hasK8s = false, hasTerraform = false, hasCI = false;
fileNodes.forEach(n => {
  const p = idToPath.get(n.id); const lp = p.toLowerCase(); const name = (n.name || '').toLowerCase();
  if (/dockerfile/.test(lp)) { hasDockerfile = true; if (!infraFiles.includes(p)) infraFiles.push(p); }
  if (/docker-compose/.test(lp)) { hasCompose = true; if (!infraFiles.includes(p)) infraFiles.push(p); }
  if (/(k8s|kubernetes|helm|charts)/.test(lp)) { hasK8s = true; if (!infraFiles.includes(p)) infraFiles.push(p); }
  if (lp.endsWith('.tf') || lp.endsWith('.tfvars')) { hasTerraform = true; if (!infraFiles.includes(p)) infraFiles.push(p); }
  if (/\.github\/workflows|\.gitlab-ci|jenkinsfile/.test(lp)) { hasCI = true; if (!infraFiles.includes(p)) infraFiles.push(p); }
  if (n.type === 'pipeline') { hasCI = true; if (!infraFiles.includes(p)) infraFiles.push(p); }
});
const deploymentTopology = { hasDockerfile, hasCompose, hasK8s, hasTerraform, hasCI, infraFiles };

// ---- I. Data pipeline ----
const dataPipeline = { schemaFiles: [], migrationFiles: [], dataModelFiles: [], apiHandlerFiles: [] };
fileNodes.forEach(n => {
  const p = idToPath.get(n.id); const lp = p.toLowerCase();
  if (/\.(sql)$/.test(lp) && /migrat/.test(lp)) dataPipeline.migrationFiles.push(p);
  else if (/\.(sql|graphql|gql|proto|prisma)$/.test(lp)) dataPipeline.schemaFiles.push(p);
  if (n.type === 'table' || n.type === 'schema') dataPipeline.schemaFiles.push(p);
  if (/\/(models|domain|entity|entities)\//.test(lp)) dataPipeline.dataModelFiles.push(p);
  if (/\/(routes|api|handlers|http|controller)\b/.test(lp)) dataPipeline.apiHandlerFiles.push(p);
});

// ---- J. Documentation coverage ----
const groupsWithDocs = new Set();
fileNodes.forEach(n => {
  if (n.type === 'document' || /\.(md|rst)$/.test(idToPath.get(n.id).toLowerCase())) {
    groupsWithDocs.add(groupOf[n.id]);
  }
});
const totalGroups = Object.keys(directoryGroups).length;
const undocumentedGroups = Object.keys(directoryGroups).filter(g => !groupsWithDocs.has(g));
const docCoverage = {
  groupsWithDocs: groupsWithDocs.size,
  totalGroups,
  coverageRatio: totalGroups ? +(groupsWithDocs.size / totalGroups).toFixed(2) : 0,
  undocumentedGroups,
};

// ---- K. Dependency direction ----
const pairDir = {};
interGroupImports.forEach(({ from, to, count }) => {
  const key = [from, to].sort().join('||');
  pairDir[key] = pairDir[key] || {};
  pairDir[key][from + '>' + to] = count;
});
const dependencyDirection = [];
Object.keys(pairDir).forEach(key => {
  const [g1, g2] = key.split('||');
  const f = pairDir[key][g1 + '>' + g2] || 0;
  const r = pairDir[key][g2 + '>' + g1] || 0;
  if (f === r) return;
  if (f > r) dependencyDirection.push({ dependent: g1, dependsOn: g2 });
  else dependencyDirection.push({ dependent: g2, dependsOn: g1 });
});

// ---- file stats ----
const filesPerGroup = {};
Object.keys(directoryGroups).forEach(g => filesPerGroup[g] = directoryGroups[g].length);
const nodeTypeCounts = {};
Object.keys(nodeTypeGroups).forEach(t => nodeTypeCounts[t] = nodeTypeGroups[t].length);

// add file-level pattern labels into a separate map for assignment help
const fileLevelPatterns = {};
fileNodes.forEach(n => {
  const lbl = fileLabel(idToPath.get(n.id), n.name || '');
  if (lbl) fileLevelPatterns[n.id] = lbl;
});

const result = {
  scriptCompleted: true,
  commonPrefix: prefix,
  directoryGroups,
  nodeTypeGroups,
  crossCategoryEdges,
  interGroupImports,
  intraGroupDensity,
  patternMatches,
  fileLevelPatterns,
  deploymentTopology,
  dataPipeline,
  docCoverage,
  dependencyDirection,
  fileStats: { totalFileNodes: fileNodes.length, filesPerGroup, nodeTypeCounts },
  fileFanIn,
  fileFanOut,
};

try { fs.writeFileSync(outPath, JSON.stringify(result, null, 2)); }
catch (e) { fail('cannot write output: ' + e.message); }
process.exit(0);
