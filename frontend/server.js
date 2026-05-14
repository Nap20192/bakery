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
const logsEnabled = process.env.FRONTEND_LOGS !== 'false';

const contentTypes = {
  '.css': 'text/css; charset=utf-8',
  '.html': 'text/html; charset=utf-8',
  '.js': 'text/javascript; charset=utf-8',
  '.json': 'application/json; charset=utf-8',
  '.svg': 'image/svg+xml',
};

createServer(async (req, res) => {
  const started = Date.now();
  const requestID = `${started.toString(36)}-${Math.random().toString(36).slice(2, 8)}`;
  res.on('finish', () => {
    log('info', 'frontend request', {
      request_id: requestID,
      method: req.method,
      path: req.url,
      status: res.statusCode,
      duration_ms: Date.now() - started,
    });
  });

  if (req.url?.startsWith('/api/')) {
    proxyAPI(req, res, requestID);
    return;
  }

  await serveStatic(req, res, requestID);
}).listen(port, host, () => {
  log('info', 'frontend started', {
    host,
    port,
    node_env: process.env.NODE_ENV || '',
    frontend_logs: logsEnabled,
    env: {
      HOST: process.env.HOST || '',
      PORT: process.env.PORT || '',
      BACKEND_URL: process.env.BACKEND_URL || '',
      FRONTEND_LOGS: process.env.FRONTEND_LOGS || '',
    },
    backend_url: backendURL.origin,
    backend_protocol: backendURL.protocol,
    backend_hostname: backendURL.hostname,
    backend_port: backendURL.port,
    root,
    dist_exists: existsSync(root),
  });
});

function proxyAPI(req, res, requestID) {
  const upstreamPath = req.url.replace(/^\/api/, '') || '/';
  const upstreamURL = new URL(upstreamPath, backendURL);
  const transport = upstreamURL.protocol === 'https:' ? httpsRequest : httpRequest;
  const started = Date.now();

  log('info', 'api proxy request', {
    request_id: requestID,
    method: req.method,
    path: req.url,
    request_headers: requestLogHeaders(req.headers),
    backend_url: backendURL.origin,
    upstream_url: sanitizeURL(upstreamURL),
    upstream_path: upstreamURL.pathname,
    upstream_protocol: upstreamURL.protocol,
    upstream_hostname: upstreamURL.hostname,
    upstream_port: upstreamURL.port,
  });

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
      log('info', 'api proxy response', {
        request_id: requestID,
        method: req.method,
        path: req.url,
        upstream_url: sanitizeURL(upstreamURL),
        upstream_status: proxyRes.statusCode || 0,
        upstream_headers: responseLogHeaders(proxyRes.headers),
        duration_ms: Date.now() - started,
      });
      res.writeHead(proxyRes.statusCode || 502, proxyRes.headers);
      proxyRes.pipe(res);
    },
  );

  proxyReq.on('error', (err) => {
    log('error', 'api proxy failed', {
      request_id: requestID,
      method: req.method,
      path: req.url,
      backend_url: backendURL.origin,
      upstream_url: sanitizeURL(upstreamURL),
      upstream_protocol: upstreamURL.protocol,
      upstream_hostname: upstreamURL.hostname,
      upstream_port: upstreamURL.port,
      duration_ms: Date.now() - started,
      error: err.message,
      code: err.code,
      errno: err.errno,
      syscall: err.syscall,
      address: err.address,
      port: err.port,
    });
    res.writeHead(502, { 'content-type': 'application/json; charset=utf-8' });
    res.end(JSON.stringify({ error: 'backend unavailable' }));
  });

  req.pipe(proxyReq);
}

async function serveStatic(req, res, requestID) {
  const url = new URL(req.url || '/', 'http://localhost');
  const pathname = normalize(decodeURIComponent(url.pathname)).replace(/^(\.\.[/\\])+/, '');
  let filePath = join(root, pathname);
  let fallbackToIndex = false;

  try {
    const info = await stat(filePath);
    if (info.isDirectory()) {
      filePath = join(filePath, 'index.html');
    }
  } catch {
    filePath = join(root, 'index.html');
    fallbackToIndex = true;
  }

  if (!existsSync(filePath)) {
    log('warn', 'static file not found', {
      request_id: requestID,
      path: req.url,
      file_path: filePath,
    });
    res.writeHead(404);
    res.end('not found');
    return;
  }

  if (fallbackToIndex) {
    log('info', 'static fallback', {
      request_id: requestID,
      path: req.url,
      file_path: filePath,
    });
  }

  res.writeHead(200, {
    'content-type': contentTypes[extname(filePath)] || 'application/octet-stream',
  });
  createReadStream(filePath).pipe(res);
}

function sanitizeURL(value) {
  const url = new URL(value);
  url.username = '';
  url.password = '';
  return url.toString();
}

function requestLogHeaders(headers) {
  return {
    host: headers.host || '',
    accept: headers.accept || '',
    origin: headers.origin || '',
    referer: headers.referer || '',
    'user-agent': headers['user-agent'] || '',
    'x-forwarded-for': headers['x-forwarded-for'] || '',
    'x-forwarded-host': headers['x-forwarded-host'] || '',
    'x-forwarded-proto': headers['x-forwarded-proto'] || '',
  };
}

function responseLogHeaders(headers) {
  return {
    'content-type': headers['content-type'] || '',
    'content-length': headers['content-length'] || '',
    server: headers.server || '',
  };
}

function log(level, message, payload = {}) {
  if (!logsEnabled) return;
  const record = {
    level,
    message,
    at: new Date().toISOString(),
    ...payload,
  };
  const line = JSON.stringify(record);
  if (level === 'error') {
    console.error(line);
    return;
  }
  if (level === 'warn') {
    console.warn(line);
    return;
  }
  console.log(line);
}
