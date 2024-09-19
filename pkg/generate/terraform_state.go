package generate

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/hashicorp/terraform-exec/tfexec"
	tfjson "github.com/hashicorp/terraform-json"
)

func getState(ctx context.Context, cfg *Config) (*tfjson.State, error) {
	state, err := cfg.Terraform.Show(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to read terraform state: %w", err)
	}
	return state, nil
}

func getPlannedState(ctx context.Context, cfg *Config) (*tfjson.Plan, error) {
	tempWorkingDir, err := os.MkdirTemp("", "terraform-generate")
	if err != nil {
		return nil, fmt.Errorf("failed to create temporary working directory: %w", err)
	}
	defer os.RemoveAll(tempWorkingDir)

	planFile := filepath.Join(tempWorkingDir, "plan.tfplan")
	if _, err := cfg.Terraform.Plan(ctx, tfexec.Out(planFile)); err != nil {
		return nil, fmt.Errorf("failed to read terraform plan: %w", err)
	}

	return cfg.Terraform.ShowPlanFile(ctx, planFile)
}
