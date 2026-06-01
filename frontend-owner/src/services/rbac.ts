import type { UserProfile } from '@/contracts/api'

export const OWNER_ROLES = ['owner-app', 'admin', 'super_admin', 'platform_admin', 'support_agent', 'billing_admin', 'auditor'] as const

export type OwnerRole = (typeof OWNER_ROLES)[number]

export type OwnerPermission =
  | 'app.tenants.read'
  | 'app.tenants.create'
  | 'app.tenants.update'
  | 'app.tenants.delete'
  | 'app.users.read'
  | 'app.users.manage'
  | 'app.audit.read'
  | 'app.settings.read'
  | 'app.settings.update'

const allOwnerPermissions: OwnerPermission[] = [
  'app.tenants.read',
  'app.tenants.create',
  'app.tenants.update',
  'app.tenants.delete',
  'app.users.read',
  'app.users.manage',
  'app.audit.read',
  'app.settings.read',
  'app.settings.update',
]

const ownerRolePermissions: Record<OwnerRole, OwnerPermission[]> = {
  'owner-app': allOwnerPermissions,
  admin: allOwnerPermissions.filter((permission) => permission !== 'app.tenants.delete'),
  super_admin: allOwnerPermissions,
  platform_admin: ['app.tenants.read', 'app.tenants.create', 'app.tenants.update', 'app.users.read', 'app.users.manage', 'app.audit.read', 'app.settings.read', 'app.settings.update'],
  support_agent: ['app.tenants.read', 'app.users.read'],
  billing_admin: ['app.tenants.read'],
  auditor: ['app.tenants.read', 'app.audit.read'],
}

export function isOwnerRole(role: string | undefined | null): role is OwnerRole {
  return !!role && OWNER_ROLES.includes(role as OwnerRole)
}

export function getOwnerPermissions(user: UserProfile | null | undefined): OwnerPermission[] {
  if (user?.permissions?.length) return user.permissions as OwnerPermission[]
  const role = user?.app_role
  if (!isOwnerRole(role)) return []
  return ownerRolePermissions[role]
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

export function canOwner(user: UserProfile | null | undefined, permission: OwnerPermission): boolean {
  return hasPermission(getOwnerPermissions(user), permission)
}
