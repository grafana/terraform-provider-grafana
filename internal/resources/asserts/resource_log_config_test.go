package asserts_test

import (
	"testing"

	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
)

func TestAccAssertsLogConfig_basic(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	// This test is skipped because PR2 implements CREATE/UPDATE but not DELETE
	// Without DELETE, we can't properly clean up test resources
	// The full CRUD test will be implemented in PR3
	t.Skip("Skipping test - PR2 implements CREATE/UPDATE but not DELETE, can't clean up test resources")
}

func TestAccAssertsLogConfig_minimal(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	// This test is skipped because PR2 implements CREATE/UPDATE but not DELETE
	// Without DELETE, we can't properly clean up test resources
	// The full CRUD test will be implemented in PR3
	t.Skip("Skipping test - PR2 implements CREATE/UPDATE but not DELETE, can't clean up test resources")
}
