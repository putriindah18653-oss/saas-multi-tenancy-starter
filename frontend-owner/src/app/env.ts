function resolveApiBaseUrl() {
  const configured = import.meta.env.VITE_API_BASE_URL as string | undefined
  if (configured) return configured

  if (typeof window !== 'undefined') {
    return `${window.location.protocol}//${window.location.hostname}:8080/api/v1`
  }

  return 'http://localhost:8080/api/v1'
}

export const appEnv = {
  apiBaseUrl: resolveApiBaseUrl(),
}
