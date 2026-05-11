package cloud

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/grafana/grafana-com-public-clients/go/gcom"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

var pluginInstallConflictBackoff = func(resp *http.Response, err error) bool {
	if err != nil && strings.Contains(strings.ToLower(err.Error()), "conflict") {
		time.Sleep(10 * time.Second) // Do not retry too fast on in-progress install/delete conflicts
		return true
	}
	return false
}

var (
	resourcePluginInstallationID = common.NewResourceID(
		common.StringIDField("stackSlug"),
		common.StringIDField("pluginSlug"),
	)
)

const LatestVersion = "latest"

func resourcePluginInstallation() *common.Resource {
	schema := &schema.Resource{
		Description: `
Manages Grafana Cloud Plugin Installations.

* [Plugin management](https://grafana.com/docs/grafana/latest/administration/plugin-management/)

Required access policy scopes:

* stack-plugins:read
* stack-plugins:write
* stack-plugins:delete
`,
		Schema: map[string]*schema.Schema{
			"stack_slug": {
				Description: "The stack id to which the plugin should be installed.",
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
			},
			"slug": {
				Description: "Slug of the plugin to be installed.",
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
			},
			"version": {
				Description: "Version of the plugin to be installed. Defaults to 'latest' and installs the most recent version. Terraform will detect new version as drift for plan/apply.",
				Type:        schema.TypeString,
				Optional:    true,
				Default:     LatestVersion,
				ForceNew:    true,
			},
		},
		CreateContext: withClient[schema.CreateContextFunc](resourcePluginInstallationCreate),
		ReadContext:   withClient[schema.ReadContextFunc](resourcePluginInstallationRead),
		UpdateContext: nil,
		DeleteContext: withClient[schema.DeleteContextFunc](resourcePluginInstallationDelete),
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
	}

	return common.NewLegacySDKResource(
		common.CategoryCloud,
		"grafana_cloud_plugin_installation",
		resourcePluginInstallationID,
		schema,
	).WithLister(cloudListerFunction(listStackPlugins))
}

func listStackPlugins(ctx context.Context, client *gcom.APIClient, data *ListerData) ([]string, error) {
	stacks, err := data.Stacks(ctx, client)
	if err != nil {
		return nil, err
	}

	var pluginIDs []string
	for _, stack := range stacks {
		var plugins *gcom.GetInstancePlugins200Response
		plErr := RetryGCOM(ctx, GCOMRetryConfig{}, func() (*http.Response, error) {
			p, hr, pe := client.InstancesAPI.GetInstancePlugins(ctx, stack.Slug).Execute()
			plugins = p
			return hr, pe
		})
		if plErr != nil {
			return nil, plErr
		}
		for _, plugin := range plugins.Items {
			pluginIDs = append(pluginIDs, resourcePluginInstallationID.Make(stack.Slug, plugin.PluginSlug))
		}
	}

	return pluginIDs, nil
}

func resourcePluginInstallationCreate(ctx context.Context, d *schema.ResourceData, client *gcom.APIClient) diag.Diagnostics {
	stackSlug := d.Get("stack_slug").(string)
	pluginSlug := d.Get("slug").(string)
	version := d.Get("version").(string)

	req := gcom.PostInstancePluginsRequest{
		Plugin:  pluginSlug,
		Version: common.Ref(version),
	}

	err := RetryGCOM(ctx, GCOMRetryConfig{OnTransient: pluginInstallConflictBackoff}, func() (*http.Response, error) {
		_, hr, e := client.InstancesAPI.PostInstancePlugins(ctx, stackSlug).
			PostInstancePluginsRequest(req).
			XRequestId(ClientRequestID()).Execute()
		return hr, e
	})
	if err != nil {
		return apiError(err)
	}

	d.SetId(resourcePluginInstallationID.Make(stackSlug, pluginSlug))

	return nil
}

func resourcePluginInstallationRead(ctx context.Context, d *schema.ResourceData, client *gcom.APIClient) diag.Diagnostics {
	split, err := resourcePluginInstallationID.Split(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	stackSlug, pluginSlug := split[0], split[1]

	var installation *gcom.FormattedApiInstancePlugin
	if ierr := RetryGCOM(ctx, GCOMRetryConfig{}, func() (*http.Response, error) {
		in, hr, ie := client.InstancesAPI.GetInstancePlugin(ctx, stackSlug.(string), pluginSlug.(string)).Execute()
		installation = in
		return hr, ie
	}); ierr != nil {
		if errDiag, shouldReturn := common.CheckReadError("plugin", d, ierr); shouldReturn {
			return errDiag
		}
	}
	desiredVersion := d.Get("version").(string)
	catalogVersion := ""
	if desiredVersion == LatestVersion {
		var catalogPlugin *gcom.FormattedApiPlugin
		cerr := RetryGCOM(ctx, GCOMRetryConfig{}, func() (*http.Response, error) {
			cp, hr, ce := client.PluginsAPI.GetPlugin(ctx, pluginSlug.(string)).Execute()
			catalogPlugin = cp
			return hr, ce
		})
		if cerr != nil {
			if errDiag, shouldReturn := common.CheckReadError("plugin", d, cerr); shouldReturn {
				return errDiag
			}
		}
		catalogVersion = catalogPlugin.Version
	}

	d.Set("stack_slug", installation.InstanceSlug)
	d.Set("slug", installation.PluginSlug)

	if desiredVersion == LatestVersion && installation.Version == catalogVersion {
		d.Set("version", LatestVersion)
	} else {
		d.Set("version", installation.Version)
	}
	d.SetId(resourcePluginInstallationID.Make(stackSlug, pluginSlug))

	return nil
}

func resourcePluginInstallationDelete(ctx context.Context, d *schema.ResourceData, client *gcom.APIClient) diag.Diagnostics {
	split, err := resourcePluginInstallationID.Split(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	stackSlug, pluginSlug := split[0], split[1]

	delErr := RetryGCOM(ctx, GCOMRetryConfig{TreatNotFoundAsSuccess: true, OnTransient: pluginInstallConflictBackoff}, func() (*http.Response, error) {
		_, hr, e := client.InstancesAPI.DeleteInstancePlugin(ctx, stackSlug.(string), pluginSlug.(string)).XRequestId(ClientRequestID()).Execute()
		return hr, e
	})
	if delErr != nil {
		return apiError(delErr)
	}

	return nil
}
