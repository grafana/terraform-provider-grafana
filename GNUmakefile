GRAFANA_VERSION ?= latest
DOCKER_COMPOSE_ARGS ?= --pull always --force-recreate --detach --remove-orphans --wait --renew-anon-volumes

# Equivalence Makefile targets — see equivalence-tests/README.md Prerequisites & Commands.
EQUIV_CACHE_BIN := $(CURDIR)/.cache/bin
EQUIV_BIN ?= $(EQUIV_CACHE_BIN)/terraform-equivalence-testing

.PHONY: equivalence-test-ensure-bin \
	equivalence-test-update equivalence-test-diff equivalence-test-diff-local \
	equivalence-test-update-run equivalence-test-diff-run equivalence-test-diff-local-run

# terraform-equivalence-testing is installed lazily
$(EQUIV_BIN):
	@mkdir -p "$(EQUIV_CACHE_BIN)"
	GOBIN="$(EQUIV_CACHE_BIN)" go install github.com/hashicorp/terraform-equivalence-testing@v0.5.0

equivalence-test-ensure-bin:
ifeq ($(EQUIV_BIN),$(EQUIV_CACHE_BIN)/terraform-equivalence-testing)
	@$(MAKE) $(EQUIV_BIN)
else
	@test -x "$(EQUIV_BIN)" \
		|| { echo "EQUIV_BIN not found or not executable: $(EQUIV_BIN)"; exit 1; }
endif

equivalence-test-update-run: equivalence-test-ensure-bin
	env -u TF_CLI_CONFIG_FILE \
		GRAFANA_URL="$${GRAFANA_URL:-http://localhost:3000}" \
		GRAFANA_AUTH="$${GRAFANA_AUTH:-admin:admin}" \
		$(EQUIV_BIN) update \
		--goldens="$(CURDIR)/equivalence-tests/goldens" \
		--tests="$(CURDIR)/equivalence-tests/tests"

equivalence-test-diff-run: equivalence-test-ensure-bin
	env -u TF_CLI_CONFIG_FILE \
		GRAFANA_URL="$${GRAFANA_URL:-http://localhost:3000}" \
		GRAFANA_AUTH="$${GRAFANA_AUTH:-admin:admin}" \
		$(EQUIV_BIN) diff \
		--goldens="$(CURDIR)/equivalence-tests/goldens" \
		--tests="$(CURDIR)/equivalence-tests/tests"

# Build provider from this checkout and diff JSON vs checked-in goldens (uses dev_overrides;
# other providers still resolve via direct{}).
equivalence-test-diff-local-run: equivalence-test-ensure-bin
	REPO_ROOT="$(CURDIR)" \
		EQUIV_BIN="$(EQUIV_BIN)" \
		GRAFANA_URL="$${GRAFANA_URL:-http://localhost:3000}" \
		GRAFANA_AUTH="$${GRAFANA_AUTH:-admin:admin}" \
		bash "$(CURDIR)/equivalence-tests/diff-local.sh"

# Fresh Grafana via docker compose (same stack as testacc-oss-docker); no manual cleanup.
define equivalence-test-with-grafana
	REPO_ROOT="$(CURDIR)" \
		GRAFANA_VERSION="$(GRAFANA_VERSION)" \
		DOCKER_COMPOSE_ARGS="$(DOCKER_COMPOSE_ARGS)" \
		bash "$(CURDIR)/equivalence-tests/run-with-grafana.sh" $(1)
endef

equivalence-test-update:
	$(call equivalence-test-with-grafana,equivalence-test-update-run)

equivalence-test-diff:
	$(call equivalence-test-with-grafana,equivalence-test-diff-run)

equivalence-test-diff-local:
	$(call equivalence-test-with-grafana,equivalence-test-diff-local-run)

testacc:
	go build -o testdata/plugins/registry.terraform.io/grafana/grafana/999.999.999/$$(go env GOOS)_$$(go env GOARCH)/terraform-provider-grafana_v999.999.999_$$(go env GOOS)_$$(go env GOARCH) .
	TF_ACC=1 go test ./... -v $(TESTARGS) -timeout 120m

# Test OSS features
testacc-oss:
	GRAFANA_AUTH="$${GRAFANA_AUTH:-admin:admin}" TF_ACC_OSS=true make testacc

# Test Enterprise features
testacc-enterprise:
	GRAFANA_AUTH="$${GRAFANA_AUTH:-admin:admin}" TF_ACC_ENTERPRISE=true make testacc

