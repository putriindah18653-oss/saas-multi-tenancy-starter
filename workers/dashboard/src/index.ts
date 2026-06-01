interface Env {
  BACKEND_ORIGIN: string;
  ENV: string;
  INTERNAL_PROXY_SECRET: string;
  CACHE_TENANT: KVNamespace;
}

interface TenantConfig {
  tenant_id: string;
  dashboard_api_url?: string;
  features?: Record<string, boolean>;
}

// In-memory rate limiter (per-isolate). For production multi-isolate deployments,
// use a Durable Object or Workers Rate Limiting API instead.
const rateLimitCache = new Map<string, { count: number; resetAt: number }>();
const RATE_LIMIT_MAX = 300; // requests per window
const RATE_LIMIT_WINDOW_MS = 60_000; // 1 minute

function isRateLimited(ip: string): boolean {
  const now = Date.now();
  const entry = rateLimitCache.get(ip);
  if (!entry || now > entry.resetAt) {
    rateLimitCache.set(ip, { count: 1, resetAt: now + RATE_LIMIT_WINDOW_MS });
    return false;
  }
  entry.count++;
  if (entry.count > RATE_LIMIT_MAX) return true;
  return false;
}

// Periodic cleanup to prevent memory leak.
setInterval(() => {
  const now = Date.now();
  for (const [key, entry] of rateLimitCache) {
    if (now > entry.resetAt) rateLimitCache.delete(key);
  }
}, RATE_LIMIT_WINDOW_MS);

function corsHeaders(origin: string): Headers {
  const headers = new Headers();
  headers.set("Access-Control-Allow-Origin", origin || "*");
  headers.set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS");
  headers.set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Tenant-ID");
  headers.set("Access-Control-Allow-Credentials", "true");
  headers.set("Access-Control-Max-Age", "86400");
  return headers;
}

async function resolveTenant(request: Request, env: Env): Promise<TenantConfig | null> {
  const url = new URL(request.url);
  const host = url.hostname;

  // 1. Resolve by host from KV
  const cached = await env.CACHE_TENANT.get<{ tenant_id: string; dashboard_api_url?: string; features?: Record<string, boolean> }>(`tenant:dashboard:${host}`, { type: "json" }).catch((err: unknown) => {
    console.error('[resolve-tenant] KV read error:', err);
    return null;
  });
  if (cached) return cached;

  // 2. Fallback: try owner-domain tenant resolution
  //    e.g. manage.portalonline.id -> tenant is owner platform context
  if (host.startsWith("manage.")) {
    const baseDomain = host.slice(host.indexOf(".") + 1);
    const ownerCached = await env.CACHE_TENANT.get<{ tenant_id: string }>(`owner:${baseDomain}`, { type: "json" }).catch((err: unknown) => {
      console.error('[resolve-tenant] owner KV read error:', err);
      return null;
    });
    if (ownerCached) {
      return { tenant_id: ownerCached.tenant_id, dashboard_api_url: env.BACKEND_ORIGIN };
    }
  }

  return null;
}

async function proxyToBackend(request: Request, env: Env, tenant: TenantConfig | null): Promise<Response> {
  const url = new URL(request.url);
  const apiBase = tenant?.dashboard_api_url || env.BACKEND_ORIGIN;
  const backendUrl = new URL(request.url);
  backendUrl.hostname = new URL(apiBase).hostname;
  backendUrl.port = new URL(apiBase).port;

  const headers = new Headers(request.headers);

  // Strip client-supplied X-Tenant-ID; backend only trusts X-Internal-Tenant-ID
  headers.delete("X-Tenant-ID");

  // Inject server-resolved tenant context
  if (tenant?.tenant_id) {
    headers.set("X-Internal-Tenant-ID", tenant.tenant_id);
    headers.set("X-Tenant-ID", tenant.tenant_id);
  }

  // Authenticate to backend so it trusts X-Internal-Tenant-ID
  if (env.INTERNAL_PROXY_SECRET) {
    headers.set("X-Internal-Proxy-Secret", env.INTERNAL_PROXY_SECRET);
  }

  // Mark origin as internal proxy
  headers.set("X-Internal-Proxy", "cf-dashboard-worker");

  // Forward client IP for audit
  const cfConnectingIp = request.headers.get("CF-Connecting-IP");
  if (cfConnectingIp) {
    headers.set("X-Forwarded-For", cfConnectingIp);
  }

  // Only clone body for methods with content
  let body: ArrayBuffer | null = null;
  if (request.method !== "GET" && request.method !== "HEAD" && request.method !== "DELETE" && request.method !== "OPTIONS") {
    const contentLength = parseInt(request.headers.get("Content-Length") || "0", 10);
    if (contentLength > 0) {
      body = await request.clone().arrayBuffer();
    }
  }

  const proxyRequest = new Request(backendUrl, {
    method: request.method,
    headers,
    body,
    redirect: "manual",
  });

  return fetch(proxyRequest);
}

function isApiRequest(url: URL): boolean {
  return url.pathname.startsWith("/api/") || url.pathname === "/health" || url.pathname === "/ready";
}

export default {
  async fetch(request: Request, env: Env): Promise<Response> {
    const url = new URL(request.url);

    // CORS preflight
    if (request.method === "OPTIONS") {
      return new Response(null, { status: 204, headers: corsHeaders(request.headers.get("Origin") || "") });
    }

    // Health check (public)
    if (url.pathname === "/cf-health") {
      return new Response("ok", { status: 200 });
    }

    // Protect infrastructure endpoints from public access
    if (url.pathname === "/metrics" || url.pathname === "/ready") {
      return new Response(JSON.stringify({ error: "forbidden" }), {
        status: 403,
        headers: { "Content-Type": "application/json" },
      });
    }

    // Rate limit
    const ip = request.headers.get("CF-Connecting-IP") || "unknown";
    if (isRateLimited(ip)) {
      return new Response(JSON.stringify({ success: false, error: { code: "rate_limited", message: "too many requests" } }), {
        status: 429,
        headers: { "Content-Type": "application/json", "Retry-After": "60" },
      });
    }

    // API requests -> proxy to backend
    if (isApiRequest(url)) {
      const tenant = await resolveTenant(request, env);
      if (!tenant && url.pathname.startsWith("/api/v1/tenant")) {
        return new Response(JSON.stringify({ success: false, error: { code: "tenant_not_found", message: "tenant not found for this domain" } }), {
          status: 404,
          headers: { "Content-Type": "application/json" },
        });
      }
      const res = await proxyToBackend(request, env, tenant);
      // Add CORS headers to backend responses
      const corsRes = new Response(res.body, res);
      corsRes.headers.set("Access-Control-Allow-Origin", request.headers.get("Origin") || "*");
      corsRes.headers.set("Access-Control-Allow-Credentials", "true");
      return corsRes;
    }

    // Static assets and SPA fallback -> pass through to backend origin static server
    return proxyToBackend(request, env, null);
  },
};
