local grafanaVersions = ['9.4.3', '9.3.8', '9.2.13', '8.5.21', '7.5.17'];
local images = {
  go: 'golang:1.18',
  python: 'python:3.9-alpine',
  lint: 'golangci/golangci-lint:v1.49',
  terraform: 'hashicorp/terraform',
  grafana(version): 'grafana/grafana:' + version,
};

local terraformPath = '/drone/terraform-provider-grafana/terraform';
local installTerraformStep = {
  name: 'download-terraform',
  image: images.terraform,
  commands: [
    'cp /bin/terraform ' + terraformPath,
    'chmod a+x ' + terraformPath,
  ],
};

local secret(name, vaultPath, vaultKey) = {
  kind: 'secret',
  name: name,
  get: {
    path: vaultPath,
    name: vaultKey,
  },
};

local fromSecret(secret) = {
  from_secret: secret.name,
};

local secrets = {
  // Grafana Cloud API test secrets
  cloudOrg: secret('grafana-cloud-org', 'infra/data/ci/terraform-provider-grafana/cloud', 'cloud-org'),
  cloudApiKey: secret('grafana-cloud-api-key', 'infra/data/ci/terraform-provider-grafana/cloud', 'cloud-api-key'),

  // Grafana Enterprise test secrets (Instance running in Grafana Cloud)
  cloudInstanceUrl: secret('grafana-cloud-instance-url', 'infra/data/ci/terraform-provider-grafana/cloud', 'cloud-instance-url'),
  apiToken: secret('grafana-api-token', 'infra/data/ci/terraform-provider-grafana/cloud', 'api-key'),
  smToken: secret('grafana-sm-token', 'infra/data/ci/terraform-provider-grafana/cloud', 'sm-access-token'),
  onCallToken: secret('grafana-oncall-token', 'infra/data/ci/terraform-provider-grafana/cloud', 'oncall-access-token'),
};

local pipeline(name, steps, services=[]) = {
  kind: 'pipeline',
  type: 'docker',
  name: name,
  workspace: {
    path: '/drone/terraform-provider-grafana',
  },
  platform: {
    os: 'linux',
    arch: 'amd64',
  },
  steps: steps,
  services: services,
  trigger: {
    branch: ['master'],
    event: ['pull_request', 'push'],
  },
};

local withConcurrencyLimit(limit) = {
  concurrency: { limit: limit },
};

local onPromoteTrigger = {
  trigger: {
    event: ['promote'],
  },
};

[
  pipeline(
    'lint', steps=[
      {
        name: 'lint',
        image: images.lint,
        commands: [
          'golangci-lint --version',
          'golangci-lint run ./...',
        ],
      },
      {
        name: 'terraform-fmt',
        image: images.terraform,
        commands: [
          |||
            terraform fmt -recursive -check || (echo "Terraform files aren't formatted. Run 'terraform fmt -recursive && go generate'"; exit 1;)
          |||,
        ],
      },
    ]
  ),

  pipeline(
    'docs', steps=[
      {
        name: 'check for drift',
        image: images.go,
        commands: [
          'apt update && apt install -y jq',
          'go generate',
          'gitstatus="$(git status --porcelain)"',
          'if [ -n "$gitstatus" ]; then',
          '  echo "$gitstatus"',
          '  echo "docs are out of sync, run \\"go generate\\""',
          '  exit 1',
          'fi',
        ],
      },
      {
        name: 'check for broken links',
        image: images.python,
        commands: [
          'pip3 install linkchecker',
          'linkchecker --config ./.linkcheckerrc docs/',
        ],
      },
    ]
  ),

  pipeline(
    'unit tests',
    steps=[
      installTerraformStep,
      {
        name: 'tests',
        image: images.go,
        commands: [
          'go test ./...',
        ],
        environment: {
          TF_ACC_TERRAFORM_PATH: terraformPath,
        },
      },
    ]
  ),

  pipeline(
    'cloud api tests',
    steps=[
      installTerraformStep,
      {
        name: 'tests',
        image: images.go,
        commands: [
          'make testacc-cloud-api',
        ],
        environment: {
          GRAFANA_CLOUD_API_KEY: fromSecret(secrets.cloudApiKey),
          GRAFANA_CLOUD_ORG: fromSecret(secrets.cloudOrg),
          TF_ACC_TERRAFORM_PATH: terraformPath,
        },
      },
    ]
  )
  + withConcurrencyLimit(1)
  + onPromoteTrigger,

  pipeline(
    'cloud instance tests',
    steps=[
      installTerraformStep,
      {
        name: 'wait for instance',
        image: images.go,
        commands: ['.drone/wait-for-instance.sh $${GRAFANA_URL}'],
        environment: {
          GRAFANA_URL: fromSecret(secrets.cloudInstanceUrl),
        },
      },
      {
        name: 'tests',
        image: images.go,
        commands: ['make testacc-cloud-instance'],
        environment: {
          GRAFANA_URL: fromSecret(secrets.cloudInstanceUrl),
          GRAFANA_AUTH: fromSecret(secrets.apiToken),
          GRAFANA_SM_ACCESS_TOKEN: fromSecret(secrets.smToken),
          GRAFANA_ORG_ID: 1,
          GRAFANA_ONCALL_ACCESS_TOKEN: fromSecret(secrets.onCallToken),
          TF_ACC_TERRAFORM_PATH: terraformPath,
        },
      },
    ]
  )
  + withConcurrencyLimit(1),
]
+ [
  pipeline(
    'oss tests: %s' % version,
    steps=[
      installTerraformStep,
      {
        name: 'tests',
        image: images.go,
        commands: [
          'sleep 5',  // https://docs.drone.io/pipeline/docker/syntax/services/#initialization
          'make testacc-oss',
        ],
        environment: {
          GRAFANA_URL: 'http://grafana:3000',
          GRAFANA_AUTH: 'admin:admin',
          GRAFANA_VERSION: version,
          GRAFANA_ORG_ID: 1,
          TF_ACC_TERRAFORM_PATH: terraformPath,
        },
      },
    ],
    services=[
      {
        name: 'grafana',
        image: images.grafana(version),
        environment: {
          // Prevents error="database is locked"
          GF_DATABASE_URL: 'sqlite3:///var/lib/grafana/grafana.db?cache=private&mode=rwc&_journal_mode=WAL',
        },
      },
    ],
  )
  for version in grafanaVersions
]
+ std.objectValuesAll(secrets)
