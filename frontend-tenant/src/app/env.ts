export const appEnv = {
  apiBaseUrl: import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080/api/v1',
  tenantHeader: import.meta.env.VITE_TENANT_HEADER || 'X-Tenant-ID',
}
