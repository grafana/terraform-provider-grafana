package cloud

import (
	"encoding/json"
	"testing"

	"github.com/grafana/grafana-com-public-clients/go/gcom"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
)

func TestUnitStackUpdateRequestV1_DeleteProtectionFalseInJSONPayload(t *testing.T) {
	t.Parallel()

	// False deleteProtection must appear in JSON (same as updateStack).
	stack := gcom.StackUpdateRequestV1{
		DeleteProtection: *gcom.NewNullableBool(common.Ref(false)),
	}

	payload, err := json.Marshal(stack)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}

	raw, ok := decoded["deleteProtection"]
	if !ok {
		t.Fatalf("deleteProtection missing from JSON payload: %s", string(payload))
	}

	if b, ok := raw.(bool); !ok || b {
		t.Fatalf("deleteProtection: got %v (%T), want false bool", raw, raw)
	}
}
