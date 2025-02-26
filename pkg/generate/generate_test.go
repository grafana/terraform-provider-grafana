package generate_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"text/template"

	"github.com/grafana/grafana-openapi-client-go/client/access_control"
	"github.com/grafana/grafana-openapi-client-go/client/service_accounts"
	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/grafana/terraform-provider-grafana/v3/internal/common"
	"github.com/grafana/terraform-provider-grafana/v3/internal/testutils"
	"github.com/grafana/terraform-provider-grafana/v3/pkg/generate"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/slices"
)

type generateTestCase struct {
	name           string
	config         string                         // Terraform configuration to apply
	stateCheck     func(s *terraform.State) error // Check the Terraform state after applying. Useful to extract computed attributes from state.
	generateConfig func(cfg *generate.Config)
	check          func(t *testing.T, tempDir string)                   // Check the generated files
	resultCheck    func(t *testing.T, result generate.GenerationResult) // Check the generation result

	tfInstallDir string // Directory where Terraform is installed. Used to avoid reinstalling it for each test case.
}

func (tc *generateTestCase) Run(t *testing.T) {
	stateCheck := func(s *terraform.State) error { return nil }
	if tc.stateCheck != nil {
		stateCheck = tc.stateCheck
	}
	t.Run(tc.name, func(t *testing.T) {
		resource.Test(t, resource.TestCase{
			ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
			Steps: []resource.TestStep{
				{
					Config: tc.config,
					Check: resource.ComposeTestCheckFunc(
						stateCheck,
						func(s *terraform.State) error {
							tempDir := t.TempDir()

							// Default configs, use `generateConfig` to override
							config := generate.Config{
								OutputDir:       tempDir,
								Clobber:         true,
								Format:          generate.OutputFormatHCL,
								ProviderVersion: "999.999.999", // Using the code from the current branch
								Grafana: &generate.GrafanaConfig{
									URL:  "http://localhost:3000",
									Auth: "admin:admin",
								},
								TerraformInstallConfig: generate.TerraformInstallConfig{
									InstallDir: tc.tfInstallDir,
									PluginDir:  pluginDir(t),
								},
							}
							if tc.generateConfig != nil {
								tc.generateConfig(&config)
							}

							result := generate.Generate(context.Background(), &config)
							if tc.resultCheck != nil {
								tc.resultCheck(t, result)
							} else {
								require.Len(t, result.Errors, 0, "expected no errors, got: %v", result.Errors)
							}

							if tc.check != nil {
								tc.check(t, tempDir)
							}

							return nil
						},
					),
				},
			},
		})
	})
}

