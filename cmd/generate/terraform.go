package main

import (
	"os"
	"os/exec"

	"github.com/hashicorp/hcl/v2/hclwrite"
)

func runTerraform(dir string, command ...string) error {
	cmd := exec.Command("terraform", command...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func writeBlocks(filepath string, blocks ...*hclwrite.Block) error {
	contents := hclwrite.NewFile()
	for i, b := range blocks {
		if i > 0 {
			contents.Body().AppendNewline()
		}
		contents.Body().AppendBlock(b)
	}

	hclFile, err := os.Create(filepath)
	if err != nil {
		return err
	}
	if _, err := contents.WriteTo(hclFile); err != nil {
		return err
	}
	return hclFile.Close()
}
