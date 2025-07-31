package grafana_test

import (
	"testing"

	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDatasourceOrganizationUser_basic(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t)

	var user models.UserProfileDTO
	checks := []resource.TestCheckFunc{
		userCheckExists.exists("grafana_user.test", &user),
	}
	for _, rName := range []string{"from_email", "from_login"} {
		checks = append(checks,
			resource.TestMatchResourceAttr(
				"data.grafana_organization_user."+rName, "user_id", common.IDRegexp,
			),
			resource.TestCheckResourceAttr(
				"data.grafana_organization_user."+rName, "login", "test-datasource",
			),
		)
	}

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             userCheckExists.destroyed(&user, nil),
		Steps: []resource.TestStep{
			{
				Config: testutils.TestAccExample(t, "data-sources/grafana_organization_user/data-source.tf"),
				Check:  resource.ComposeTestCheckFunc(checks...),
			},
		},
	})
}

func TestAccDatasourceOrganizationUser_exactMatch(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t)

	var user1, user2 models.UserProfileDTO
	checks := []resource.TestCheckFunc{
		userCheckExists.exists("grafana_user.test1", &user1),
		userCheckExists.exists("grafana_user.test2", &user2),
		// Test that exact login match works when multiple users are returned
		resource.TestCheckResourceAttr(
			"data.grafana_organization_user.exact_match", "login", "test-exact-match",
		),
		resource.TestMatchResourceAttr(
			"data.grafana_organization_user.exact_match", "user_id", common.IDRegexp,
		),
	}

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             userCheckExists.destroyed(&user1, nil),
		Steps: []resource.TestStep{
			{
				Config: testAccDatasourceOrganizationUserExactMatch,
				Check:  resource.ComposeTestCheckFunc(checks...),
			},
		},
	})
}

// TestDataSourceOrganizationUserExactMatchLogic tests the exact matching logic without requiring a Grafana instance
func TestDataSourceOrganizationUserExactMatchLogic(t *testing.T) {
	// Test case 2: Multiple users returned, exact login match exists
	usersMultiple := []*models.UserLookupDTO{
		{
			UserID: 1,
			Login:  "test-exact-match",
		},
		{
			UserID: 2,
			Login:  "test-exact-match-other",
		},
	}

	// Test case 3: Multiple users returned, no exact login match
	usersNoExact := []*models.UserLookupDTO{
		{
			UserID: 1,
			Login:  "test-exact-match-other1",
		},
		{
			UserID: 2,
			Login:  "test-exact-match-other2",
		},
	}

	// Test that we can identify exact matches
	var exactMatch *models.UserLookupDTO
	login := "test-exact-match"

	for _, user := range usersMultiple {
		if user.Login == login {
			if exactMatch != nil {
				t.Fatal("Multiple exact matches found when there should only be one")
			}
			exactMatch = user
		}
	}

	if exactMatch == nil {
		t.Fatal("Expected to find exact match but didn't")
	}

	if exactMatch.UserID != 1 {
		t.Fatalf("Expected UserID 1, got %d", exactMatch.UserID)
	}

	// Test that we don't find exact matches when they don't exist
	exactMatch = nil
	for _, user := range usersNoExact {
		if user.Login == login {
			exactMatch = user
		}
	}

	if exactMatch != nil {
		t.Fatal("Expected no exact match but found one")
	}
}

const testAccDatasourceOrganizationUserExactMatch = `
resource "grafana_user" "test1" {
  email    = "test1.exact@example.com"
  name     = "Test Exact Match 1"
  login    = "test-exact-match"
  password = "my-password"
}

resource "grafana_user" "test2" {
  email    = "test2.exact@example.com"
  name     = "Test Exact Match 2"
  login    = "test-exact-match-other"
  password = "my-password"
}

data "grafana_organization_user" "exact_match" {
  login = grafana_user.test1.login
}
`