func TestAccGenerate(t *testing.T) {
	testutils.CheckEnterpriseTestsEnabled(t, ">=10.0.0")

	// Install Terraform to a temporary directory to avoid reinstalling it for each test case.
	installDir := t.TempDir()

	var AlertRule1ID string
	cases := []generateTestCase{
		{
			name:   "dashboard",
			config: testutils.TestAccExample(t, "resources/grafana_dashboard/resource.tf"),
			check: func(t *testing.T, tempDir string) {
				assertFiles(t, tempDir, "testdata/generate/dashboard", []string{
					".terraform",
					".terraform.lock.hcl",
				})
			},
		},
		{
			name: "large-dashboards-exported-to-files",
			config: func() string {
				absPath, err := filepath.Abs("testdata/generate/dashboard-large/resources.tf")
				require.NoError(t, err)
				content, err := os.ReadFile(absPath)
				require.NoError(t, err)
				config := strings.ReplaceAll(string(content), "${path.module}", filepath.Dir(absPath))
				return config
			}(),
			generateConfig: func(cfg *generate.Config) {
				cfg.IncludeResources = []string{
					"grafana_dashboard.*",
					"grafana_folder.*",
				}
			},
			check: func(t *testing.T, tempDir string) {
				assertFiles(t, tempDir, "testdata/generate/dashboard-large", []string{
					".terraform",
					".terraform.lock.hcl",
				})
			},
		},
		{
			name:   "dashboard-json",
			config: testutils.TestAccExample(t, "resources/grafana_dashboard/resource.tf"),
			generateConfig: func(cfg *generate.Config) {
				cfg.Format = generate.OutputFormatJSON
			},
			check: func(t *testing.T, tempDir string) {
				assertFiles(t, tempDir, "testdata/generate/dashboard-json", []string{
					".terraform",
					".terraform.lock.hcl",
				})
			},
		},
		{
			name:   "dashboard-crossplane",
			config: testutils.TestAccExample(t, "resources/grafana_dashboard/resource.tf"),
			generateConfig: func(cfg *generate.Config) {
				cfg.Format = generate.OutputFormatCrossplane
			},
			check: func(t *testing.T, tempDir string) {
				assertFiles(t, tempDir, "testdata/generate/dashboard-crossplane", nil)
			},
		},
		{
			name:   "dashboard-filter-strict",
			config: testutils.TestAccExample(t, "resources/grafana_dashboard/resource.tf"),
			generateConfig: func(cfg *generate.Config) {
				cfg.IncludeResources = []string{"grafana_dashboard.my-dashboard-uid"}
			},
			check: func(t *testing.T, tempDir string) {
				assertFiles(t, tempDir, "testdata/generate/dashboard-filtered", []string{
					".terraform",
					".terraform.lock.hcl",
				})
			},
		},
		{
			name:   "dashboard-filter-wildcard-on-resource-type",
			config: testutils.TestAccExample(t, "resources/grafana_dashboard/resource.tf"),
			generateConfig: func(cfg *generate.Config) {
				cfg.IncludeResources = []string{"*.my-dashboard-uid"}
			},
			check: func(t *testing.T, tempDir string) {
				assertFiles(t, tempDir, "testdata/generate/dashboard-filtered", []string{
					".terraform",
					".terraform.lock.hcl",
				})
			},
		},
		{
			name:   "dashboard-filter-wildcard-on-resource-name",
			config: testutils.TestAccExample(t, "resources/grafana_dashboard/resource.tf"),
			generateConfig: func(cfg *generate.Config) {
				cfg.IncludeResources = []string{"grafana_dashboard.*"}
			},
			check: func(t *testing.T, tempDir string) {
				assertFiles(t, tempDir, "testdata/generate/dashboard-filtered", []string{
					".terraform",
					".terraform.lock.hcl",
				})
			},
		},
		{
			name:   "filter-all",
			config: testutils.TestAccExample(t, "resources/grafana_dashboard/resource.tf"),
			generateConfig: func(cfg *generate.Config) {
				cfg.IncludeResources = []string{"doesnot.exist"}
			},
			check: func(t *testing.T, tempDir string) {
				assertFiles(t, tempDir, "testdata/generate/empty", []string{
					".terraform",
					".terraform.lock.hcl",
				})
			},
		},
		{
			name:   "with-creds",
			config: testutils.TestAccExample(t, "resources/grafana_dashboard/resource.tf"),
			generateConfig: func(cfg *generate.Config) {
				cfg.IncludeResources = []string{"doesnot.exist"}
				cfg.OutputCredentials = true
			},
			check: func(t *testing.T, tempDir string) {
				assertFiles(t, tempDir, "testdata/generate/empty-with-creds", []string{
					".terraform",
					".terraform.lock.hcl",
				})
			},
		},
		{
			name: "alerting-in-org",
			config: func() string {
				content, err := os.ReadFile("testdata/generate/alerting-in-org.tf")
				require.NoError(t, err)
				return string(content)
			}(),
			generateConfig: func(cfg *generate.Config) {
				// The alerting rule group sometimes also creates an annotation.
				// It seems to be async so it makes the test flaky.
				// We can include only the resources we care about to avoid this.x
				cfg.IncludeResources = []string{
					"grafana_contact_point.*",
					"grafana_folder.*",
					"grafana_message_template.*",
					"grafana_mute_timing.*",
					"grafana_notification_policy.*",
					"grafana_organization.*",
					"grafana_rule_group.*",
				}
			},
			stateCheck: func(s *terraform.State) error {
				// Save the ID of the first alert rule for later use
				alertGroupResource, ok := s.RootModule().Resources["grafana_rule_group.my_alert_rule"]
				if !ok {
					return fmt.Errorf("expected resource 'grafana_rule_group.my_alert_rule' to be present")
				}
				AlertRule1ID = alertGroupResource.Primary.Attributes["rule.0.uid"]
				if AlertRule1ID == "" {
					return fmt.Errorf("expected 'rule.0.uid' to be present in 'grafana_rule_group.my_alert_rule' attributes")
				}
				return nil
			},
			check: func(t *testing.T, tempDir string) {
				templateAttrs := map[string]string{
					"AlertRule1ID": AlertRule1ID,
				}
				assertFilesWithTemplating(t, tempDir, "testdata/generate/alerting-in-org", []string{
					".terraform",
					".terraform.lock.hcl",
				}, templateAttrs)
			},
		},
		{
			name:   "fail-to-generate",
			config: " ",
			generateConfig: func(cfg *generate.Config) {
				cfg.Grafana.IsGrafanaCloudStack = true // Querying Grafana Cloud stuff will fail (this is a local instance)
			},
			resultCheck: func(t *testing.T, result generate.GenerationResult) {
				require.Greater(t, len(result.Success), 0, "expected successes, got: %+v", result)
				require.Greater(t, len(result.Errors), 1, "expected more than one error, got: %+v", result)
				gotCloudErrors := false
				for _, err := range result.Errors {
					resourceError, ok := err.(generate.ResourceError)
					require.True(t, ok, "expected ResourceError, got: %v", err)
					if strings.HasPrefix(resourceError.Resource.Name, "grafana_machine_learning") || strings.HasPrefix(resourceError.Resource.Name, "grafana_slo") {
						gotCloudErrors = true
						break
					}
				}
				require.True(t, gotCloudErrors, "expected errors related to Grafana Cloud resources, got: %v", result.Errors)
			},
		},
	}

	for _, tc := range cases {
		tc.tfInstallDir = installDir
		tc.Run(t)
	}
}

