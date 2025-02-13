BIN_DIR := bin
SDK_VER := 0.23.1
SDK_CLI := $(BIN_DIR)/grafana-app-sdk-$(SDK_VER)

KINDSS_DIR := kinds
GOFGEN_DIR := pkg/generated
KIND_FILES := $(shell find $(KINDSS_DIR) -type f -name "*.cue")

all: generate

$(SDK_CLI):
	@mkdir -p $(BIN_DIR)
	@./scripts/install-sdk.sh $(SDK_VER) $(BIN_DIR)
sdk-cli: $(SDK_CLI)

$(GOFGEN_DIR): $(KIND_FILES) $(SDK_CLI)
	@$(SDK_CLI) generate --cuepath=$(KINDSS_DIR) --gogenpath=$(GOFGEN_DIR) --nomanifest --crdencoding=none
	@touch $@
generate: $(GOFGEN_DIR)
