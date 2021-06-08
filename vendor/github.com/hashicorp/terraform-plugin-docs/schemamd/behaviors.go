package schemamd

import (
	tfjson "github.com/hashicorp/terraform-json"
)

// childIsRequired returns true for blocks with min items > 0 or explicitly required
// attributes
func childIsRequired(block *tfjson.SchemaBlockType, att *tfjson.SchemaAttribute) bool {
	if att != nil {
		return att.Required
	}

	return block.MinItems > 0
}

// childIsOptional returns true for blocks with with min items 0, but any required or
// optional children, or explicitly optional attributes
func childIsOptional(block *tfjson.SchemaBlockType, att *tfjson.SchemaAttribute) bool {
	if att != nil {
		return att.Optional
	}

	if block.MinItems > 0 {
		return false
	}

	for _, childBlock := range block.Block.NestedBlocks {
		if childIsRequired(childBlock, nil) {
			return true
		}
		if childIsOptional(childBlock, nil) {
			return true
		}
	}

	for _, childAtt := range block.Block.Attributes {
		if childIsRequired(nil, childAtt) {
			return true
		}
		if childIsOptional(nil, childAtt) {
			return true
		}
	}

	return false
}

// childIsReadOnly returns true for blocks where all leaves are read only (computed
// but not optional)
func childIsReadOnly(block *tfjson.SchemaBlockType, att *tfjson.SchemaAttribute) bool {
	if att != nil {
		// these shouldn't be able to be required, but just in case
		return att.Computed && !att.Optional && !att.Required
	}

	if block.MinItems != 0 || block.MaxItems != 0 {
		return false
	}

	for _, childBlock := range block.Block.NestedBlocks {
		if !childIsReadOnly(childBlock, nil) {
			return false
		}
	}

	for _, childAtt := range block.Block.Attributes {
		if !childIsReadOnly(nil, childAtt) {
			return false
		}
	}

	return true
}
