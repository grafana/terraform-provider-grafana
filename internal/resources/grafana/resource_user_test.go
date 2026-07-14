package grafana_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"

	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
)

func TestAccUser_basic(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t)

	var user models.UserProfileDTO
	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             userCheckExists.destroyed(&user, nil),
		Steps: []resource.TestStep{
			{
				Config: testAccUserConfig_basic,
				Check: resource.ComposeTestCheckFunc(
					userCheckExists.exists("grafana_user.test", &user),
					resource.TestCheckResourceAttr(
						"grafana_user.test", "email", "terraform-test@localhost",
					),
					resource.TestCheckResourceAttr(
						"grafana_user.test", "name", "Terraform Test",
					),
					resource.TestCheckResourceAttr(
						"grafana_user.test", "login", "tt",
					),
					resource.TestCheckResourceAttr(
						"grafana_user.test", "password", "abc123",
					),
					resource.TestMatchResourceAttr(
						"grafana_user.test", "id", common.IDRegexp,
					),
				),
			},
			{
				Config: testAccUserConfig_mixedCase,
				Check: resource.ComposeTestCheckFunc(
					userCheckExists.exists("grafana_user.test", &user),
					resource.TestCheckResourceAttr(
						"grafana_user.test", "email", "terraform-test@localhost",
					),
					resource.TestCheckResourceAttr(
						"grafana_user.test", "login", "tt",
					),
				),
			},
			{
				Config: testAccUserConfig_update,
				Check: resource.ComposeTestCheckFunc(
					userCheckExists.exists("grafana_user.test", &user),
					resource.TestCheckResourceAttr(
						"grafana_user.test", "email", "terraform-test-update@localhost",
					),
					resource.TestCheckResourceAttr(
						"grafana_user.test", "name", "Terraform Test Update",
					),
					resource.TestCheckResourceAttr(
						"grafana_user.test", "login", "ttu",
					),
					resource.TestCheckResourceAttr(
						"grafana_user.test", "password", "zyx987",
					),
					resource.TestCheckResourceAttr(
						"grafana_user.test", "is_admin", "true",
					),
				),
			},
			{
				ResourceName:            "grafana_user.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"password"},
			},
		},
	})
}

func TestAccUser_NeedsBasicAuth(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=9.0.0")

	// Subprocess: fresh provider server (SDK keeps one server per process; order can mask the error).
	if os.Getenv("GRAFANA_NEEDSBASICAUTH_SUBPROCESS") != "1" {
		_, file, _, _ := runtime.Caller(0)
		dir := filepath.Dir(file)
		moduleRoot := filepath.Join(dir, "..", "..", "..")
		cmd := exec.Command("go", "test", "-run", "^TestAccUser_NeedsBasicAuth$", "-v", "-count=1", "-timeout", "2m", "./internal/resources/grafana/...")
		cmd.Dir = moduleRoot
		cmd.Env = append(os.Environ(), "GRAFANA_NEEDSBASICAUTH_SUBPROCESS=1", "TF_ACC=1", "TF_ACC_OSS=true")
		for _, k := range []string{"GRAFANA_URL", "GRAFANA_AUTH", "GRAFANA_BASIC_AUTH", "GRAFANA_VERSION"} {
			if v := os.Getenv(k); v != "" {
				cmd.Env = append(cmd.Env, k+"="+v)
			}
		}
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("NeedsBasicAuth test (subprocess) failed: %v\n%s", err, out)
		}
		return
	}

	// Subprocess: run the real test
	_, token := orgScopedTest(t)
	resource.Test(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testutils.ConfigWithTokenProvider(t, token, testAccUserConfig_basic),
				ExpectError: regexp.MustCompile(`(global scope resources cannot be managed with an API key\. Use basic auth instead)`),
			},
		},
	})
}

const testAccUserConfig_basic = `
resource "grafana_user" "test" {
  email    = "terraform-test@localhost"
  name     = "Terraform Test"
  login    = "tt"
  password = "abc123"
  is_admin = false
}
`

const testAccUserConfig_mixedCase = `
resource "grafana_user" "test" {
  email    = "tErrAForm-TEST@localhost"
  name     = "Terraform Test"
  login    = "tT"
  password = "abc123"
  is_admin = false
}
`

const testAccUserConfig_update = `
resource "grafana_user" "test" {
  email    = "terraform-test-update@localhost"
  name     = "Terraform Test Update"
  login    = "ttu"
  password = "zyx987"
  is_admin = true
}
`
