## Development

Configure an access token for Backstage:

```console
export BACKSTAGE_TOKEN=<get access token defined on EngHub>
```

Set up a port-forward with kubectl or k9s:

```console
kubectl port-forward -n backstage service/backstage-ingress 8080
export BACKSTAGE_URL=http://localhost:8080
```

Get a GitHub token with the 'project' scope:

```console
gh auth login -s 'project'
export GITHUB_TOKEN=$(gh auth token)
```

