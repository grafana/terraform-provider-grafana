local grafanaVersions = ['10.1.2', '10.0.6', '9.5.10', '8.5.27'];
local images = {
  go: 'golang:1.21',
  terraform: 'hashicorp/terraform',
  grafana(version): 'grafana/grafana:' + version,
  grafanaEnterprise(version): 'grafana/grafana-enterprise:' + version,
};

local workspace = '/drone/terraform-provider-grafana';
local terraformPath = workspace + '/terraform';
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

  // Grafana Cloud Instance test secrets
  cloudInstanceUrl: secret('grafana-cloud-instance-url', 'infra/data/ci/terraform-provider-grafana/cloud', 'cloud-instance-url'),
  apiToken: secret('grafana-api-token', 'infra/data/ci/terraform-provider-grafana/cloud', 'api-key'),
  smToken: secret('grafana-sm-token', 'infra/data/ci/terraform-provider-grafana/cloud', 'sm-access-token'),
  onCallToken: secret('grafana-oncall-token', 'infra/data/ci/terraform-provider-grafana/cloud', 'oncall-access-token'),

  // Grafana Enterprise
  enterpriseLicense: secret('grafana-enterprise-license', 'infra/data/ci/terraform-provider-grafana/enterprise', 'license.jwt'),
};

local pipeline(name, steps, services=[]) = {
  kind: 'pipeline',
  type: 'docker',
  name: name,
  workspace: {
    path: workspace,
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

local localTestPipeline(
  version,
  name='oss tests: %s' % version,
  makeTarget='testacc-oss',
  providerEnvMixin={},
  grafanaEnvMixin={},
  grafanaImage=images.grafana,
      ) =
  pipeline(
    name,
    steps=[
      installTerraformStep,
      {
        name: 'tests',
        image: images.go,
        commands: [
          'sleep 5',  // https://docs.drone.io/pipeline/docker/syntax/services/#initialization
          'make %s' % makeTarget,
        ],
        environment: {
          GRAFANA_URL: 'http://grafana:3000',
          GRAFANA_AUTH: 'admin:admin',
          GRAFANA_VERSION: version,
          TF_ACC_TERRAFORM_PATH: terraformPath,
        } + providerEnvMixin,
      },
    ],
    services=[
      {
        name: 'grafana',
        image: grafanaImage(version),
        environment: {
          // Prevents error="database is locked"
          GF_SERVER_ROOT_URL: 'http://grafana:3000',
          GF_DATABASE_URL: 'sqlite3:///var/lib/grafana/grafana.db?cache=private&mode=rwc&_journal_mode=WAL',
        } + grafanaEnvMixin,
      },
    ],
  );

[
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
          GRAFANA_ONCALL_ACCESS_TOKEN: fromSecret(secrets.onCallToken),
          TF_ACC_TERRAFORM_PATH: terraformPath,
        },
      },
    ]
  )
  + withConcurrencyLimit(1),

  // Grafana Enterprise tests
  localTestPipeline(
    grafanaVersions[0],
    name='enterprise tests',
    makeTarget='testacc-enterprise',
    grafanaEnvMixin={ GF_ENTERPRISE_LICENSE_TEXT: fromSecret(secrets.enterpriseLicense) },
    grafanaImage=images.grafanaEnterprise
  ),

  // Grafana OSS tests behind a TLS proxy tests
  // This is the equivalent of `make testacc-docker-tls`
  local certPath = workspace + '/testdata';
  localTestPipeline(
    grafanaVersions[0],
    name='tls proxy tests',
    providerEnvMixin={
      GRAFANA_URL: 'https://mtls-proxy:3001',
      GRAFANA_TLS_KEY: '%s/client.key' % certPath,
      GRAFANA_TLS_CERT: '%s/client.crt' % certPath,
      GRAFANA_CA_CERT: '%s/ca.crt' % certPath,
      TESTARGS: '-run ".*_basic"',  // Tests are slower behind the proxy, let's just run the basic (smaller) ones
    }
  ) + {
    steps: [
      {
        name: 'generate certs',
        image: images.go,
        commands: [
          'cd %s && go run . && ls -lah' % certPath,
        ],
        depends_on: ['clone'],
      },
      {
        name: 'mtls-proxy',
        image: 'squareup/ghostunnel:v1.5.2',
        detach: true,
        command: [
          'server',
          '--listen=0.0.0.0:3001',
          '--target=grafana:3000',
          '--unsafe-target',
          '--key=%s/grafana.key' % certPath,
          '--cert=%s/grafana.crt' % certPath,
          '--cacert=%s/ca.crt' % certPath,
          '--allow-cn=client',
        ],
        depends_on: ['generate certs'],
      },
    ] + std.map(function(s) s { depends_on: ['generate certs'] }, super.steps),
  },
]
+ [localTestPipeline(version) for version in grafanaVersions]
+ std.objectValuesAll(secrets)
