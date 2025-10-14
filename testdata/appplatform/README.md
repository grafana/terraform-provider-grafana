# Test Setup for Grafana App Platform resources

This repo contains the test setup for the new app platform based Grafana Terraform Provider resources. Currently supports dashboards and playlists.

The repo defines a couple of folders and dashboards and a playlist (folders are still using the existing folders API):

* `folders.tf` defines the folders
* `dashboards.tf` defines the dashboards
* `playlists.tf` defines the playlist

In order to use this repo you need to make sure you are using the terraform provider from the feature branch `feat/appplatform-dashboards`, to do that you'll need to clone and build the provider:
```console
# Clone the repo
git clone git@github.com:grafana/terraform-provider-grafana.git
cd terraform-provider-grafana

# Switch to the feature branch
git checkout feat/appplatform-dashboards

# Build the binary
go build

# Override your Terraform config to use custom-built Grafana Terraform provider binary
# (see docs at https://github.com/grafana/terraform-provider-grafana?tab=readme-ov-file#local-development-with-grafana)
cat <<EOF >> "${HOME}/.terraformrc"
provider_installation {
   dev_overrides {
      "grafana/grafana" = "$(pwd)" # this path is the directory where the binary is built
  }
  # For all other providers, install them directly from their origin provider
  # registries as normal. If you omit this, Terraform will _only_ use
  # the dev_overrides block, and so no other providers will be available.
  direct {}
}
EOF
```

You'll also need to run Grafana with necessary feature flags:
```console
# Clone the repo
git clone git@github.com:grafana/grafana.git
cd grafana

# Create a custom config
cat <<EOF >> "conf/custom.ini"
app_mode = development
target = all

[server]
protocol = https

[feature_toggles]
grafanaAPIServer = true
kubernetesAggregator = true
grafanaAPIServerEnsureKubectlAccess = true
unifiedStorage = true
idForwarding = true
kubernetesPlaylists = true
kubernetesFeatureToggles = true
kubernetesDashboards = true
kubernetesClientDashboardsFolders = true
kubernetesRestore = true
kubernetesSnapshots = true
k8SFolderCounts = true
k8SFolderMove = true

[grafana-apiserver]
storage_type = unified
EOF

# Run the server
make build-go run-go
```

Next you'll need to create a service account with permissions to manage folders, dashboards and playlists (you can create an admin one as well) and generate a service token for it.

Finally, you can set up this repo:
```console
# Clone the test repo
git clone git@github.com:radiohead/terraform-provider-grafana-test.git
cd terraform-provider-grafana-test

# Configure Terraform
cat <<EOF >> "terraform.tfvars"
grafana_url  = "https://localhost:3000/"
grafana_auth = "<your-service-account-token-here>"
EOF

# Initialize
terraform init

# Plan
terraform plan

# Apply
terraform apply -auto-approve
```