func TestAccGenerate_RestrictedPermissions(t *testing.T) {
	testutils.CheckEnterpriseTestsEnabled(t, ">=10.0.0")

	// Create SA with no permissions
	randString := acctest.RandString(10)
	client := testutils.Provider.Meta().(*common.Client).GrafanaAPI.Clone().WithOrgID(0)
	sa, err := client.ServiceAccounts.CreateServiceAccount(
		service_accounts.NewCreateServiceAccountParams().WithBody(&models.CreateServiceAccountForm{
			Name: "test-no-permissions-" + randString,
			Role: "None",
		},
		))
	require.NoError(t, err)
	t.Cleanup(func() {
		client.ServiceAccounts.DeleteServiceAccount(sa.Payload.ID)
	})

	saToken, err := client.ServiceAccounts.CreateToken(
		service_accounts.NewCreateTokenParams().WithBody(&models.AddServiceAccountTokenCommand{
			Name: "test-no-permissions-" + randString,
		},
		).WithServiceAccountID(sa.Payload.ID),
	)
	require.NoError(t, err)

	// Allow the SA to read dashboards
	if _, err := client.AccessControl.CreateRole(&models.CreateRoleForm{
		Name: randString,
		Permissions: []*models.Permission{
			{
				Action: "dashboards:read",
				Scope:  "dashboards:*",
			},
			{
				Action: "folders:read",
				Scope:  "folders:*",
			},
		},
		UID: randString,
	}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		client.AccessControl.DeleteRole(access_control.NewDeleteRoleParams().WithRoleUID(randString))
	})
	if _, err := client.AccessControl.SetUserRoles(sa.Payload.ID, &models.SetUserRolesCommand{
		RoleUids: []string{randString},
		Global:   false,
	}); err != nil {
		t.Fatal(err)
	}

	tc := generateTestCase{
		name:   "restricted-permissions",
		config: testutils.TestAccExample(t, "resources/grafana_dashboard/resource.tf"),
		generateConfig: func(cfg *generate.Config) {
			cfg.Grafana.Auth = saToken.Payload.Key
		},
		resultCheck: func(t *testing.T, result generate.GenerationResult) {
			assert.NotEmpty(t, result.Errors, "expected errors, got: %+v", result)
			for _, err := range result.Errors {
				// Check that all errors are non critical
				_, ok := err.(generate.NonCriticalError)
				assert.True(t, ok, "expected NonCriticalError, got: %v (Type: %T)", err, err)
			}
		},
		check: func(t *testing.T, tempDir string) {
			assertFiles(t, tempDir, "testdata/generate/dashboard-restricted-permissions", []string{
				".terraform",
				".terraform.lock.hcl",
			})
		},
	}

	tc.Run(t)
}

