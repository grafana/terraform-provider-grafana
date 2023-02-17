#!/usr/bin/env bash

# Generate docs!
go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs -ignore-deprecated

# Generate subcategories
# Follow https://github.com/hashicorp/terraform-plugin-docs/issues/156 to see if anything is merged upstream. If so, we should use that.
SCRIPTPATH="$( cd -- "$(dirname "$0")" >/dev/null 2>&1 ; pwd -P )"

echo -e "\nAdding subcategories to docs..."
for f in $(find ./docs -name "*.md" | sed -e 's|^\./docs/||' -e 's|\.md$||'); do
    echo "  - ${f}"

    subcategory=$(jq -r --arg cat "${f}" '.[$cat]' "${SCRIPTPATH}/subcategories.json")
    if [[ "${subcategory}" == "ignore" ]]; then
        continue
    elif [[ "${subcategory}" != "null" ]] && [[ -n "${subcategory}" ]]; then
        sed -i.bak "s/subcategory: \"\"/subcategory: \"${subcategory}\"/" ./docs/${f}.md &&
            rm ./docs/${f}.md.bak
    else
        echo "No subcategory for $f. Define one in ./tools/subcategories.json" && exit 1
    fi
done
