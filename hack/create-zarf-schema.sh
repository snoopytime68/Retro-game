#!/usr/bin/env bash

set -euo pipefail

add_yaml_extensions() {
  local input_file=$1
  local dst_folder="schema"

  jq '
    def addPatternProperties:
      . +
      if has("properties") then
        {"patternProperties": {"^x-": {}}}
      else
        {}
      end;

    walk(if type == "object" then addPatternProperties else . end)
  ' "$input_file" > "$dst_folder/$input_file"
  rm "$input_file"
}

go run schema/src/main.go v1alpha1 > "zarf_package_v1alpha1.schema.json"
go run schema/src/main.go v1beta1 > "zarf_package_v1beta1.schema.json"

add_yaml_extensions "zarf_package_v1alpha1.schema.json"
add_yaml_extensions "zarf_package_v1beta1.schema.json"
