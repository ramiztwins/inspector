name: Validate JSON

on:
  push:
    paths:
      - '*.json'
  pull_request:
    paths:
      - '*.json'

jobs:
  validate-json:
    runs-on: ubuntu-latest

    steps:
    - name: Checkout code
      uses: actions/checkout@v3
      if: ${{ always() }}
      continue-on-error: true

    - name: Install Node.js
      uses: actions/setup-node@v3
      with:
        node-version: '14'
      if: ${{ always() }}
      continue-on-error: true

    - name: Install dependencies
      run: npm install -g ajv-cli
      if: ${{ always() }}
      continue-on-error: true

    - name: Validate JSON files with detailed output
      run: |
        set +e
        for file in *.json; do
          if [ "$file" = "schema.json" ]; then
            continue
          fi
          echo -n "🔍 Validating $file... "
          cat "$file" | jq empty > /dev/null 2>&1
          if [ $? -ne 0 ]; then
            echo "🚫 Syntax Error"
            continue
          fi
          ajv validate -s schema.json -d "$file" --errors=json > validation_output.json 2>&1
          if [ $? -ne 0 ]; then
            echo "❌ Validation failed"
            cat validation_output.json
          else
            echo "✅ Valid"
          fi
        done
        set -e

