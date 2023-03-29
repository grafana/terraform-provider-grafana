-- Local SLO Dev Environment -- 
1. Start up the Local SLO Dev Environment
2. Send a POST Request with a sample request body 
POST Request: http://grafana.k3d.localhost:3000/api/plugins/grafana-slo-app/resources/v1/slo
{
   "name":"test name",
   "description":"test description",
   "service":"service",
   "labels": [{"key": "name", "value": "testslo"}],
   "objectives":[
      {
         "value":0.995,
         "window":"30d"
      }
   ],
   "query":{
      "freeFormQuery":"sum(rate(apiserver_request_total{code!=\"500\"}[$__rate_interval])) / sum(rate(apiserver_request_total[$__rate_interval]))"
   },
   "alerting":{
      "fastBurn":{
         "annotations":[
            {
               "key":"annots-key",
               "value":"annots-Fast Burn"
            },
            {
               "key":"Description",
               "value":"Fast Burn Description"
            }
         ],
         "labels":[
            {
               "key":"Type",
               "value":"SLO"
            }
         ]
      },
      "slowBurn":{
         "annotations":[
            {
               "key":"Name",
               "value":"Slow Burn"
            },
            {
               "key":"Description",
               "value":"Slow Burn Description"
            }
         ],
         "labels":[
            {
               "key":"Type",
               "value":"SLO"
            }
         ]
      }
   }
}

-- Terraform Provider -- 
1. Within the root directory of the terraform-provider-grafana, run `make install`. This creates a Grafana Terraform Provider
2. Switch to the slo_testing directory `cd slo_testing`
3. Run the command `terraform init`
4. Run the command `terraform apply`. This will execute the `slo-ds-read.tf` state file, which currently just sends a GET request to the http://localhost:3000/api/plugins/grafana-slo-app/resources/v1/slo endpoint, and returns it in Terraform. 
5. Ensure to delete the `.terraform.lock.hcl` file that exists before rebuilding the terraform provider. 

-- TBD -- 
1. Currently - I am just doing this in local dev, I need to set this up and test it with a HG Account
2. Client Wrapper - I am currently just sending requests by creating an HTTP Client, ideally we should create a Go Client wrapper around our API. This will be done and refactored at a later point in time. 
3. CREATE, UPDATE, and DELETE methods TBD. 
4. Tests TBD.
5. Authentication - currently it appears that our API endpoint isn't protected at all - a user doesn't need to send a Grafana Token or anything in order to access our API Endpoint. Did we want to change this? 