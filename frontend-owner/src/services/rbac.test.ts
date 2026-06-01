import { describe, it, expect } from 'vitest'
import {
  isOwnerRole,
  getOwnerPermissions,
  hasPermission,
  hasAnyPermission,
  hasAllPermissions,
  canOwner,
  OWNER_ROLES,
} from '@/services/rbac'
import type { UserProfile } from '@/contracts/api'

function makeUser(overrides: Partial<UserProfile> = {}): UserProfile {
  return {
    id: 'u1',
    email: 'test@example.com',
    ...overrides,
  }
}

describe('isOwnerRole', () => {
  it('returns true for known roles', () => {
    for (const role of OWNER_ROLES) {
      expect(isOwnerRole(role)).toBe(true)
    }
  })

  it('returns false for unknown roles', () => {
    expect(isOwnerRole('unknown')).toBe(false)
    expect(isOwnerRole('')).toBe(false)
    expect(isOwnerRole(null)).toBe(false)
    expect(isOwnerRole(undefined)).toBe(false)
  })
})

describe('getOwnerPermissions', () => {
  it('uses explicit permissions when available', () => {
    const user = makeUser({ permissions: ['app.tenants.read', 'app.tenants.create'] })
    expect(getOwnerPermissions(user)).toEqual(['app.tenants.read', 'app.tenants.create'])
  })

  it('falls back to role-based permissions', () => {
    const user = makeUser({ app_role: 'auditor' })
    const perms = getOwnerPermissions(user)
    expect(perms).toContain('app.tenants.read')
    expect(perms).toContain('app.audit.read')
    expect(perms).not.toContain('app.tenants.delete')
  })

  it('returns empty array for unknown roles', () => {
    const user = makeUser({ app_role: 'unknown' })
    expect(getOwnerPermissions(user)).toEqual([])
  })

  it('returns empty for null/undefined user', () => {
    expect(getOwnerPermissions(null)).toEqual([])
    expect(getOwnerPermissions(undefined)).toEqual([])
  })

  it('owner-app has all permissions', () => {
    const user = makeUser({ app_role: 'owner-app' })
    const perms = getOwnerPermissions(user)
    expect(perms).toContain('app.tenants.read')
    expect(perms).toContain('app.tenants.create')
    expect(perms).toContain('app.tenants.update')
    expect(perms).toContain('app.tenants.delete')
    expect(perms).toContain('app.audit.read')
    expect(perms).toContain('app.settings.read')
    expect(perms).toContain('app.settings.update')
  })

  it('support_agent can read tenants and users only', () => {
    const user = makeUser({ app_role: 'support_agent' })
    const perms = getOwnerPermissions(user)
    expect(perms).toEqual(['app.tenants.read', 'app.users.read'])
  })

  it('billing_admin can only read tenants', () => {
    const user = makeUser({ app_role: 'billing_admin' })
    expect(getOwnerPermissions(user)).toEqual(['app.tenants.read'])
  })

  it('admin has all except delete', () => {
    const user = makeUser({ app_role: 'admin' })
    const perms = getOwnerPermissions(user)
    expect(perms).toContain('app.tenants.read')
    expect(perms).not.toContain('app.tenants.delete')
  })
})

describe('hasPermission', () => {
  it('returns true when permission is present', () => {
    expect(hasPermission(['app.tenants.read', 'app.tenants.create'], 'app.tenants.read')).toBe(true)
  })

  it('returns false when permission is absent', () => {
    expect(hasPermission(['app.tenants.read'], 'app.tenants.create')).toBe(false)
  })

  it('returns false for empty/null permissions array', () => {
    expect(hasPermission([], 'app.tenants.read')).toBe(false)
    expect(hasPermission(null, 'app.tenants.read')).toBe(false)
    expect(hasPermission(undefined, 'app.tenants.read')).toBe(false)
  })
})

describe('hasAnyPermission', () => {
  it('returns true when at least one matches', () => {
    expect(hasAnyPermission(['app.tenants.read'], ['app.tenants.read', 'app.tenants.delete'])).toBe(true)
  })

  it('returns false when none match', () => {
    expect(hasAnyPermission(['app.tenants.read'], ['app.tenants.delete', 'app.audit.read'])).toBe(false)
  })
})

describe('hasAllPermissions', () => {
  it('returns true when all required permissions are present', () => {
    expect(hasAllPermissions(['app.tenants.read', 'app.tenants.create', 'app.tenants.update'], ['app.tenants.read', 'app.tenants.create'])).toBe(true)
  })

  it('returns false when any is missing', () => {
    expect(hasAllPermissions(['app.tenants.read'], ['app.tenants.read', 'app.tenants.create'])).toBe(false)
  })
})

describe('canOwner', () => {
  it('returns true for owner-app on any permission', () => {
    const user = makeUser({ app_role: 'owner-app' })
    expect(canOwner(user, 'app.tenants.delete')).toBe(true)
    expect(canOwner(user, 'app.settings.update')).toBe(true)
  })

  it('returns false for billing_admin on non-read permission', () => {
    const user = makeUser({ app_role: 'billing_admin' })
    expect(canOwner(user, 'app.tenants.read')).toBe(true)
    expect(canOwner(user, 'app.tenants.create')).toBe(false)
    expect(canOwner(user, 'app.tenants.update')).toBe(false)
  })

  it('returns false for null user', () => {
    expect(canOwner(null, 'app.tenants.read')).toBe(false)
  })
})
