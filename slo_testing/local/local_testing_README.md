# How to Test the SLO Terraform Provider - Locally

## Set Up Local Environment
1. Start up the Local SLO Dev Environment which should be available at http://localhost:3000/a/grafana-slo-app/home

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

1. Delete any `.terraform.lock.hcl` and `terraform.tfstate` and `terraform.tfstate.backup` files. Within the terraform-provider-grafana root directory, run `make install`. 
2. Create a SLO / Send a POST Request to the endpoint (see `slo_sample.json` for an example) 
3. Change to the `slo_testing/local` directory. 
4. Comment out all the `.tf` files within the `slo_testing/local` folder, EXCEPT for the `slo-datasource-read.tf` file
5. Within `data_source_slo.go` - you MUST comment out L233-235 - for some reason, there is a bug that does not work if the Authorization Header is set. 
6. Within the `slo_testing/local` directory, run the commands `terraform init` and `terraform apply`. Ensure to remove the `.terraform.lock.hcl` and any `terraform.tfstate` files.
7. You should see a list of all SLOs within the terminal that you've run terraform from. 

Elaine Questions - I'm open to modifying the shape of the Schema returned by Terraform on the read, so any thoughts here are welcome! Right now - I've just mirrored the structure we get back from the API. 

## Testing SLO Resource

### Testing the CREATE Method
Objective - we want to be able to define a SLO Resource within Terraform that should be created. Once the resource has successfully been created, we want to display the newly created SLO resource within the Terraform interface. 

The `slo-resource-create.tf` file will create two SLOs. 

1. Within `resource_slo.go` - you MUST comment out Lines 244-246, Lines 408-410, Line 473-475, and Lines 514-516, there is a bug that does not work if the Authorization Header is set. 
2. Delete any `.terraform.lock.hcl` and `terraform.tfstate` and `terraform.tfstate.backup` files. Within the terraform-provider-grafana root directory, run `make install`.
3. Change to the `slo_testing/local` directory. 
4. Comment out all the `.tf` files within the `slo_testing/local` folder, EXCEPT for the `slo-resource-create.tf` file
5. Within the `slo_testing` directory, run the commands `terraform init` and `terraform apply`. Ensure to remove the `.terraform.lock.hcl` and any `terraform.tfstate` files.
6. Within your terminal, you should see the output of the two newly created SLO from within Terraform, and two new SLOs within the SLO UI.

### Testing the UPDATE Method
1. Within `resource_slo.go` - you MUST comment out Lines 244-246, Lines 408-410, Line 473-475, and Lines 514-516, there is a bug that does not work if the Authorization Header is set. 
2. Within the terraform-provider-grafana root directory, run `make install`.
3. Change to the `slo_testing/local` directory. 
4. Comment out all the `.tf` files within the `slo_testing/local` folder, EXCEPT for the `slo-resource-update.tf` file
5. Run the command `terraform init`
6. Run the command `terraform apply`. This creates the resource specified below. 
7. To ensure that the PUT endpoint works, modify any of the values within the resource below, and re-run `terraform apply`. 

### Testing the DELETE Method
Objective - we want to be able to delete a SLO Resource that was created with Terraform. 
After creating the two SLO resources from the CREATE Method, we will DELETE them. 

Testing the DELETE Method / terraform destroy
1. Delete any `.terraform.lock.hcl` and `terraform.tfstate` and `terraform.tfstate.backup` files
2. Within the `slo_testing` directory, run the commands `terraform init`. Keep the `slo-resource-create.tf` file open, and execute `terraform apply`. This creates two new SLOs from the Terraform CLI.
3. Create a regular SLO using the UI. At this point, you should have 3 SLOs - 2 created from Terraform, and 1 created from the UI
4. To delete all Terraformed SLO resources, execute the command `terraform destroy`, and type `yes` in the terminal to confirm the delete
5. The two newly created Terraformed SLO Resources should be deleted, and you should still have the SLO that was created through the UI remaining.

### Testing the IMPORT Method
1. Within the terraform-provider-grafana root directory, run `make install`.
2. Change to the `slo_testing/local` directory. 
3. Comment out all the `.tf` files within the `slo_testing/local` folder, EXCEPT for the `slo-resource-import.tf` file
4. Create a SLO using the UI or Postman. Take note of the SLO's UUID
5. Execute the command `terraform init`
6. Within the Terraform CLI directly, type in the command: `terraform import grafana_slo_resource.sample slo_UUID`
7. Now execute the command: `terraform state show grafana_slo_resource.sample` - you should see the data from the imported Resource. 