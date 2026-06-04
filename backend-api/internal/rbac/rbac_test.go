package rbac

import (
	"strings"
	"testing"
)

// TestTenantMembershipQueryEnforcesActiveTenant documents and locks in the P0
// tenant-access-lifecycle guarantee: a tenant-scoped membership lookup MUST join
// the tenants table and only allow tenants whose status is 'active'. Suspended or
// soft-deleted tenants must therefore be forbidden for every endpoint guarded by
// RequireTenantAccess (X-Tenant-ID), even when the user_tenants row is still active.
//
// Note on test strategy: rbac.Service holds a concrete *pgxpool.Pool rather than an
// interface, so wiring pgxmock would require refactoring the service to an interface
// plus pulling in a new module over the network. To keep this fix minimal and
// dependency-free we validate the query contract directly. The behavior is also
// covered end-to-end by the middleware (RequireTenantAccess) returning 403 when
// TenantMembership reports the user is not an active member of an active tenant.
func TestTenantMembershipQueryEnforcesActiveTenant(t *testing.T) {
	q := strings.ToLower(tenantMembershipQuery)

	if !strings.Contains(q, "join tenants") {
		t.Fatalf("expected membership query to join tenants table, got: %s", tenantMembershipQuery)
	}
	if !strings.Contains(q, "t.status='active'") {
		t.Fatalf("expected membership query to require active tenant status, got: %s", tenantMembershipQuery)
	}
	if !strings.Contains(q, "ut.is_active=true") {
		t.Fatalf("expected membership query to require active membership, got: %s", tenantMembershipQuery)
	}
	// Selecting the role from the membership row (not the tenant) keeps the returned
	// role scoped to the user's tenant role.
	if !strings.Contains(q, "select ut.role") {
		t.Fatalf("expected membership query to select the membership role, got: %s", tenantMembershipQuery)
	}
}
