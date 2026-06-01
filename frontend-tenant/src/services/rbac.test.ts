import { describe, it, expect } from 'vitest'
import {
  isTenantRole,
  isTenantOwnerRole,
  getTenantPermissions,
  hasPermission,
  hasAnyPermission,
  hasAllPermissions,
  canTenant,
  TENANT_ROLES,
  TENANT_OWNER_ROLES,
} from '@/services/rbac'
import type { TenantMembership } from '@/contracts/api'

function makeMembership(overrides: Partial<TenantMembership> = {}): TenantMembership {
  return { tenant_id: 't1', tenant_name: 'Test', role: 'staff', ...overrides }
}

describe('isTenantRole', () => {
  it('returns true for all known roles', () => {
    for (const role of TENANT_ROLES) expect(isTenantRole(role)).toBe(true)
  })
  it('returns false for unknown/null/undefined', () => {
    expect(isTenantRole('unknown')).toBe(false)
    expect(isTenantRole(null)).toBe(false)
    expect(isTenantRole(undefined)).toBe(false)
  })
})

describe('isTenantOwnerRole', () => {
  it('returns true for owner roles', () => {
    for (const role of TENANT_OWNER_ROLES) expect(isTenantOwnerRole(role)).toBe(true)
  })
  it('returns false for non-owner', () => {
    expect(isTenantOwnerRole('staff')).toBe(false)
    expect(isTenantOwnerRole(null)).toBe(false)
  })
})

describe('getTenantPermissions', () => {
  it('returns all permissions for owner', () => {
    const perms = getTenantPermissions(makeMembership({ role: 'owner-tenant' }))
    expect(perms).toContain('tenant.users.invite')
    expect(perms).toContain('tenant.users.remove')
    expect(perms).toContain('tenant.billing.manage')
  })
  it('admin has read + update but not invite', () => {
    const perms = getTenantPermissions(makeMembership({ role: 'admin' }))
    expect(perms).toContain('tenant.users.read')
    expect(perms).toContain('tenant.users.update')
    expect(perms).not.toContain('tenant.users.invite')
  })
  it('finance has billing but not users', () => {
    const perms = getTenantPermissions(makeMembership({ role: 'finance' }))
    expect(perms).toContain('tenant.billing.read')
    expect(perms).not.toContain('tenant.users.read')
  })
  it('viewer has dashboard + reports only', () => {
    const perms = getTenantPermissions(makeMembership({ role: 'viewer' }))
    expect(perms).toEqual(['tenant.dashboard.read', 'tenant.reports.read'])
  })
  it('returns [] for null/undefined/invalid', () => {
    expect(getTenantPermissions(null)).toEqual([])
    expect(getTenantPermissions(makeMembership({ role: 'nonexistent' as any }))).toEqual([])
  })
})

describe('hasPermission', () => {
  it('true when present, false when absent', () => {
    expect(hasPermission(['tenant.dashboard.read', 'tenant.users.read'], 'tenant.users.read')).toBe(true)
    expect(hasPermission(['tenant.dashboard.read'], 'tenant.users.read')).toBe(false)
  })
  it('false for empty array', () => {
    expect(hasPermission([], 'tenant.dashboard.read')).toBe(false)
  })
})

describe('hasAnyPermission', () => {
  it('true when at least one matches', () => {
    expect(hasAnyPermission(['tenant.dashboard.read'], ['tenant.dashboard.read', 'tenant.users.read'])).toBe(true)
  })
  it('false when none match', () => {
    expect(hasAnyPermission(['tenant.dashboard.read'], ['tenant.users.read', 'tenant.audit.read'])).toBe(false)
  })
})

describe('hasAllPermissions', () => {
  it('true when all required present', () => {
    expect(hasAllPermissions(['tenant.dashboard.read', 'tenant.users.read'], ['tenant.dashboard.read', 'tenant.users.read'])).toBe(true)
  })
  it('false when any missing', () => {
    expect(hasAllPermissions(['tenant.dashboard.read'], ['tenant.dashboard.read', 'tenant.users.read'])).toBe(false)
  })
})

describe('canTenant', () => {
  it('support can read users but not invite', () => {
    const m = makeMembership({ role: 'support' })
    expect(canTenant(m, 'tenant.users.read')).toBe(true)
    expect(canTenant(m, 'tenant.users.invite')).toBe(false)
  })
  it('returns false for null membership', () => {
    expect(canTenant(null, 'tenant.dashboard.read')).toBe(false)
  })
})
