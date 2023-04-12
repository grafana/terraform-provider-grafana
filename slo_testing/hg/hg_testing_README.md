# How to Test the SLO Terraform Provider - Hosted Grafana

## Create your HG Account and Get the SLO Plugin Deployed
Generate a new Service Account Token, and set the environment variable GRAFANA_AUTH to the value of your token (or you can specify the `auth` field within the Terraform State file). 
Within the `.tf` files within `slo_testing/hg`, ensure that you set the `url` field to be the `url` of your HG Instance.

## Understanding Terraform Provider Code Flow
1. Within the terraform root directory, run `make install`. This command creates a binary of the Terraform Provider and moves it into the appropriate Terraform plugin directory, which allows for testing of a custom provider. 

   * Note: you may need to modify the `OS_ARCH=darwin_arm64` property within the Makefile to match your operating system

2. Within `provider.go`, we create two resources - a `grafana_slo_datasource` and a `grafana_slo_resource`. 

### Types of Resources
Datasource - datasources are resources that are external to Terraform (i.e. not managed by Terraform state). When interacting with a Datasource, they can be used to READ information, and datasources can also be imported (i.e. converted) into Resources, which allows Terraform state to control them. 

Resources - these are resources that can be managed by Terraform state. This means that you CREATE, READ, UPDATE, DELETE. If an IMPORT method is defined, you can also convert Datasources into Resources (i.e. this means that if you import a resource created by the UI (i.e. a Datasource) it can be converted into a Resrouce, which can be managed by Terraform).

## Testing Datasource - READ
`internal/resources/slo/data_source_slo.go`

This file defines a schema, that matches the response shape of a GET request to the SLO Endpoint. 
```
{
    "slos": [
        {
            "uuid": "bik1rpvkvbzxnfkutzmkh",
            "name": "test1",
            ...
        },
        {
            "uuid": "94pqcghz92hybc3iwircy",
            "name": "test2",
            ...
        },
    ]
}
```

Objective - we want to send a GET Request to the SLO Endpoint that returns a list of all SLOs, and we want to be able to READ that information and output it to the Terraform CLI.

1. Within the terraform-provider-grafana root directory, run `make install`.
2. Within your SLO UI, create a SLO. 
3. Set the GRAFANA_AUTH environment variable to your HG Grafana API Key
4. Within the `slo-datasource-read-hg.tf` file, set the url to be the url of your HG Instance. 
5. Ensure that Lines 236-238 within the `data_source_slo.go` are UNCOMMENTED. 
6. Comment out all the `.tf` files within the `slo_testing/hg` folder, EXCEPT for the `slo-datasource-read-hg.tf` file
7. Within the `slo_testing/hg` directory, run the commands `terraform init` and `terraform apply`. Ensure to remove the `.terraform.lock.hcl` and any `terraform.tfstate` files.
8. You should see a list of all SLOs within your Terraform CLI.

## Testing Resource - CREATE
Objective - we want to be able to define a SLO Resource within Terraform state that should be created. Once the resource has successfully been created, we want to display the newly created SLO resource within the Terraform CLI. 

The `slo-resource-create.tf` file will create two SLOs. 

Testing the CREATE Method
1. Delete any `.terraform.lock.hcl` and `terraform.tfstate` and `terraform.tfstate.backup` files. Within the terraform-provider-grafana root directory, run `make install`.
2. Change to the `slo_testing/hg` directory. 
3. Comment out all the `.tf` files within the `slo_testing/hg` folder, EXCEPT for the `slo-resource-create.tf` file
4. Within the `slo_testing/hg` directory, run the commands `terraform init` and `terraform apply`. 
5. Within your terminal, you should see the output of the two newly created SLO from within Terraform, and two newly created SLOs within the SLO UI.

## Testing the UPDATE Method
1. Delete any `.terraform.lock.hcl` and `terraform.tfstate` and `terraform.tfstate.backup` files. Within the terraform-provider-grafana root directory, run `make install`.
2. Change to the `slo_testing/hg` directory. 
3. Comment out all the `.tf` files within the `slo_testing/local` folder, EXCEPT for the `slo-resource-update.tf` file
4. Within the `slo_testing/hg` directory, run the commands `terraform init` and `terraform apply`. This creates the resource specified below in the terraform state file.
5. To ensure that the PUT endpoint works, modify any of the values within the resource below, and re-run `terraform apply`. 
6. Make a GET Request to the API Endpoint to ensure the resource was properly modified. 

## Testing the DELETE/DESTROY Method
Objective - we want to be able to delete a SLO Resource that was created with Terraform. 
After creating the two SLO resources from the CREATE Method, we will DELETE them. 

1. Delete any `.terraform.lock.hcl` and `terraform.tfstate` and `terraform.tfstate.backup` files
2. Within the `slo_testing/hg` directory, ensure the `slo-resource-create.tf` file is uncommented. Run the commands `terraform init` and execute `terraform apply`. This creates two new SLOs from the Terraform CLI.
3. Create a regular SLO using the UI. At this point, you should have 3 SLOs - 2 created from Terraform, and 1 created from the UI
4. To delete all Terraformed SLO resources, execute the command `terraform destroy`, and type `yes` in the terminal to confirm the delete
5. The two newly created Terraformed SLO Resources should be deleted, and you should still see the SLO that was created through the UI remaining.

### Testing the IMPORT Method
1. Within the terraform-provider-grafana root directory, run `make install`.
2. Change to the `slo_testing/hg` directory. 
3. Comment out all the `.tf` files within the `slo_testing/local` folder, EXCEPT for the `slo-resource-import.tf` file
4. Create a SLO using the UI or Postman. Take note of the SLO's UUID
5. Execute the command `terraform init`
6. Within the Terraform CLI directly, type in the command: `terraform import grafana_slo_resource.sample slo_UUID`
7. Now execute the command: `terraform state show grafana_slo_resource.sample` - you should see the data from the imported Resource. 
8. To verify that this resource is now under Terraform control, within the `slo-resource-import.tf` file, comment out lines 14-18. Then, within the CLI run `terraform destroy`. This should destroy the resource from within the Terraform CLI. 

### TBD ###
2. Figure out the Bug after Creating Terraform Resources (cannot go and Edit a SLO - why?).
2. Integrate into the existing Grafana Golang Client (https://github.com/grafana/grafana-api-golang-client)
3. Tests.