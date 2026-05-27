export {
  hasPermission,
  hasAnyPermission,
  hasAllPermissions,
  canOwner,
  getOwnerPermissions,
  isOwnerRole,
} from '@/services/rbac'

export type { OwnerPermission, Permission, OwnerRole } from '@/services/rbac'
