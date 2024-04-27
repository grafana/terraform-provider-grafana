package main

import (
	"os"
	"os/exec"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/zclconf/go-cty/cty"
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

func newBlock(labels []string, attributes map[string]any) *hclwrite.Block {
	b := hclwrite.NewBlock(labels[0], labels[1:])
	for k, v := range attributes {
		switch v := v.(type) {
		case hcl.Traversal:
			b.Body().SetAttributeTraversal(k, v)
		case string:
			b.Body().SetAttributeValue(k, cty.StringVal(v))
		case cty.Value:
			b.Body().SetAttributeValue(k, v)
		case []map[string]any: // Simplified blocks
			for _, blockAttributes := range v {
				b.Body().AppendBlock(newBlock([]string{k}, blockAttributes))
			}
		}
	}
	return b
}

func providerBlock(attributes map[string]any) *hclwrite.Block {
	return newBlock([]string{"provider", "grafana"}, attributes)
}

func resourceBlock(resourceType, resourceName string, attributes map[string]any) *hclwrite.Block {
	return newBlock([]string{"resource", resourceType, resourceName}, attributes)
}

func traversal(root string, attrs ...string) hcl.Traversal {
	tr := hcl.Traversal{hcl.TraverseRoot{Name: root}}
	for _, attr := range attrs {
		tr = append(tr, hcl.TraverseAttr{Name: attr})
	}
	return tr
}