# Test Cloud API features
testacc-cloud-api:
	TF_ACC_CLOUD_API=true make testacc

# Test Cloud instance features (ex: Machine Learning and Synthetic Monitoring)
testacc-cloud-instance:
	TF_ACC_CLOUD_INSTANCE=true make testacc

# Test a single package's Cloud instance acceptance tests.
# Used by the matrixed `cloudinstance` GitHub Actions job: each matrix shard
# provisions its own ephemeral stack via tools/teststack and then runs the
# tests in PKG against it.
#
# Usage:
#   PKG=./internal/resources/oncall/... make testacc-cloud-instance-pkg
#   PKG=./internal/resources/asserts/... TESTARGS='-run TestAccResourceStack' make testacc-cloud-instance-pkg
testacc-cloud-instance-pkg:
	@test -n "$(PKG)" || { echo "PKG is required, e.g. PKG=./internal/resources/oncall/..."; exit 2; }
	go build -o testdata/plugins/registry.terraform.io/grafana/grafana/999.999.999/$$(go env GOOS)_$$(go env GOARCH)/terraform-provider-grafana_v999.999.999_$$(go env GOOS)_$$(go env GOARCH) .
	TF_ACC=1 TF_ACC_CLOUD_INSTANCE=true go test $(PKG) -v $(TESTARGS) -timeout 25m

testacc-oss-docker:
	export GRAFANA_URL=http://0.0.0.0:3000 && \
	export GRAFANA_VERSION=$(GRAFANA_VERSION) && \
	docker compose up $(DOCKER_COMPOSE_ARGS) && \
	make testacc-oss && \
	docker compose down

testacc-enterprise-docker:
	export DOCKER_USER_UID="$(shell id -u)" && \
	export GRAFANA_URL=http://0.0.0.0:3000 && \
	export GRAFANA_VERSION=$(GRAFANA_VERSION) && \
	make -C testdata generate && \
	GRAFANA_IMAGE=grafana/grafana-enterprise docker compose up $(DOCKER_COMPOSE_ARGS) && \
	make testacc-enterprise && \
	docker compose down

testacc-tls-docker:
	export GRAFANA_URL=https://0.0.0.0:3001 && \
	export GRAFANA_VERSION=$(GRAFANA_VERSION) && \
	make -C testdata generate && \
	docker compose --profile tls up $(DOCKER_COMPOSE_ARGS) && \
	GRAFANA_TLS_KEY=$$(pwd)/testdata/client.key GRAFANA_TLS_CERT=$$(pwd)/testdata/client.crt GRAFANA_CA_CERT=$$(pwd)/testdata/ca.crt make testacc-oss && \
	docker compose --profile tls down

testacc-subpath-docker:
	export GRAFANA_SUBPATH=/grafana/ && \
	export GF_SERVER_SERVE_FROM_SUB_PATH=true && \
	export GRAFANA_URL=http://0.0.0.0:3001$${GRAFANA_SUBPATH} && \
	export GRAFANA_VERSION=$(GRAFANA_VERSION) && \
	docker compose --profile proxy up $(DOCKER_COMPOSE_ARGS) && \
	make testacc-oss && \
	docker compose --profile proxy down

integration-test:
	DOCKER_COMPOSE_ARGS="$(DOCKER_COMPOSE_ARGS)" GRAFANA_VERSION=$(GRAFANA_VERSION) ./testdata/integration/test.sh

release:
	@./scripts/release.sh

golangci-lint:
	docker run \
		--rm \
		--volume "$(shell pwd):/src" \
		--workdir "/src" \
		golangci/golangci-lint:v2.12.2 golangci-lint run ./... -v

docs:
	go generate ./...

codeowners:
	go run ./tools/codeowners > .github/CODEOWNERS

codeowners-check:
	go run ./tools/codeowners --check

linkcheck:
	docker run --rm --entrypoint sh -v "$$PWD:$$PWD" -w "$$PWD" python:3.11-alpine -c "pip3 install linkchecker && linkchecker --config .linkcheckerrc docs"

update-schema: ## Update provider schema only
	go build .
	./scripts/generate_schema.sh > provider_schema.json

generate-issue-template:
	go run scripts/generate_issue_template.go

generate-templates: ## Generate issue templates with schema update
	go run scripts/generate_issue_template.go --update-schema

generate-templates-quick: ## Generate issue templates (using cached schema)
	go run scripts/generate_issue_template.go
