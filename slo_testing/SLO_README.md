-- Local SLO Dev Environment -- 
1. Start up the Local SLO Dev Environment
2. Send a POST Request with a sample request body 
POST Request: http://grafana.k3d.localhost:3000/api/plugins/grafana-slo-app/resources/v1/slo
{
   "name":"test slo name",
   "description":"test slo description",
   "service":"service",
   "labels": [{"key": "custom", "value": "value"}],
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
               "key":"annotsfastburnkey",
               "value":"annotsfastburnvalue"
            }
         ],
         "labels":[
            {
               "key":"type",
               "value":"slo"
            }
         ]
      },
      "slowBurn":{
         "annotations":[
            {
               "key":"annotsslowburnkey",
               "value":"annotsslowburnvalue"
            }
         ],
         "labels":[
            {
               "key":"type",
               "value":"slo"
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
3. Tests TBD.

-- Questions -- 
1. Why is our `Objectives` within the Slo struct a slice of Objectives? Shouldn't each Slo only have one Objective? 
type Slo struct {
	Uuid                  string        `json:"uuid"`
	Name                  string        `json:"name"`
	Description           string        `json:"description"`
	Service               string        `json:"service,omitempty"`
	Query                 Query         `json:"query"`
	Alerting              *Alerting     `json:"alerting,omitempty"`
	Labels                *[]Label      `json:"labels,omitempty"`
	Objectives            []Objective   `json:"objectives"`
	DrilldownDashboardUid string        `json:"dashboardUid,omitempty"`
	DrilldownDashboardRef *DashboardRef `json:"drillDownDashboardRef,omitempty"`
}

2. I cannot seem to add custom labels to the "Alerting" structure for some reason. Might be something wrong with the API. 
