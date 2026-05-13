import { createReadStream, existsSync } from 'node:fs';
import { stat } from 'node:fs/promises';
import { createServer } from 'node:http';
import { request as httpRequest } from 'node:http';
import { request as httpsRequest } from 'node:https';
import { extname, join, normalize } from 'node:path';
import { fileURLToPath } from 'node:url';

const port = Number(process.env.PORT || 8080);
const host = process.env.HOST || '0.0.0.0';
const backendURL = new URL(process.env.BACKEND_URL || 'http://bakery.railway.internal:8080');
const root = join(fileURLToPath(new URL('.', import.meta.url)), 'dist');

const contentTypes = {
  '.css': 'text/css; charset=utf-8',
  '.html': 'text/html; charset=utf-8',
  '.js': 'text/javascript; charset=utf-8',
  '.json': 'application/json; charset=utf-8',
  '.svg': 'image/svg+xml',
};

createServer(async (req, res) => {
  if (req.url?.startsWith('/api/')) {
    proxyAPI(req, res);
    return;
  }

  await serveStatic(req, res);
}).listen(port, host, () => {
  console.log(`frontend listening on ${host}:${port}`);
});

function proxyAPI(req, res) {
  const upstreamPath = req.url.replace(/^\/api/, '') || '/';
  const upstreamURL = new URL(upstreamPath, backendURL);
  const transport = upstreamURL.protocol === 'https:' ? httpsRequest : httpRequest;

  const proxyReq = transport(
    upstreamURL,
    {
      method: req.method,
      headers: {
        ...req.headers,
        host: backendURL.host,
      },
    },
    (proxyRes) => {
      res.writeHead(proxyRes.statusCode || 502, proxyRes.headers);
      proxyRes.pipe(res);
    },
  );

  proxyReq.on('error', (err) => {
    console.error('api proxy failed', err);
    res.writeHead(502, { 'content-type': 'application/json; charset=utf-8' });
    res.end(JSON.stringify({ error: 'backend unavailable' }));
  });

  req.pipe(proxyReq);
}

async function serveStatic(req, res) {
  const url = new URL(req.url || '/', 'http://localhost');
  const pathname = normalize(decodeURIComponent(url.pathname)).replace(/^(\.\.[/\\])+/, '');
  let filePath = join(root, pathname);

  try {
    const info = await stat(filePath);
    if (info.isDirectory()) {
      filePath = join(filePath, 'index.html');
    }
  } catch {
    filePath = join(root, 'index.html');
  }

  if (!existsSync(filePath)) {
    res.writeHead(404);
    res.end('not found');
    return;
  }

  res.writeHead(200, {
    'content-type': contentTypes[extname(filePath)] || 'application/octet-stream',
  });
  createReadStream(filePath).pipe(res);
}
