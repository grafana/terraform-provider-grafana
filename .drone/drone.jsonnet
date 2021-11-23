local grafana = 'grafana/grafana:8.0.3';
local build = 'grafana/build-container:1.4.7';

local secret(name, vaultPath, vaultKey) = {
  kind: 'secret',
  name: name,
  get: {
    path: vaultPath,
    name: vaultKey,
  },

  fromSecret:: { from_secret: name },
};
local apiToken = secret('grafana-api-token', 'infra/data/ci/terraform-provider-grafana/cloud', 'api-key');
local smToken = secret('grafana-sm-token', 'infra/data/ci/terraform-provider-grafana/cloud', 'sm-access-token');

local pipeline(name, steps, services=[]) = {
  kind: 'pipeline',
  type: 'docker',
  name: name,
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
        image: build,
        commands: [
          'golangci-lint --version',
          'golangci-lint run ./...',
        ],
      },
    ]
  ),

  pipeline(
    'oss tests',
    steps=[
      {
        name: 'tests',
        image: build,
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
        image: grafana,
        environment: {
          // Prevents error="database is locked"
          GF_DATABASE_URL: 'sqlite3:///var/lib/grafana/grafana.db?cache=private&mode=rwc&_journal_mode=WAL',
        },
      },
    ],
  ),

  pipeline(
    'cloud tests',
    steps=[
      {
        name: 'tests',
        image: build,
        commands: ['make testacc-cloud'],
        environment: {
          GRAFANA_URL: 'https://terraformprovidergrafana.grafana.net/',
          GRAFANA_AUTH: apiToken.fromSecret,
          GRAFANA_SM_ACCESS_TOKEN: smToken.fromSecret,
          GRAFANA_ORG_ID: 1,
        },
      },
    ]
  ),

  apiToken,
  smToken,
]
