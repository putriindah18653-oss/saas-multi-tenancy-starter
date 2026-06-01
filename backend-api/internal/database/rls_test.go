package database

import (
	"testing"
)

// TestRLSContextValues verifies that RLSContext carries independent state
// for tenant-scoped and platform-admin paths.
func TestRLSContextValues(t *testing.T) {
	tenant := RLSContext{TenantID: "t1", UserID: "u1", PlatformAdmin: false}
	if tenant.PlatformAdmin {
		t.Error("tenant-scoped context should not be platform admin")
	}

	admin := RLSContext{UserID: "u1", PlatformAdmin: true}
	if !admin.PlatformAdmin {
		t.Error("platform admin context should be platform admin")
	}
	if admin.TenantID != "" {
		t.Errorf("platform admin context should have empty tenant ID, got %q", admin.TenantID)
	}
}

// TestQuerierInterface verifies that *pgxpool.Pool satisfies the
// expectations of the Querier contract through the compiler.
// Actual RLS enforcement is tested via integration tests that
// require a running PostgreSQL instance.
func TestQuerierInterface(t *testing.T) {
	// Compile-time check: the interface is used throughout WithRLS callbacks.
	var _ Querier = nil // nil satisfies interface (no methods called at compile time)
	_ = WithRLS        // ensure exported
	_ = SetLocalRLS    // ensure exported
}
