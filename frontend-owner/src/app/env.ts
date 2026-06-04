/**
 * Resolves the backend API base URL at runtime.
 *
 * Priority: VITE_API_BASE_URL env var → current origin:8080 → localhost:8080.
 * Uses `window.location` in the browser and falls back to `localhost` during SSR/SSG.
 */
function resolveApiBaseUrl() {
  const configured = (import.meta.env.VITE_API_BASE_URL as string | undefined)?.trim()
  if (configured) return configured

  if (import.meta.env.PROD) {
    throw new Error('VITE_API_BASE_URL is required for production builds')
  }

  if (typeof window !== 'undefined') {
    return `${window.location.protocol}//${window.location.hostname}:8080/api/v1`
  }

  return 'http://localhost:8080/api/v1'
}

/** Resolved API base URL. Use `appEnv.apiBaseUrl` elsewhere. */
export const appEnv = {
  apiBaseUrl: resolveApiBaseUrl(),
}
