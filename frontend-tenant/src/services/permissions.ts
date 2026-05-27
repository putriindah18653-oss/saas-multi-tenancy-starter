export {
  hasPermission,
  hasAnyPermission,
  hasAllPermissions,
  canTenant,
  getTenantPermissions,
  isTenantRole,
} from '@/services/rbac'

export type { TenantPermission, Permission, TenantRole } from '@/services/rbac'
