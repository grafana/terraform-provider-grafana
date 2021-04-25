package cmd

import (
	"flag"
	"fmt"

	"github.com/hashicorp/terraform-plugin-docs/internal/provider"
)

type generateCmd struct {
	commonCmd

	flagLegacySidebar bool
}

func (cmd *generateCmd) Synopsis() string {
	return "generates a plugin website from code, templates, and examples for the current directory"
}

func (cmd *generateCmd) Help() string {
	return `Usage: tfplugindocs generate`
}

func (cmd *generateCmd) Flags() *flag.FlagSet {
	fs := flag.NewFlagSet("generate", flag.ExitOnError)
	fs.BoolVar(&cmd.flagLegacySidebar, "legacy-sidebar", false, "generate the legacy .erb sidebar file")
	return fs
}

func (cmd *generateCmd) Run(args []string) int {
	fs := cmd.Flags()
	err := fs.Parse(args)
	if err != nil {
		cmd.ui.Error(fmt.Sprintf("unable to parse flags: %s", err))
		return 1
	}

	return cmd.run(cmd.runInternal)
}

func (cmd *generateCmd) runInternal() error {
	err := provider.Generate(cmd.ui, cmd.flagLegacySidebar)
	if err != nil {
		return fmt.Errorf("unable to generate website: %w", err)
	}

	return nil
}
