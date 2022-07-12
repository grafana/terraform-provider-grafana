local grafanaVersions = std.parseJson(std.extVar('grafanaVersions'));
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

  fromSecret:: { from_secret: name },
};
local cloudApiKey = secret('grafana-cloud-api-key', 'infra/data/ci/terraform-provider-grafana/cloud', 'cloud-api-key');
local apiToken = secret('grafana-api-token', 'infra/data/ci/terraform-provider-grafana/cloud', 'api-key');
local smToken = secret('grafana-sm-token', 'infra/data/ci/terraform-provider-grafana/cloud', 'sm-access-token');
local onCallToken = secret('grafana-oncall-token', 'infra/data/ci/terraform-provider-grafana/cloud', 'oncall-access-token');

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
          GRAFANA_CLOUD_API_KEY: cloudApiKey.fromSecret,
          GRAFANA_CLOUD_ORG: 'terraformprovidergrafana',
          TF_ACC_TERRAFORM_PATH: terraformPath,
        },
      },
    ]
  ) + {
    concurrency: { limit: 1 },
  },

  local cloud_instance_url = 'https://terraformprovidergrafana.grafana.net/';
  pipeline(
    'cloud instance tests',
    steps=[
      installTerraformStep,
      {
        name: 'wait for instance',
        image: images.go,
        commands: ['.drone/wait-for-instance.sh ' + cloud_instance_url],
      },
      {
        name: 'tests',
        image: images.go,
        commands: ['make testacc-cloud-instance'],
        environment: {
          GRAFANA_URL: cloud_instance_url,
          GRAFANA_AUTH: apiToken.fromSecret,
          GRAFANA_SM_ACCESS_TOKEN: smToken.fromSecret,
          GRAFANA_ORG_ID: 1,
          GRAFANA_ONCALL_ACCESS_TOKEN: onCallToken.fromSecret,
          TF_ACC_TERRAFORM_PATH: terraformPath,
        },
      },
    ]
  ) + {
    concurrency: { limit: 1 },
  },

  cloudApiKey,
  apiToken,
  smToken,
  onCallToken,
] +
[
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
