TEST?=$$(go list ./... |grep -v 'vendor')
GOFMT_FILES?=$$(find . -name '*.go' |grep -v vendor)
WEBSITE_REPO=github.com/hashicorp/terraform-website
PKG_NAME=grafana
GRAFANA_VERSION ?= "latest"

default: build

build: fmtcheck
	go install

test: fmtcheck
	go test $(TEST) $(TESTARGS) -timeout=30s -parallel=4

testacc: fmtcheck
	TF_ACC=1 go test $(TEST) -v $(TESTARGS) -timeout 120m

test-serve: fmtcheck
	@docker pull "grafana/grafana:$(GRAFANA_VERSION)"
	docker run -p 127.0.0.1:3000:3000 "grafana/grafana:$(GRAFANA_VERSION)"

test-serve-tls: fmtcheck
	GRAFANA_VERSION=$(GRAFANA_VERSION) \
		docker-compose up --abort-on-container-exit --renew-anon-volumes

vet:
	@echo "go vet ."
	@go vet $$(go list ./... | grep -v vendor/) ; if [ $$? -eq 1 ]; then \
		echo ""; \
		echo "Vet found suspicious constructs. Please check the reported constructs"; \
		echo "and fix them if necessary before submitting the code for review."; \
		exit 1; \
	fi

fmt:
	gofmt -w $(GOFMT_FILES)

fmtcheck:
	@sh -c "'$(CURDIR)/scripts/gofmtcheck.sh'"

errcheck:
	@sh -c "'$(CURDIR)/scripts/errcheck.sh'"

test-compile:
	@if [ "$(TEST)" = "./..." ]; then \
		echo "ERROR: Set TEST to a specific package. For example,"; \
		echo "  make test-compile TEST=./$(PKG_NAME)"; \
		exit 1; \
	fi
	go test -c $(TEST) $(TESTARGS)

changelog:
	@test $${RELEASE_VERSION?Please set environment variable RELEASE_VERSION}
	@test $${CHANGELOG_GITHUB_TOKEN?Please set environment variable CHANGELOG_GITHUB_TOKEN}
	@docker run -it --rm \
		-v $$PWD:/usr/local/src/your-app \
		-e CHANGELOG_GITHUB_TOKEN=$$CHANGELOG_GITHUB_TOKEN \
		ferrarimarco/github-changelog-generator \
		--user grafana \
		--project terraform-provider-grafana \
		--future-release $$RELEASE_VERSION
	@git add CHANGELOG.md && git commit -m "Release $$RELEASE_VERSION"

release: changelog
	@git tag $$RELEASE_VERSION
	@git push origin $$RELEASE_VERSION
