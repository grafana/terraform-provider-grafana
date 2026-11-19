package appplatform

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	tfresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/require"
)

func TestConnectionPlanModifierRejectsFolderUID(t *testing.T) {
	ctx := context.Background()

	var schemaResp tfresource.SchemaResponse
	Connection().Resource.Schema(ctx, tfresource.SchemaRequest{}, &schemaResp)
	require.False(t, schemaResp.Diagnostics.HasError())

	configWithFolder := func(t *testing.T, folderUID *string) tfsdk.Config {
		t.Helper()
		plan := tfsdk.Plan{
			Schema: schemaResp.Schema,
			Raw:    tftypes.NewValue(schemaResp.Schema.Type().TerraformType(ctx), nil),
		}
		d := plan.SetAttribute(ctx, path.Root("metadata").AtName("uid"), "my-connection")
		require.False(t, d.HasError(), d)
		if folderUID != nil {
			d = plan.SetAttribute(ctx, path.Root("metadata").AtName("folder_uid"), *folderUID)
			require.False(t, d.HasError(), d)
		}
		return tfsdk.Config{Schema: plan.Schema, Raw: plan.Raw}
	}

	folder := "some-folder"
	empty := ""

	cases := []struct {
		name      string
		folderUID *string
		wantErr   bool
	}{
		{name: "no folder_uid", folderUID: nil, wantErr: false},
		{name: "empty folder_uid", folderUID: &empty, wantErr: false},
		{name: "folder_uid set", folderUID: &folder, wantErr: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := tfresource.ModifyPlanRequest{Config: configWithFolder(t, tc.folderUID)}
			resp := &tfresource.ModifyPlanResponse{}

			connectionPlanModifier(ctx, req, resp)

			require.Equal(t, tc.wantErr, resp.Diagnostics.HasError(), resp.Diagnostics)
		})
	}
}
