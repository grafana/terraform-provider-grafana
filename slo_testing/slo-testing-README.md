# How to Test the SLO Terraform Provider - Hosted Grafana

# Installation
Install Terraform here - https://developer.hashicorp.com/terraform/tutorials/aws-get-started/install-cli#install-terraform. 

## HG Account Set Up
For members of the SLO Team, you should be able to use the `sloapp.grafana-dev.net` for testing or your own HG Instance.
Within Administration, generate a new Service Account Token.

Set the environment variable GRAFANA_AUTH to the value of your token `export GRAFANA_AUTH=<YOUR_SERVICE_ACCOUNT_TOKEN_HERE>`
Within the `.tf` files within `slo_testing/hg`, ensure that you set the `url` field to be the `url` of your HG Instance.

## Creating the TF Binary
1. Within the grafana-terraform-provider root directory, run `go build`. This creates a binary of the terraform-provider-grafana.
2. Within the grafana-terraform-provider root directory, create a file called `.terraformrc` with the following contents. Update the path to the path of your local `grafana-terraform-provider`. This ensures that it will use the local binary version of the terraform-provider-grafana.
```
provider_installation {
    dev_overrides {
       "grafana/grafana" = "/path/to/your/grafana/terraform-provider" # this path is the directory where the binary is built
   }
   # For all other providers, install them directly from their origin provider
   # registries as normal. If you omit this, Terraform will _only_ use
   # the dev_overrides block, and so no other providers will be available.
   direct {}
 }
```

### Types of Resources
Datasource - datasources are resources that are external to Terraform (i.e. not managed by Terraform state). When interacting with a Datasource, they can be used to READ information, and datasources can also be imported (i.e. converted) into Resources, which allows Terraform state to control them. 

Resources - these are resources that can be managed by Terraform state. This means that you CREATE, READ, UPDATE, DELETE them. 

## Testing Datasource - READ
Objective - we want to send a GET Request to the SLO Endpoint that returns a list of all SLOs, and we want to be able to READ that information and output it to the Terraform CLI.

1. Delete any `.terraform.lock.hcl` and `terraform.tfstate` and `terraform.tfstate.backup` files. Within the terraform-provider-grafana root directory, run `go build`. Set the GRAFANA_AUTH environment variable to your HG Grafana API Key, if not already done.
2. Change to the `slo_testing` directory `cd slo_testing`.
3. Within your SLO UI, create a SLO if one does not already exist. 
4. Within the `slo-datasource-read.tf` file, ensure the url is set to the url of your HG Instance. 
5. Comment out all the `.tf` files within the `slo_testing` folder, EXCEPT for the `slo-datasource-read.tf` file
6. Within the `slo_testing` directory, run the commands `terraform init` and `terraform apply`. 
7. You should see a list of all SLOs within your Terraform CLI.

## Testing Resource - CREATE
Objective - we want to be able to define a SLO Resource within Terraform state that should be created. Once the resource has successfully been created, we want to display the newly created SLO resource within the Terraform CLI. 

The `slo-resource-create.tf` file will create two SLOs. 

1. Change to the `slo_testing/hg` directory. 
2. Comment out all the `.tf` files within the `slo_testing` folder, EXCEPT for the `slo-resource-create.tf` file
3. Within the `slo_testing` directory, run the command `terraform apply`. 
4. Within your terminal, you should see the output of the a newly created SLO from within Terraform, and the same newly created SLO within the SLO UI. 

## Testing the UPDATE Method
Objective - we want to be able to update a SLO Resource created within Terraform. Once the resource has successfully been modified, we want to display the newly created SLO resource within the Terraform CLI. 

1. Do NOT delete any `.terraform.lock.hcl` and `terraform.tfstate` and `terraform.tfstate.backup` files. Ensure that this step is executed after testing the CREATE method. 
2. Change to the `slo_testing` directory. 
3. Comment out all the `.tf` files within the `slo_testing` folder, EXCEPT for the `slo-resource-create.tf` file
4. Modify any of the fields within the `slo-resource-create.tf` file - for example, you can change the `name` field to read `"Updated Terraform - Name Test"`.
5. Within the `slo_testing` directory, run the command `terraform apply`. This should update the resource specified in the terraform state file.
6. Check within the UI that the update was successful.

## Testing the DELETE/DESTROY Method
Objective - we want to be able to delete a SLO Resource that was created with Terraform. 

1. Do NOT delete any `.terraform.lock.hcl` and `terraform.tfstate` and `terraform.tfstate.backup` files. Ensure that this step is executed after testing the UPDATE method. 
2. Change to the `slo_testing` directory. 
3. To delete all Terraformed SLO resources, execute the command `terraform destroy`, and type `yes` in the terminal to confirm the delete
4. Any SLO Resources created with Terraform should be deleted.

### Testing the IMPORT Method
1. Change to the `slo_testing` directory. 
2. Comment out all the `.tf` files within the `slo_testing` folder, EXCEPT for the `slo-resource-import.tf` file
3. Create a SLO using the UI or Postman. Take note of the SLO's UUID
4. Within the Terraform CLI directly, execute the command: `terraform import grafana_slo.sample slo_UUID`
5. Now execute the command: `terraform state show grafana_slo.sample` - you should see the data from the imported Resource. 
6. To verify that this resource is now under Terraform control, execute the command `terraform destroy`. This should destroy the resource from within the Terraform CLI. 

### TBD ###
1. Once the GAPI Branch has been approved, remove the `replace` within `go.mod`
2. Remove `slo_testing` folder
