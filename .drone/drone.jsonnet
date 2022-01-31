local grafanaVersions = ['8.3.3', '8.2.7', '8.1.8', '8.0.7', '7.5.12'];
local images = {
  go: 'golang:1.16',
  lint: 'golangci/golangci-lint',
  grafana(version): 'grafana/grafana:' + version,
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
          'go generate',
          'gitstatus="$(git status --porcelain)"',
          'if [ -n "$gitstatus" ]; then',
          '  echo "$gitstatus"',
          '  echo "docs are out of sync, run \\"go generate\\""',
          '  exit 1',
          'fi',
        ],
      },
    ]
  ),

  pipeline(
    'cloud tests',
    steps=[
      {
        name: 'tests',
        image: images.go,
        commands: [
          'make testacc-cloud',
        ],
        environment: {
          GRAFANA_URL: 'https://terraformprovidergrafana.grafana.net/',
          GRAFANA_AUTH: apiToken.fromSecret,
          GRAFANA_CLOUD_API_KEY: cloudApiKey.fromSecret,
          GRAFANA_SM_ACCESS_TOKEN: smToken.fromSecret,
          GRAFANA_ORG_ID: 1,
        },
      },
    ]
  ),

  cloudApiKey,
  apiToken,
  smToken,
] +
[
  pipeline(
    'oss tests: %s' % version,
    steps=[
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
          GRAFANA_ORG_ID: 1,
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
