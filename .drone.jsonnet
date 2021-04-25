local golang = 'golang:1.16';
local grafana = 'grafana/grafana:7.4.2';

// We'd like the same pipeline for testing pull requests as we do for building
// master. The only difference is their names and triggers.
local pipeline(name, trigger) = {
  kind: 'pipeline',
  type: 'docker',
  name: name,
  platform: {
    os: 'linux',
    arch: 'amd64',
  },
  steps: [
    {
      name: 'tests',
      image: golang,
      commands: [
        'make testacc',
      ],
      environment: {
        GRAFANA_URL: 'http://grafana:3000',
        GRAFANA_AUTH: 'admin:admin',
        GRAFANA_ORG_ID: 1,
      },
    },
  ],
  services: [
    {
      name: 'grafana',
      image: grafana,
    },
  ],
  trigger: trigger,
};

[
  pipeline('test-pr', {
    event: ['pull_request'],
  }),
  pipeline('build-master', {
    branch: ['master'],
    event: ['push'],
  }),
]
