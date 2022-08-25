#!/usr/bin/env bash

# Generate docs!
go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs

# Generate subcategories
# Follow https://github.com/hashicorp/terraform-plugin-docs/issues/156 to see if anything is merged upstream. If so, we should use that.
SCRIPTPATH="$( cd -- "$(dirname "$0")" >/dev/null 2>&1 ; pwd -P )"
subcategories="$(cat ${SCRIPTPATH}/subcategories.json)"

echo -e "\nAdding subcategories to docs..."
for f in $(find ./docs/ -name "*.md"); do
    f=${f#"./docs/"}
    f="${f%.*}"
    echo "  - ${f}"

    subcategory="$(echo ${subcategories} | jq -r ".[\"${f}\"]")"
    if [[ "${subcategory}" == "ignore" ]]; then
        continue
    elif [[ "${subcategory}" != "null" ]] && [[ -n "${subcategory}" ]]; then
        sed -i "s/subcategory: \"\"/subcategory: \"${subcategory}\"/" ./docs/${f}.md
    else
        echo "No subcategory for $f. Define one in ./tools/subcategories.json" && exit 1
    fi
done