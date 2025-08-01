package grafana_test

import (
	"os"
	"strings"
	"testing"

	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDatasourceDashboard_basic(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t)

	var dashboard models.DashboardFullWithMeta
	checks := []resource.TestCheckFunc{
		dashboardCheckExists.exists("grafana_dashboard.test", &dashboard),
		resource.TestCheckResourceAttr(
			"data.grafana_dashboard.from_uid", "title", "Production Overview",
		),
		resource.TestMatchResourceAttr(
			"data.grafana_dashboard.from_uid", "dashboard_id", common.IDRegexp,
		),
		resource.TestCheckResourceAttr(
			"data.grafana_dashboard.from_uid", "uid", "test-ds-dashboard-uid",
		),
		resource.TestCheckResourceAttr(
			"data.grafana_dashboard.from_uid", "url", strings.TrimRight(os.Getenv("GRAFANA_URL"), "/")+"/d/test-ds-dashboard-uid/production-overview",
		),
	}

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             dashboardCheckExists.destroyed(&dashboard, nil),
		Steps: []resource.TestStep{
			{
				Config: testutils.TestAccExample(t, "data-sources/grafana_dashboard/data-source.tf"),
				Check:  resource.ComposeTestCheckFunc(checks...),
			},
		},
	})
}