func TestAccGenerate_SMCheck(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	randomString := acctest.RandStringFromCharSet(10, acctest.CharSetAlpha)

	var smCheckID string
	tc := generateTestCase{
		name: "sm-check",
		config: testutils.TestAccExampleWithReplace(t, "resources/grafana_synthetic_monitoring_check/http_basic.tf", map[string]string{
			`"HTTP Defaults"`: strconv.Quote(randomString),
		}),
		stateCheck: func(s *terraform.State) error {
			checkResource, ok := s.RootModule().Resources["grafana_synthetic_monitoring_check.http"]
			if !ok {
				return fmt.Errorf("expected resource 'grafana_synthetic_monitoring_check.http' to be present")
			}
			smCheckID = checkResource.Primary.ID
			return nil
		},
		generateConfig: func(cfg *generate.Config) {
			cfg.Grafana = &generate.GrafanaConfig{
				URL:           os.Getenv("GRAFANA_URL"),
				Auth:          os.Getenv("GRAFANA_AUTH"),
				SMURL:         os.Getenv("GRAFANA_SM_URL"),
				SMAccessToken: os.Getenv("GRAFANA_SM_ACCESS_TOKEN"),
			}
			cfg.IncludeResources = []string{"grafana_synthetic_monitoring_check." + smCheckID}
		},
		check: func(t *testing.T, tempDir string) {
			templateAttrs := map[string]string{
				"ID":  smCheckID,
				"Job": randomString,
			}
			assertFilesWithTemplating(t, tempDir, "testdata/generate/sm-check", []string{
				".terraform",
				".terraform.lock.hcl",
			}, templateAttrs)
		},
	}

	tc.Run(t)
}

func TestAccGenerate_OnCall(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	randomString := acctest.RandStringFromCharSet(10, acctest.CharSetAlpha)

	tfConfig := fmt.Sprintf(`
	resource "grafana_oncall_integration" "test" {
	name = "%[1]s"
	type = "grafana"
	default_route {}
	}
	
	resource "grafana_oncall_escalation_chain" "test"{
	name = "%[1]s"
	}
	
	resource "grafana_oncall_escalation" "test" {
	escalation_chain_id = grafana_oncall_escalation_chain.test.id
	type = "wait"
	duration = "300"
	position = 0
	}
	
	resource "grafana_oncall_schedule" "test" {
	name = "%[1]s"
	type = "calendar"
	time_zone = "America/New_York"
	}
	`, randomString)

	var (
		oncallIntegrationID     string
		oncallEscalationChainID string
		oncallEscalationID      string
		oncallScheduleID        string
	)
	tc := generateTestCase{
		name:   "oncall",
		config: tfConfig,
		generateConfig: func(cfg *generate.Config) {
			cfg.Grafana = &generate.GrafanaConfig{
				URL:               os.Getenv("GRAFANA_URL"),
				Auth:              os.Getenv("GRAFANA_AUTH"),
				OnCallURL:         "https://oncall-prod-us-central-0.grafana.net/oncall",
				OnCallAccessToken: os.Getenv("GRAFANA_ONCALL_ACCESS_TOKEN"),
			}
			cfg.IncludeResources = []string{
				"grafana_oncall_integration." + oncallIntegrationID,
				"grafana_oncall_escalation_chain." + oncallEscalationChainID,
				"grafana_oncall_escalation." + oncallEscalationID,
				"grafana_oncall_schedule." + oncallScheduleID,
			}
		},
		stateCheck: func(s *terraform.State) error {
			integrationResource, ok := s.RootModule().Resources["grafana_oncall_integration.test"]
			if !ok {
				return fmt.Errorf("expected resource 'grafana_oncall_integration.test' to be present")
			}
			oncallIntegrationID = integrationResource.Primary.ID

			chainResource, ok := s.RootModule().Resources["grafana_oncall_escalation_chain.test"]
			if !ok {
				return fmt.Errorf("expected resource 'grafana_oncall_escalation_chain.test' to be present")
			}
			oncallEscalationChainID = chainResource.Primary.ID

			escalationResource, ok := s.RootModule().Resources["grafana_oncall_escalation.test"]
			if !ok {
				return fmt.Errorf("expected resource 'grafana_oncall_escalation.test' to be present")
			}
			oncallEscalationID = escalationResource.Primary.ID

			scheduleResource, ok := s.RootModule().Resources["grafana_oncall_schedule.test"]
			if !ok {
				return fmt.Errorf("expected resource 'grafana_oncall_schedule.test' to be present")
			}
			oncallScheduleID = scheduleResource.Primary.ID

			return nil
		},
		check: func(t *testing.T, tempDir string) {
			templateAttrs := map[string]string{
				"Name":              randomString,
				"IntegrationID":     oncallIntegrationID,
				"EscalationChainID": oncallEscalationChainID,
				"EscalationID":      oncallEscalationID,
				"ScheduleID":        oncallScheduleID,
			}
			assertFilesWithTemplating(t, tempDir, "testdata/generate/oncall-resources", []string{
				".terraform",
				".terraform.lock.hcl",
			}, templateAttrs)
		},
	}

	tc.Run(t)
}

