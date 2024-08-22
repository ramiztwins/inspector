const fs = require('fs');
const Ajv2020 = require('ajv/dist/2020');
const addFormats = require('ajv-formats');

// 2020-12 Schema class
const ajv = new Ajv2020({ allErrors: true, strict: false });
addFormats(ajv);

const args = process.argv.slice(2);
const schemaFile = args[0];
const configFile = args[1];

// Load and parse schema and config files.
const schema = JSON.parse(fs.readFileSync(schemaFile, 'utf8'));
const data = JSON.parse(fs.readFileSync(configFile, 'utf8'));

// Compile schema.
const validate = ajv.compile(schema);
const valid = validate(data);

let log = '';

if (!valid) {
  log += '## ❌ ERROR: Check Schema test failed \n\n';
  log += '### Details:\n\n';
  log += '```diff\n';
  validate.errors.forEach(error => {
    log += `- ${error.instancePath} ${error.message}\n`;
  });
  log += '\n```\n';
  fs.writeFileSync('log.md', log);
  process.exit(1);
} else {
  log += '## ✅ SUCCESS: Check Schema test passed \n\n';
  fs.writeFileSync('log.md', log);
  process.exit(0);
}