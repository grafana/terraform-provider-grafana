package schemamd

import (
	"fmt"
	"io"
	"sort"
	"strings"

	tfjson "github.com/hashicorp/terraform-json"
	"github.com/zclconf/go-cty/cty"
)

func Render(schema *tfjson.Schema, w io.Writer) error {
	_, err := io.WriteString(w, "## Schema\n\n")
	if err != nil {
		return err
	}

	err = writeRootBlock(w, schema.Block)
	if err != nil {
		return fmt.Errorf("unable to render schema: %w", err)
	}

	return nil
}

type groupFilter struct {
	topLevelTitle string
	nestedTitle   string

	// only one of these will be passed depending on the type of child
	filter func(block *tfjson.SchemaBlockType, att *tfjson.SchemaAttribute) bool
}

var (
	groupFilters = []groupFilter{
		{"### Required", "Required:", childIsRequired},
		{"### Optional", "Optional:", childIsOptional},
		{"### Read-Only", "Read-Only:", childIsReadOnly},
	}
)

type nestedType struct {
	anchorID string
	path     []string
	block    *tfjson.SchemaBlock
	object   *cty.Type

	group groupFilter
}

func writeAttribute(w io.Writer, path []string, att *tfjson.SchemaAttribute, group groupFilter) ([]nestedType, error) {
	name := path[len(path)-1]

	_, err := io.WriteString(w, "- **"+name+"** ")
	if err != nil {
		return nil, err
	}

	if name == "id" && att.Description == "" {
		att.Description = "The ID of this resource."
	}

	err = WriteAttributeDescription(w, att, false)
	if err != nil {
		return nil, err
	}
	if att.AttributeType.IsTupleType() {
		return nil, fmt.Errorf("TODO: tuples are not yet supported")
	}

	anchorID := "nestedatt--" + strings.Join(path, "--")
	nestedTypes := []nestedType{}
	switch {
	case att.AttributeType.IsObjectType():
		_, err = io.WriteString(w, " (see [below for nested schema](#"+anchorID+"))")
		if err != nil {
			return nil, err
		}

		nestedTypes = append(nestedTypes, nestedType{
			anchorID: anchorID,
			path:     path,
			object:   &att.AttributeType,

			group: group,
		})
	case att.AttributeType.IsCollectionType() && att.AttributeType.ElementType().IsObjectType():
		_, err = io.WriteString(w, " (see [below for nested schema](#"+anchorID+"))")
		if err != nil {
			return nil, err
		}

		nt := att.AttributeType.ElementType()
		nestedTypes = append(nestedTypes, nestedType{
			anchorID: anchorID,
			path:     path,
			object:   &nt,

			group: group,
		})
	}

	_, err = io.WriteString(w, "\n")
	if err != nil {
		return nil, err
	}

	return nestedTypes, nil
}

func writeBlockType(w io.Writer, path []string, block *tfjson.SchemaBlockType) ([]nestedType, error) {
	name := path[len(path)-1]

	_, err := io.WriteString(w, "- **"+name+"** ")
	if err != nil {
		return nil, err
	}

	err = WriteBlockTypeDescription(w, block)
	if err != nil {
		return nil, fmt.Errorf("unable to write block description for %q: %w", name, err)
	}

	anchorID := "nestedblock--" + strings.Join(path, "--")
	nt := nestedType{
		anchorID: anchorID,
		path:     path,
		block:    block.Block,
	}

	_, err = io.WriteString(w, " (see [below for nested schema](#"+anchorID+"))")
	if err != nil {
		return nil, err
	}

	_, err = io.WriteString(w, "\n")
	if err != nil {
		return nil, err
	}

	return []nestedType{nt}, nil
}

func writeRootBlock(w io.Writer, block *tfjson.SchemaBlock) error {
	return writeBlockChildren(w, nil, block, true)
}