// assertFiles checks that all files in the "expectedFilesDir" directory match the files in the "gotFilesDir" directory.
func assertFiles(t *testing.T, gotFilesDir, expectedFilesDir string, ignoreDirEntries []string) {
	t.Helper()
	assertFilesWithTemplating(t, gotFilesDir, expectedFilesDir, ignoreDirEntries, nil)
}

// assertFilesWithTemplating checks that all files in the "expectedFilesDir" directory match the files in the "gotFilesDir" directory.
func assertFilesWithTemplating(t *testing.T, gotFilesDir, expectedFilesDir string, ignoreDirEntries []string, attributes map[string]string) {
	t.Helper()

	if attributes != nil {
		expectedFilesDir = templateDir(t, expectedFilesDir, attributes)
	}

	assertFilesSubdir(t, gotFilesDir, expectedFilesDir, "", ignoreDirEntries)
}

func templateDir(t *testing.T, dir string, attributes map[string]string) string {
	t.Helper()

	templatedDir := t.TempDir()

	// Copy all dirs and files from the expected directory to the templated directory
	// Template all files that end with ".tmpl", renaming them to remove the ".tmpl" suffix
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relativePath, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}

		templatedPath := filepath.Join(templatedDir, relativePath)
		if info.IsDir() {
			return os.MkdirAll(templatedPath, 0755)
		}

		// Copy the file
		isTmpl := strings.HasSuffix(info.Name(), ".tmpl")
		templatedPath = strings.TrimSuffix(templatedPath, ".tmpl")
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if isTmpl {
			fileTmpl, err := template.New(path).Parse(string(content))
			if err != nil {
				return err
			}
			var templatedContent strings.Builder
			if err := fileTmpl.Execute(&templatedContent, attributes); err != nil {
				return err
			}
			content = []byte(templatedContent.String())
		}
		return os.WriteFile(templatedPath, content, 0600)
	})
	require.NoError(t, err)

	return templatedDir
}

func assertFilesSubdir(t *testing.T, gotFilesDir, expectedFilesDir, subdir string, ignoreDirEntries []string) {
	t.Helper()

	originalGotFilesDir := gotFilesDir
	originalExpectedFilesDir := expectedFilesDir
	if subdir != "" {
		gotFilesDir = filepath.Join(gotFilesDir, subdir)
		expectedFilesDir = filepath.Join(expectedFilesDir, subdir)
	}

	// Check that all generated files are expected (recursively)
	gotFiles, err := os.ReadDir(gotFilesDir)
	if err != nil {
		t.Logf("folder %s was not generated as expected", subdir)
		t.Fail()
		return
	}
	for _, gotFile := range gotFiles {
		relativeName := filepath.Join(subdir, gotFile.Name())
		if slices.Contains(ignoreDirEntries, relativeName) {
			continue
		}

		if gotFile.IsDir() {
			assertFilesSubdir(t, originalGotFilesDir, originalExpectedFilesDir, filepath.Join(subdir, gotFile.Name()), ignoreDirEntries)
			continue
		}

		if _, err := os.Stat(filepath.Join(expectedFilesDir, gotFile.Name())); err != nil {
			t.Logf("file %s was generated but wasn't expected", relativeName)
			t.Fail()
		}
	}

	// Verify the contents of the generated files (recursively)
	// All files in the expected directory should be present in the generated directory
	expectedFiles, err := os.ReadDir(expectedFilesDir)
	if err != nil {
		t.Logf("folder %s was generated but wasn't expected", subdir)
		t.Fail()
		return
	}
	for _, expectedFile := range expectedFiles {
		if expectedFile.IsDir() {
			assertFilesSubdir(t, originalGotFilesDir, originalExpectedFilesDir, filepath.Join(subdir, expectedFile.Name()), ignoreDirEntries)
			continue
		}
		expectedContent, err := os.ReadFile(filepath.Join(expectedFilesDir, expectedFile.Name()))
		require.NoError(t, err)

		gotContent, err := os.ReadFile(filepath.Join(gotFilesDir, expectedFile.Name()))
		require.NoError(t, err)

		assert.Equal(t, strings.TrimSpace(string(expectedContent)), strings.TrimSpace(string(gotContent)))
	}
}

func pluginDir(t *testing.T) string {
	t.Helper()

	repoRoot, err := filepath.Abs("../..")
	require.NoError(t, err)
	return filepath.Join(repoRoot, "testdata", "plugins")
}
