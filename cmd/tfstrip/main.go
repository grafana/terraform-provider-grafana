package main

import (
	"errors"
	"log"
	"os"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"
)

type Config struct {
	LogLevel string `hcl:"log_level"`
}

func main() {
	src, err := os.ReadFile("/home/duologic/git/grafana/terraform-provider-grafana/out/stack-terraformprovidergrafana2-resources.tf")
	if err != nil {
		panic(err)
	}

	file, diags := hclwrite.ParseConfig(src, "decode.tf", hcl.Pos{Line: 1, Column: 1})
	if diags.HasErrors() {
		err := errors.New("an error occurred")
		if err != nil {
			panic(err)
		}
	}
	for _, block := range file.Body().Blocks() {
		for name, attribute := range block.Body().Attributes() {
			log.Printf("%+v", name)
			if string(attribute.Expr().BuildTokens(nil).Bytes()) == " null" {
				block.Body().RemoveAttribute(name)
			}
		}
	}

	log.Print(string(file.Bytes()))
}
