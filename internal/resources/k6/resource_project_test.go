package k6_test

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	"github.com/grafana/k6-cloud-openapi-client-go/k6"

	"github.com/grafana/terraform-provider-grafana/v3/internal/common"
	"github.com/grafana/terraform-provider-grafana/v3/internal/testutils"
)

var defaultIDRegexp = regexp.MustCompile(`^\d{5,8}$`)

func TestAccProject_basic(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	var project k6.ProjectApiModel

	projectName := "Terraform Test Project " + acctest.RandString(8)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy: resource.ComposeTestCheckFunc(
			projectCheckExists.destroyed(&project),
		),
		Steps: []resource.TestStep{
			{
				Config: testutils.TestAccExampleWithReplace(t, "resources/grafana_k6_project/resource.tf", map[string]string{
					"Terraform Test Project": projectName,
				}),
				Check: resource.ComposeTestCheckFunc(
					projectCheckExists.exists("grafana_k6_project.test_project", &project),
					resource.TestMatchResourceAttr("grafana_k6_project.test_project", "id", defaultIDRegexp),
					resource.TestCheckResourceAttr("grafana_k6_project.test_project", "name", projectName),
					resource.TestMatchResourceAttr("grafana_k6_project.test_project", "is_default", regexp.MustCompile(`^(true|false)$`)),
					resource.TestCheckResourceAttrSet("grafana_k6_project.test_project", "grafana_folder_uid"),
					testutils.CheckLister("grafana_k6_project.test_project"),
				),
			},
			{
				ResourceName:            "grafana_k6_project.test_project",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"grafana_folder_uid"},
			},
			// Delete the project and check that TF sees a difference
			{
				PreConfig: func() {
					commonClient := testutils.Provider.Meta().(*common.Client)
					client := commonClient.K6APIClient
					config := commonClient.K6APIConfig

					ctx := context.WithValue(context.Background(), k6.ContextAccessToken, config.Token)
					deleteReq := client.ProjectsAPI.ProjectsDestroy(ctx, project.Id).XStackId(config.StackID)

					_, err := deleteReq.Execute()
					if err != nil {
						t.Fatalf("error deleting project: %s", err)
					}
				},
				RefreshState:       true,
				ExpectNonEmptyPlan: true,
			},
			// Recreate the project
			{
				Config: testutils.TestAccExampleWithReplace(t, "resources/grafana_k6_project/resource.tf", map[string]string{
					"Terraform Test Project": projectName,
				}),
				Check: resource.ComposeAggregateTestCheckFunc(
					projectCheckExists.exists("grafana_k6_project.test_project", &project),
					resource.TestCheckResourceAttr("grafana_k6_project.test_project", "name", projectName),
				),
			},
			// Change the title of a project. This shouldn't recreate the project.
			{
				Config: testutils.TestAccExampleWithReplace(t, "resources/grafana_k6_project/resource.tf", map[string]string{
					projectName: projectName + " Updated",
				}),
				Check: resource.ComposeTestCheckFunc(
					testAccProjectWasntRecreated("grafana_k6_project.test_project", &project),
					testAccProjectUnchangedAttr("grafana_k6_project.test_project", "id", func() string { return strconv.Itoa(int(project.GetId())) }),
					resource.TestCheckResourceAttr("grafana_k6_project.test_project", "name", projectName+" Updated"),
					testAccProjectUnchangedAttr("grafana_k6_project.test_project", "grafana_folder_uid", project.GetGrafanaFolderUid),
					testAccProjectUnchangedAttr("grafana_k6_project.test_project", "created", func() string { return project.GetCreated().Truncate(time.Microsecond).Format(time.RFC3339Nano) }),
				),
			},
		},
	})
}

func testAccProjectUnchangedAttr(resName, attrName string, oldValueGetter func() string) resource.TestCheckFunc {
	return resource.TestCheckResourceAttrWith(resName, attrName, func(newVal string) error {
		if oldValue := oldValueGetter(); oldValue != newVal {
			return fmt.Errorf("%s has changed: %s -> %s", attrName, oldValue, newVal)
		}
		return nil
	})
}

func testAccProjectWasntRecreated(rn string, oldProject *k6.ProjectApiModel) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		newProjectResource, ok := s.RootModule().Resources[rn]
		if !ok {
			return fmt.Errorf("project not found: %s", rn)
		}
		if newProjectResource.Primary.ID == "" {
			return fmt.Errorf("project id not set")
		}
		var newProjectID int32
		if projectID, err := strconv.Atoi(newProjectResource.Primary.ID); err != nil {
			return fmt.Errorf("could not convert project id to integer: %s", err.Error())
		} else if newProjectID, err = common.ToInt32(projectID); err != nil {
			return fmt.Errorf("could not convert project id to int32: %s", err.Error())
		}

		client := testutils.Provider.Meta().(*common.Client).K6APIClient
		config := testutils.Provider.Meta().(*common.Client).K6APIConfig

		ctx := context.WithValue(context.Background(), k6.ContextAccessToken, config.Token)
		newProject, _, err := client.ProjectsAPI.ProjectsRetrieve(ctx, newProjectID).
			XStackId(config.StackID).
			Execute()
		if err != nil {
			return fmt.Errorf("error getting project: %s", err)
		}
		if newProject.Created != oldProject.Created {
			return fmt.Errorf("project creation date has changed: %s -> %s", oldProject.Created, newProject.Created)
		}
		if !oldProject.GetUpdated().Before(newProject.GetUpdated()) {
			return fmt.Errorf("project update date hasn't changed: %s -> %s", oldProject.Updated, newProject.Updated)
		}
		return nil
	}
}
