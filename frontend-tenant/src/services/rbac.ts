import type { TenantMembership } from '@/contracts/api'

/**
 * Tenant roles. `owner-tenant` and `tenant_owner` are identical in permissions;
 * both represent the tenant owner (backward compatible backend naming).
 * Similarly `admin` and `tenant_admin` are equivalent.
 */
export const TENANT_ROLES = ['owner-tenant', 'admin', 'finance', 'support', 'tenant_owner', 'tenant_admin', 'manager', 'staff', 'viewer'] as const

export type TenantRole = (typeof TENANT_ROLES)[number]

export const TENANT_ROLE_OPTIONS: Array<{ value: TenantRole; label: string }> =
  TENANT_ROLES.map((role) => ({ value: role, label: role }))

export const TENANT_OWNER_ROLES: TenantRole[] = ['owner-tenant', 'tenant_owner']

export function isTenantOwnerRole(role: string | undefined | null): boolean {
  return isTenantRole(role) && TENANT_OWNER_ROLES.includes(role)
}

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

const tenantRolePermissions: Record<TenantRole, TenantPermission[]> = {
  'owner-tenant': allOwnerPermissions(),
  tenant_owner: allOwnerPermissions(),
  admin: adminPermissions(),
  tenant_admin: adminPermissions(),
  manager: ['tenant.dashboard.read', 'tenant.reports.read'],
  staff: ['tenant.dashboard.read'],
  finance: ['tenant.dashboard.read', 'tenant.billing.read', 'tenant.billing.manage', 'tenant.reports.read'],
  support: ['tenant.dashboard.read', 'tenant.users.read', 'tenant.support.manage'],
  viewer: ['tenant.dashboard.read', 'tenant.reports.read'],
}

function allOwnerPermissions(): TenantPermission[] {
  return [
    'tenant.dashboard.read',
    'tenant.users.read', 'tenant.users.invite', 'tenant.users.update', 'tenant.users.remove',
    'tenant.settings.read', 'tenant.settings.update',
    'tenant.audit.read',
    'tenant.billing.read', 'tenant.billing.manage',
    'tenant.reports.read',
  ]
}

function adminPermissions(): TenantPermission[] {
  return [
    'tenant.dashboard.read',
    'tenant.users.read', 'tenant.users.update', 'tenant.users.remove',
    'tenant.settings.read', 'tenant.settings.update',
    'tenant.audit.read',
    'tenant.reports.read',
  ]
}

export function isTenantRole(role: string | undefined | null): role is TenantRole {
  return !!role && TENANT_ROLES.includes(role as TenantRole)
}

export function getTenantPermissions(membership: TenantMembership | null | undefined): TenantPermission[] {
  const role = membership?.role
  if (!isTenantRole(role)) return []
  return tenantRolePermissions[role]
}

export function hasPermission(current: TenantPermission[] | string[] | undefined | null, required: TenantPermission | string): boolean {
  if (!current || current.length === 0) return false
  return current.includes(required as TenantPermission)
}

export function hasAnyPermission(current: TenantPermission[] | string[] | undefined | null, required: Array<TenantPermission | string>): boolean {
  if (!current || current.length === 0) return false
  return required.some((perm) => current.includes(perm as TenantPermission))
}

export function hasAllPermissions(current: TenantPermission[] | string[] | undefined | null, required: Array<TenantPermission | string>): boolean {
  if (!current || current.length === 0) return false
  return required.every((perm) => current.includes(perm as TenantPermission))
}

export function canTenant(membership: TenantMembership | null | undefined, permission: TenantPermission): boolean {
  return hasPermission(getTenantPermissions(membership), permission)
}
