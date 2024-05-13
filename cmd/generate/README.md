# Code Generation

Generate `terraform-provider-grafana` resources from your Grafana instance or Grafana Cloud account.

## Usage

```txt
NAME:
   terraform-provider-grafana-generate - Generate `terraform-provider-grafana` resources from your Grafana instance or Grafana Cloud account.

USAGE:
   terraform-provider-grafana-generate [options]

COMMANDS:
   help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --clobber, -c                       Delete all files in the output directory before generating resources (default: false) [$TFGEN_CLOBBER]
   --help, -h                          show help
   --output-dir value, -o value        Output directory for generated resources [$TFGEN_OUTPUT_DIR]
   --output-format value, -f value     Output format for generated resources. Supported formats are: [json hcl crossplane] (default: "hcl") [$TFGEN_OUTPUT_FORMAT]
   --terraform-provider-version value  Version of the Grafana provider to generate resources for. Defaults to the release version (same as the generator version). [$TFGEN_TERRAFORM_PROVIDER_VERSION]

   Grafana

   --grafana-auth value  Service account token or username:password for the Grafana instance [$TFGEN_GRAFANA_AUTH]
   --grafana-url value   URL of the Grafana instance to generate resources from [$TF_GEN_GRAFANA_URL]

   Grafana Cloud

   --cloud-access-policy-token value         Access policy token for Grafana Cloud [$TFGEN_CLOUD_ACCESS_POLICY_TOKEN]
   --cloud-create-stack-service-account      Create a service account for each Grafana Cloud stack, allowing generation and management of resources in that stack. (default: false) [$TFGEN_CLOUD_CREATE_STACK_SERVICE_ACCOUNT]
   --cloud-org value                         Organization ID or name for Grafana Cloud [$TFGEN_CLOUD_ORG]
   --cloud-stack-service-account-name value  Name of the service account to create for each Grafana Cloud stack. (default: "tfgen-management") [$TFGEN_CLOUD_STACK_SERVICE_ACCOUNT_NAME]
```

## Maturity

> _The code in this folder should be considered experimental. Documentation is only
available alongside the code. It comes with no support, but we are keen to receive
feedback on the product and suggestions on how to improve it, though we cannot commit
to resolution of any particular issue. No SLAs are available. It is not meant to be used
in production environments, and the risks are unknown/high._

Grafana Labs defines experimental features as follows:

> Projects and features in the Experimental stage are supported only by the Engineering
teams; on-call support is not available. Documentation is either limited or not provided
outside of code comments. No SLA is provided.
>
> Experimental projects or features are primarily intended for open source engineers who
want to participate in ensuring systems stability, and to gain consensus and approval
for open source governance projects.
>
> Projects and features in the Experimental phase are not meant to be used in production
environments, and the risks are unknown/high.
