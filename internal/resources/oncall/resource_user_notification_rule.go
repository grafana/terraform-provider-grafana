package oncall

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"

	onCallAPI "github.com/grafana/amixr-api-go-client"
	"github.com/grafana/terraform-provider-grafana/v3/internal/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

var userNotificationRuleTypeOptions = []string{
	"wait",
	"notify_by_slack",
	"notify_by_sms",
	"notify_by_phone_call",
	"notify_by_telegram",
	"notify_by_email",
}

var userNotificationRuleTypeOptionsVerbal = strings.Join(userNotificationRuleTypeOptions, ", ")

func resourceUserNotificationRule() *common.Resource {
	schema := &schema.Resource{
		Description: `
* [HTTP API](https://grafana.com/docs/oncall/latest/oncall-api-reference/personal_notification_rules/)
`,
		CreateContext: withClient[schema.CreateContextFunc](resourceUserNotificationRuleCreate),
		ReadContext:   withClient[schema.ReadContextFunc](resourceUserNotificationRuleRead),
		UpdateContext: withClient[schema.UpdateContextFunc](resourceUserNotificationRuleUpdate),
		DeleteContext: withClient[schema.DeleteContextFunc](resourceUserNotificationRuleDelete),
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"user_id": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "User ID",
			},
			"position": {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "Personal notification rules execute one after another starting from position=0. Position=-1 will put the escalation policy to the end of the list. A new escalation policy created with a position of an existing escalation policy will move the old one (and all following) down on the list.",
			},
			"duration": {
				Type:        schema.TypeInt,
				Optional:    true,
				Default:     300,
				Description: "A time in secs when type wait is chosen for type.",
			},
			"important": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Boolean value which indicates if a rule is “important”",
			},
			"type": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringInSlice(userNotificationRuleTypeOptions, false),
				Description:  fmt.Sprintf("The type of notification rule. Can be %s", userNotificationRuleTypeOptionsVerbal),
			},
		},
	}

	return common.NewLegacySDKResource(
		common.CategoryOnCall,
		"grafana_oncall_user_notification_rule",
		resourceID,
		schema,
	).WithLister(oncallListerFunction(listUserNotificationRules))
}

func listUserNotificationRules(client *onCallAPI.Client, listOptions onCallAPI.ListOptions) (ids []string, nextPage *string, err error) {
	resp, _, err := client.UserNotificationRules.ListUserNotificationRules(&onCallAPI.ListUserNotificationRuleOptions{ListOptions: listOptions})
	if err != nil {
		return nil, nil, err
	}
	for _, i := range resp.UserNotificationRules {
		ids = append(ids, i.ID)
	}
	return ids, resp.Next, nil
}

func resourceUserNotificationRuleCreate(ctx context.Context, d *schema.ResourceData, client *onCallAPI.Client) diag.Diagnostics {
	userId := d.Get("user_id").(string)
	position := d.Get("position").(int)
	duration := d.Get("duration").(int)
	important := d.Get("important").(bool)
	ruleType := d.Get("type").(string)

	createOptions := &onCallAPI.CreateUserNotificationRuleOptions{
		UserId:    userId,
		Position:  &position,
		Duration:  &duration,
		Important: important,
		Type:      ruleType,
	}

	userNotificationRule, _, err := client.UserNotificationRules.CreateUserNotificationRule(createOptions)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(userNotificationRule.ID)

	return resourceUserNotificationRuleRead(ctx, d, client)
}

func resourceUserNotificationRuleRead(ctx context.Context, d *schema.ResourceData, client *onCallAPI.Client) diag.Diagnostics {
	userNotificationRule, r, err := client.UserNotificationRules.GetUserNotificationRule(d.Id(), &onCallAPI.GetUserNotificationRuleOptions{})
	if err != nil {
		if r != nil && r.StatusCode == http.StatusNotFound {
			log.Printf("[WARN] removing user notification rule %s from state because it no longer exists", d.Id())
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	d.Set("user_id", userNotificationRule.UserId)
	d.Set("position", userNotificationRule.Position)
	d.Set("duration", userNotificationRule.Duration)
	d.Set("important", userNotificationRule.Important)
	d.Set("type", userNotificationRule.Type)

	return nil
}

func resourceUserNotificationRuleUpdate(ctx context.Context, d *schema.ResourceData, client *onCallAPI.Client) diag.Diagnostics {
	position := d.Get("position").(int)
	duration := d.Get("duration").(int)
	ruleType := d.Get("type").(string)

	updateOptions := &onCallAPI.UpdateUserNotificationRuleOptions{
		Position: &position,
		Duration: &duration,
		Type:     ruleType,
	}

	route, _, err := client.UserNotificationRules.UpdateUserNotificationRule(d.Id(), updateOptions)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(route.ID)
	return resourceRouteRead(ctx, d, client)
}

func resourceUserNotificationRuleDelete(ctx context.Context, d *schema.ResourceData, client *onCallAPI.Client) diag.Diagnostics {
	_, err := client.UserNotificationRules.DeleteUserNotificationRule(d.Id(), &onCallAPI.DeleteUserNotificationRuleOptions{})
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")

	return nil
}
