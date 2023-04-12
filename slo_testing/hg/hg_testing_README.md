# How to Test the SLO Terraform Provider - Hosted Grafana

## Create your HG Account and Get the SLO Plugin Deployed

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

### TBD ###
1. Testing HG for the SLO Resources for Create, Read, Update, and Import. 
2. Client Wrapper - I am currently just sending requests by creating an HTTP Client, ideally we should create a Go Client wrapper around our API. This will be done and refactored at a later point in time. 
3. Tests.