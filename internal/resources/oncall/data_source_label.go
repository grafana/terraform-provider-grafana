package oncall

import (
	"context"

	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

var dataSourceLabelName = "grafana_oncall_label"

type labelDataSourceModel struct {
	ID    basetypes.StringValue `tfsdk:"id"`
	Key   basetypes.StringValue `tfsdk:"key"`
	Value basetypes.StringValue `tfsdk:"value"`
}

func dataSourceLabel() *common.DataSource {
	return common.NewDataSource(common.CategoryOnCall, dataSourceLabelName, &labelDataSource{})
}

type labelDataSource struct {
	basePluginFrameworkDataSource
}

func (d *labelDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = dataSourceLabelName
}

func (d *labelDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "* [HTTP API](https://grafana.com/docs/oncall/latest/oncall-api-reference/users/)",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The ID of the label.",
			},
			"key": schema.StringAttribute{
				Required:    true,
				Description: "The key for the label. Keys are 1-63 characters, can only contain alphanumeric characters or underscores, and must start and end with a letter.",
			},
			"value": schema.StringAttribute{
				Required:    true,
				Description: "The value of the label. When the label is used as a static label, values are 1-63 characters, can only contain alphanumeric characters, hyphens, underscores and periods, must start with a letter and must end with a letter or digit. When the label is used as a dynamic label, the value is a Jinja2 template evaluated when an alert is received, and is not restricted.",
			},
		},
	}
}

func (d *labelDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	// Read Terraform state data into the model
	var data labelDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	data.ID = basetypes.NewStringValue(data.Key.ValueString())
	data.Key = basetypes.NewStringValue(data.Key.ValueString())
	data.Value = basetypes.NewStringValue(data.Value.ValueString())

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
	if resp.Diagnostics.HasError() {
		return
	}
}
