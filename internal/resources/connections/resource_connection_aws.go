package connections

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/grafana/terraform-provider-grafana/internal/common"
	"github.com/grafana/terraform-provider-grafana/internal/common/connections"
)

func ResourceConnectionAWS() *schema.Resource {
	return &schema.Resource{
		Description: `
An AWS connection defines a connection between Grafana Cloud and AWS   
`,

		CreateContext: ResourceCreate,
		ReadContext:   ResourceRead,
		UpdateContext: ResourceUpdate,
		DeleteContext: ResourceDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"stack_id": {
				Description: "The StackID of the grafana cloud instance",
				Type:        schema.TypeString,
				Required:    true,
			},
			"name": {
				Description: "The name of the AWS Connection.",
				Type:        schema.TypeString,
				Required:    true,
			},
			"aws_account_id": {
				Description: "The AWS Account ID being connected to",
				Type:        schema.TypeString,
				Required:    true,
			},
			"role_arn": {
				Description: "The role arn to use while making the connection",
				Type:        schema.TypeString,
				Required:    true,
			},
			"regions": {
				Description: "The regions this connection applies to",
				Type:        schema.TypeList,
				Required:    true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
		},
	}
}

func ResourceCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*common.Client).ConnectionsAPI
	stackID, c := makeConnection(d)
	err := client.CreateAWSConnection(ctx, stackID, c)
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(toID(stackID, c.Name))
	return ResourceRead(ctx, d, meta)
}

func ResourceRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(*common.Client).ConnectionsAPI
	connection, err := c.GetAWSConnection(ctx, d.Get("stack_id").(string), d.Get("name").(string))
	if err, shouldReturn := common.CheckReadError("AWSConnection", d, err); shouldReturn {
		return err
	}

	err = d.Set("name", connection.Name)
	if err != nil {
		return diag.FromErr(err)
	}
	err = d.Set("role_arn", connection.RoleARN)
	if err != nil {
		return diag.FromErr(err)
	}
	err = d.Set("aws_account_id", connection.AWSAccountID)
	if err != nil {
		return diag.FromErr(err)
	}
	err = d.Set("regions", common.StringSliceToList(connection.Regions))
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func ResourceUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*common.Client).ConnectionsAPI
	stackID, c := makeConnection(d)
	err := client.UpdateAWSConnection(ctx, stackID, c)
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(toID(stackID, c.Name))
	return ResourceRead(ctx, d, meta)
}

func ResourceDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*common.Client).ConnectionsAPI

	err := client.DeleteAWSConnection(ctx, d.Get("stack_id").(string), d.Get("name").(string))
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId("")
	return nil
}

func makeConnection(d *schema.ResourceData) (string, connections.AWSConnection) {
	stackID := d.Get("stack_id").(string)
	return stackID, connections.AWSConnection{
		Name:         d.Get("name").(string),
		AWSAccountID: d.Get("aws_account_id").(string),
		RoleARN:      d.Get("role_arn").(string),
		Regions:      common.ListToStringSlice(d.Get("regions").([]interface{})),
	}
}

func toID(stackID, name string) string {
	return fmt.Sprintf("%s_%s", stackID, name)
}
