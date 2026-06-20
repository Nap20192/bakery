import fs from "fs";
const raw = JSON.parse(fs.readFileSync(".understand-anything/tmp/ua-batch-3-raw.json","utf8"));

// rebuild combined from the two parts we already wrote (or regenerate). Re-read parts.
const p1 = JSON.parse(fs.readFileSync(".understand-anything/intermediate/batch-3-part-1.json","utf8"));
const p2 = JSON.parse(fs.readFileSync(".understand-anything/intermediate/batch-3-part-2.json","utf8"));
let nodes = [...p1.nodes, ...p2.nodes];
let edges = [...p1.edges, ...p2.edges];

function fileOfEdgeSource(s){
  const idx = s.indexOf(":");
  const rest = s.slice(idx+1);
  if (s.startsWith("file:")) return rest;
  const lastColon = rest.lastIndexOf(":");
  return rest.slice(0,lastColon);
}

// Force 4 parts to balance the deps.go / server.go heavy edge files
const parts = 4;
const files = raw.files.map(f=>f.path).sort();
const chunkSize = Math.ceil(files.length / parts);
const fileGroups = [];
for (let i=0;i<parts;i++) fileGroups.push(new Set(files.slice(i*chunkSize,(i+1)*chunkSize)));

// clean old parts
for (let i=1;i<=10;i++){ try{fs.unlinkSync(`.understand-anything/intermediate/batch-3-part-${i}.json`);}catch(e){} }

let sn=0,se=0;
for (let i=0;i<parts;i++){
  const grp = fileGroups[i];
  const pn = nodes.filter(n=> grp.has(n.filePath));
  const pe = edges.filter(e=> grp.has(fileOfEdgeSource(e.source)));
  sn+=pn.length; se+=pe.length;
  fs.writeFileSync(`.understand-anything/intermediate/batch-3-part-${i+1}.json`, JSON.stringify({nodes:pn,edges:pe},null,2));
  console.log(`part ${i+1}: files=[${[...grp].join(", ")}] nodes=${pn.length} edges=${pe.length}`);
}
console.log("sum nodes:",sn,"sum edges:",se);
console.log("import edges total:", edges.filter(e=>e.type==="imports").length);