func writeBlockChildren(w io.Writer, parents []string, block *tfjson.SchemaBlock, root bool) error {
	names := []string{}
	for n := range block.Attributes {
		names = append(names, n)
	}
	for n := range block.NestedBlocks {
		names = append(names, n)
	}

	groups := map[int][]string{}

	for _, n := range names {
		childBlock := block.NestedBlocks[n]
		childAtt := block.Attributes[n]
		for i, gf := range groupFilters {
			if gf.filter(childBlock, childAtt) {
				groups[i] = append(groups[i], n)
				goto NextName
			}
		}
		return fmt.Errorf("no match for %q, this can happen if you have incompatible schema defined, for example an "+
			"optional block where all the child attributes are computed, in which case the block itself should also "+
			"be marked computed", n)
	NextName:
	}

	nestedTypes := []nestedType{}

	for i, gf := range groupFilters {
		sortedNames := groups[i]
		if len(sortedNames) == 0 {
			continue
		}
		sort.Strings(sortedNames)

		groupTitle := gf.topLevelTitle
		if !root {
			groupTitle = gf.nestedTitle
		}

		_, err := io.WriteString(w, groupTitle+"\n\n")
		if err != nil {
			return err
		}

		for _, name := range sortedNames {
			path := append(parents, name)

			if block, ok := block.NestedBlocks[name]; ok {
				nt, err := writeBlockType(w, path, block)
				if err != nil {
					return fmt.Errorf("unable to render block %q: %w", name, err)
				}

				nestedTypes = append(nestedTypes, nt...)
				continue
			}

			if att, ok := block.Attributes[name]; ok {
				nt, err := writeAttribute(w, path, att, gf)
				if err != nil {
					return fmt.Errorf("unable to render attribute %q: %w", name, err)
				}

				nestedTypes = append(nestedTypes, nt...)
				continue
			}

			return fmt.Errorf("unexpected name in schema render %q", name)
		}

		_, err = io.WriteString(w, "\n")
		if err != nil {
			return err
		}
	}

	err := writeNestedTypes(w, nestedTypes)
	if err != nil {
		return err
	}

	return nil
}

func writeNestedTypes(w io.Writer, nestedTypes []nestedType) error {
	for _, nt := range nestedTypes {
		_, err := io.WriteString(w, "<a id=\""+nt.anchorID+"\"></a>\n")
		if err != nil {
			return err
		}

		_, err = io.WriteString(w, "### Nested Schema for `"+strings.Join(nt.path, ".")+"`\n\n")
		if err != nil {
			return err
		}

		switch {
		case nt.block != nil:
			err = writeBlockChildren(w, nt.path, nt.block, false)
			if err != nil {
				return err
			}
		case nt.object != nil:
			err = writeObjectChildren(w, nt.path, *nt.object, nt.group)
			if err != nil {
				return err
			}
		default:
			return fmt.Errorf("missing information on nested block: %s", strings.Join(nt.path, "."))
		}

		_, err = io.WriteString(w, "\n")
		if err != nil {
			return err
		}
	}

	return nil
}

func writeObjectAttribute(w io.Writer, path []string, att cty.Type, group groupFilter) ([]nestedType, error) {
	name := path[len(path)-1]

	_, err := io.WriteString(w, "- **"+name+"** (")
	if err != nil {
		return nil, err
	}

	err = WriteType(w, att)
	if err != nil {
		return nil, err
	}

	_, err = io.WriteString(w, ")")
	if err != nil {
		return nil, err
	}

	if att.IsTupleType() {
		return nil, fmt.Errorf("TODO: tuples are not yet supported")
	}

	anchorID := "nestedobjatt--" + strings.Join(path, "--")
	nestedTypes := []nestedType{}
	switch {
	case att.IsObjectType():
		_, err = io.WriteString(w, " (see [below for nested schema](#"+anchorID+"))")
		if err != nil {
			return nil, err
		}

		nestedTypes = append(nestedTypes, nestedType{
			anchorID: anchorID,
			path:     path,
			object:   &att,

			group: group,
		})
	case att.IsCollectionType() && att.ElementType().IsObjectType():
		_, err = io.WriteString(w, " (see [below for nested schema](#"+anchorID+"))")
		if err != nil {
			return nil, err
		}

		nt := att.ElementType()
		nestedTypes = append(nestedTypes, nestedType{
			anchorID: anchorID,
			path:     path,
			object:   &nt,

			group: group,
		})
	}

	_, err = io.WriteString(w, "\n")
	if err != nil {
		return nil, err
	}

	return nestedTypes, nil
}

func writeObjectChildren(w io.Writer, parents []string, ty cty.Type, group groupFilter) error {
	_, err := io.WriteString(w, group.nestedTitle+"\n\n")
	if err != nil {
		return err
	}

	atts := ty.AttributeTypes()
	sortedNames := []string{}
	for n := range atts {
		sortedNames = append(sortedNames, n)
	}
	sort.Strings(sortedNames)
	nestedTypes := []nestedType{}

	for _, name := range sortedNames {
		att := atts[name]
		path := append(parents, name)

		nt, err := writeObjectAttribute(w, path, att, group)
		if err != nil {
			return fmt.Errorf("unable to render attribute %q: %w", name, err)
		}

		nestedTypes = append(nestedTypes, nt...)
	}

	_, err = io.WriteString(w, "\n")
	if err != nil {
		return err
	}

	err = writeNestedTypes(w, nestedTypes)
	if err != nil {
		return err
	}

	return nil
}
