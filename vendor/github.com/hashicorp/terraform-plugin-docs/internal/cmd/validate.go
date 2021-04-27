package cmd

import (
	"flag"
	"fmt"

	"github.com/hashicorp/terraform-plugin-docs/internal/provider"
)

type validateCmd struct {
	commonCmd
}

func (cmd *validateCmd) Synopsis() string {
	return "validates a plugin website for the current directory"
}

func (cmd *validateCmd) Help() string {
	return `Usage: tfplugindocs validate`
}

func (cmd *validateCmd) Flags() *flag.FlagSet {
	fs := flag.NewFlagSet("validate", flag.ExitOnError)
	return fs
}

func (cmd *validateCmd) Run(args []string) int {
	fs := cmd.Flags()
	err := fs.Parse(args)
	if err != nil {
		cmd.ui.Error(fmt.Sprintf("unable to parse flags: %s", err))
		return 1
	}

	return cmd.run(cmd.runInternal)
}

func (cmd *validateCmd) runInternal() error {
	err := provider.Validate(cmd.ui)
	if err != nil {
		return fmt.Errorf("unable to validate website: %w", err)
	}

	return nil
}
