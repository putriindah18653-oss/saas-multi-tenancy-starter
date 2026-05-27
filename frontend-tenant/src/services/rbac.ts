import type { TenantMembership } from '@/stores/tenant'

export const TENANT_ROLES = ['owner-tenant', 'admin', 'finance', 'support', 'tenant_owner', 'tenant_admin', 'manager', 'staff', 'viewer'] as const

export type TenantRole = (typeof TENANT_ROLES)[number]

export type TenantPermission =
  | 'tenant.dashboard.read'
  | 'tenant.users.read'
  | 'tenant.users.invite'
  | 'tenant.users.update'
  | 'tenant.users.remove'
  | 'tenant.settings.read'
  | 'tenant.settings.update'
  | 'tenant.audit.read'
  | 'tenant.billing.read'
  | 'tenant.billing.manage'
  | 'tenant.support.manage'
  | 'tenant.reports.read'

export type Permission = TenantPermission

const tenantRolePermissions: Record<TenantRole, TenantPermission[]> = {
  'owner-tenant': [
    'tenant.dashboard.read',
    'tenant.users.read',
    'tenant.users.invite',
    'tenant.users.update',
    'tenant.users.remove',
    'tenant.settings.read',
    'tenant.settings.update',
    'tenant.audit.read',
    'tenant.billing.read',
    'tenant.billing.manage',
    'tenant.reports.read',
  ],
  tenant_owner: [
    'tenant.dashboard.read',
    'tenant.users.read',
    'tenant.users.invite',
    'tenant.users.update',
    'tenant.users.remove',
    'tenant.settings.read',
    'tenant.settings.update',
    'tenant.audit.read',
    'tenant.billing.read',
    'tenant.billing.manage',
    'tenant.reports.read',
  ],
  admin: [
    'tenant.dashboard.read',
    'tenant.users.read',
    'tenant.users.update',
    'tenant.users.remove',
    'tenant.settings.read',
    'tenant.settings.update',
    'tenant.audit.read',
    'tenant.reports.read',
  ],
  tenant_admin: [
    'tenant.dashboard.read',
    'tenant.users.read',
    'tenant.users.update',
    'tenant.users.remove',
    'tenant.settings.read',
    'tenant.settings.update',
    'tenant.audit.read',
    'tenant.reports.read',
  ],
  manager: ['tenant.dashboard.read', 'tenant.reports.read'],
  staff: ['tenant.dashboard.read'],
  finance: ['tenant.dashboard.read', 'tenant.billing.read', 'tenant.billing.manage', 'tenant.reports.read'],
  support: ['tenant.dashboard.read', 'tenant.users.read', 'tenant.support.manage'],
  viewer: ['tenant.dashboard.read', 'tenant.reports.read'],
}

export function isTenantRole(role: string | undefined | null): role is TenantRole {
  return !!role && TENANT_ROLES.includes(role as TenantRole)
}

export function getTenantPermissions(membership: TenantMembership | null | undefined): TenantPermission[] {
  const role = membership?.role
  if (!isTenantRole(role)) return []
  return tenantRolePermissions[role]
}

export function hasPermission(current: Permission[] | string[] | undefined | null, required: Permission | string): boolean {
  if (!current || current.length === 0) return false
  return current.includes(required as Permission)
}

export function hasAnyPermission(current: Permission[] | string[] | undefined | null, required: Array<Permission | string>): boolean {
  if (!current || current.length === 0) return false
  return required.some((perm) => current.includes(perm as Permission))
}

export function hasAllPermissions(current: Permission[] | string[] | undefined | null, required: Array<Permission | string>): boolean {
  if (!current || current.length === 0) return false
  return required.every((perm) => current.includes(perm as Permission))
}

export function canTenant(membership: TenantMembership | null | undefined, permission: TenantPermission): boolean {
  return hasPermission(getTenantPermissions(membership), permission)
}
