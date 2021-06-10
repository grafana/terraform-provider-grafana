local step(name, commands, image='golang:1.14') = {
  name: name,
  commands: commands,
  image: image,
};

local pipeline(name, steps=[]) = {
  kind: 'pipeline',
  type: 'docker',
  name: name,
  steps: [step('runner identification', ['echo $DRONE_RUNNER_NAME'], 'alpine')] + steps,
  trigger+: {
    ref+: [
      'refs/heads/main',
      'refs/pull/**',
      'refs/tags/v*.*.*',
    ],
  },
};

local releaseOnly = {
  when: {
    ref+: [
      'refs/heads/main',
      'refs/tags/v*.*.*',
    ],
  },
};

local prOnly = {
  when: {event: ['pull_request']},
};

[
  pipeline('build', [
    step('lint', ['make lint']),

    step('test', ['make test']),

    step('build', [
      'git fetch origin --tags',
      'git status --porcelain --untracked-files=no',
      'git diff --no-ext-diff --quiet', // fail if the workspace has modified files
      './scripts/version',
      'make build',
    ]),
  ]),
]
