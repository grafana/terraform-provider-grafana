package generate

import (
	"context"
	"fmt"

	tfjson "github.com/hashicorp/terraform-json"
)

func getState(ctx context.Context, cfg *Config) (*tfjson.State, error) {
	state, err := cfg.Terraform.Show(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to read terraform state: %w", err)
	}
	return state, nil
}
