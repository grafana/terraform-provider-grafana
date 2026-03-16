# Contributing

We welcome contributions. Here’s how to get changes merged.

## Submitting changes

1. **Fork the repo** and create a branch from `main`.
2. **Make your changes.** Follow the existing code style and patterns.
3. **Run tests** for the code you touch:
   - Unit tests: `go test ./...`
   - OSS acceptance tests (needs Grafana): see [README – Running Tests](README.md#running-tests), e.g. `make testacc-oss-docker`.
4. **Update generated docs** if you changed resource/datasource schema or examples: run `go generate ./...` (or `make docs`). CI will fail if `docs/` is out of sync.
5. **Open a pull request** against `main` with a clear description of the change.

For questions or discussion, use the [Grafana #terraform Slack channel](https://grafana.slack.com/archives/C017MUCFJUT). The Grafana Slack is public—anyone can join at [slack.grafana.com](https://slack.grafana.com/).
