package generate_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

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

func TestAccGenerate(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping long test")
	}
	testutils.CheckOSSTestsEnabled(t)

	// Install Terraform to a temporary directory to avoid reinstalling it for each test case.
	installDir := t.TempDir()

	cases := []struct {
		name           string
		config         string
		generateConfig func(cfg *generate.Config)
		check          func(t *testing.T, tempDir string)
		resultCheck    func(t *testing.T, result generate.GenerationResult)
	}{
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
				cfg.IncludeResources = []string{"grafana_dashboard._1_my-dashboard-uid"}
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
				cfg.IncludeResources = []string{"*._1_my-dashboard-uid"}
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
			check: func(t *testing.T, tempDir string) {
				assertFiles(t, tempDir, "testdata/generate/alerting-in-org", []string{
					".terraform",
					".terraform.lock.hcl",
				})
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
		t.Run(tc.name, func(t *testing.T) {
			resource.Test(t, resource.TestCase{
				ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: tc.config,
						Check: func(s *terraform.State) error {
							tempDir := t.TempDir()
							config := generate.Config{
								OutputDir:       tempDir,
								Clobber:         true,
								Format:          generate.OutputFormatHCL,
								ProviderVersion: "v3.0.0",
								Grafana: &generate.GrafanaConfig{
									URL:  "http://localhost:3000",
									Auth: "admin:admin",
								},
								TerraformInstallConfig: generate.TerraformInstallConfig{
									InstallDir: installDir,
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
					},
				},
			})
		})
	}
}

func TestAccGenerate_RestrictedPermissions(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping long test")
	}
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

	resource.Test(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testutils.TestAccExample(t, "resources/grafana_dashboard/resource.tf"),
				Check: func(s *terraform.State) error {
					tempDir := t.TempDir()
					config := generate.Config{
						OutputDir:       tempDir,
						Clobber:         true,
						Format:          generate.OutputFormatHCL,
						ProviderVersion: "v3.0.0",
						Grafana: &generate.GrafanaConfig{
							URL:  "http://localhost:3000",
							Auth: saToken.Payload.Key,
						},
					}

					result := generate.Generate(context.Background(), &config)
					assert.NotEmpty(t, result.Errors, "expected errors, got: %+v", result)
					for _, err := range result.Errors {
						// Check that all errors are non critical
						_, ok := err.(generate.NonCriticalError)
						assert.True(t, ok, "expected NonCriticalError, got: %v (Type: %T)", err, err)
					}

					assertFiles(t, tempDir, "testdata/generate/dashboard-restricted-permissions", []string{
						".terraform",
						".terraform.lock.hcl",
					})

					return nil
				},
			},
		},
	})
}

// assertFiles checks that all files in the "expectedFilesDir" directory match the files in the "gotFilesDir" directory.
func assertFiles(t *testing.T, gotFilesDir, expectedFilesDir string, ignoreDirEntries []string) {
	t.Helper()
	assertFilesSubdir(t, gotFilesDir, expectedFilesDir, "", ignoreDirEntries)
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
