package grafana_test

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDatasourceFolders_basic(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t)

	var folderA models.Folder
	var folderB models.Folder
	titleBase := "test-folder-"
	uidBase := "test-ds-folder-uid-"
	checks := []resource.TestCheckFunc{
		folderCheckExists.exists("grafana_folder.test_a", &folderA),
		folderCheckExists.exists("grafana_folder.test_b", &folderB),
		resource.TestCheckResourceAttr(
			"data.grafana_folders.test", "folders.#", "2",
		),
		resource.TestCheckTypeSetElemNestedAttrs("data.grafana_folders.test", "folders.*", map[string]string{
			"uid":   uidBase + "a",
			"title": titleBase + "a",
			"url":   fmt.Sprintf("%s/dashboards/f/%s/%s", strings.TrimRight(os.Getenv("GRAFANA_URL"), "/"), uidBase+"a", titleBase+"a"),
		}),
		resource.TestCheckTypeSetElemNestedAttrs("data.grafana_folders.test", "folders.*", map[string]string{
			"uid":   uidBase + "b",
			"title": titleBase + "b",
			"url":   fmt.Sprintf("%s/dashboards/f/%s/%s", strings.TrimRight(os.Getenv("GRAFANA_URL"), "/"), uidBase+"b", titleBase+"b"),
		}),
	}

	// TODO: Make parallelizable
	resource.Test(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy: resource.ComposeTestCheckFunc(
			folderCheckExists.destroyed(&folderA, nil),
			folderCheckExists.destroyed(&folderB, nil),
		),
		Steps: []resource.TestStep{
			{
				Config: testutils.TestAccExample(t, "data-sources/grafana_folders/data-source.tf"),
				Check:  resource.ComposeTestCheckFunc(checks...),
			},
		},
	})
}
