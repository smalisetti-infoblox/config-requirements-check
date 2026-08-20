#!/bin/bash
# Ensure starter.yaml is updated when new requirements are added

set -e

# Check if any config-requirements files were modified
config_files_changed=$(git diff --cached --name-only | grep -E "(config-requirements|requirements)" | grep -E "\.yaml$" || true)

if [ -z "$config_files_changed" ]; then
  exit 0
fi

echo "🔍 Checking if starter.yaml needs updating..."

# For each changed config file, check if requirements were added
new_requirements_added=false

for config_file in $config_files_changed; do
  if [ ! -f "$config_file" ]; then
    continue
  fi

  # Get list of requirement IDs in the staged config file
  new_ids=$(grep -oP "^\s*-\s+id:\s+\K[^ ]+|id:\s+\K[^ ]+" "$config_file" 2>/dev/null | sort | uniq || true)

  if [ -z "$new_ids" ]; then
    continue
  fi

  # Get list of requirement IDs in examples/starter.yaml
  if [ -f "examples/starter.yaml" ]; then
    starter_ids=$(grep -oP "^\s*-\s+id:\s+\K[^ ]+|id:\s+\K[^ ]+" "examples/starter.yaml" 2>/dev/null | sort | uniq || true)
  else
    starter_ids=""
  fi

  # Check if any IDs in new config are NOT in starter.yaml
  for id in $new_ids; do
    if [ -n "$starter_ids" ] && echo "$starter_ids" | grep -q "^${id}$"; then
      # ID exists in starter, that's good
      continue
    else
      # ID is in new config but not in starter
      if [ "$config_file" = "examples/starter.yaml" ]; then
        # If the config file IS the starter, skip this check
        continue
      fi

      # Check if this is a new ID (not in git yet) by looking at diff
      if git diff --cached "$config_file" | grep -q "^\+.*id: $id"; then
        echo "❌ New requirement found: $id"
        echo "   This requirement is in $config_file but NOT in examples/starter.yaml"
        new_requirements_added=true
      fi
    fi
  done
done

if [ "$new_requirements_added" = true ]; then
  echo ""
  echo "Action required:"
  echo "  1. Add corresponding example to examples/starter.yaml"
  echo "  2. Show the new feature in the starter template"
  echo "  3. Include comments explaining the new requirement"
  echo ""
  echo "Example structure:"
  echo "  - id: your-new-feature"
  echo "    summary: >-"
  echo "      Description of what this requires"
  echo "    conditions:"
  echo "      - path: some.config.path"
  echo "        equals: true"
  echo "    requires:"
  echo "      - path: another.config.path"
  echo "        equals: expected_value"
  echo "    remediation: How to fix if not met"
  echo ""
  exit 1
fi

exit 0
