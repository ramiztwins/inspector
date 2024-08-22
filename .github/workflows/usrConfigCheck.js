const fs = require('fs');
const deepDiff = require('deep-diff').diff;
const { execSync } = require('child_process');

// This script is designed to validate configuration changes by comparing 
// the current configuration file with a version stored in the main branch. 
// The focus is on ensuring that only specific sections of the configuration 
// (i.e., "targets") are modified, while any other changes are flagged as errors.

function validateConfig(configFile) {
  // Fetches the previous version of the configuration file from the main branch 
  // and stores it locally for comparison.
  execSync(`git show main:${configFile} > old_config.json`);
  
  // Parses both the old and new configuration files into JavaScript objects 
  // to enable a detailed comparison.
  const oldConfig = JSON.parse(fs.readFileSync('old_config.json', 'utf8'));
  const newConfig = JSON.parse(fs.readFileSync(configFile, 'utf8'));

  // Computes the differences between the old and new configurations. 
  // The comparison highlights the sections that have been modified.
  const differences = deepDiff(oldConfig, newConfig) || [];

  // Separates changes in the "targets" section from other changes to enforce 
  // specific rules about which parts of the configuration can be modified.
  const targetChanges = differences.filter(diff => diff.path.includes('targets'));
  const nonTargetChanges = differences.filter(diff => !diff.path.includes('targets'));

  // Check for edits or additions to existing fields in the "targets" section, 
  // which is not allowed under the new validation rules.
  const invalidTargetEdits = targetChanges.filter(change => {
    // Disallow edits (kind 'E') and additions (kind 'N') to existing targets.
    return (change.kind === 'E' || change.kind === 'N') && change.path.some(p => !Number.isInteger(p));
  });

  // Determines the outcome of the validation based on where the changes occurred:
  // - If changes are detected outside of the "targets" section, or no changes 
  //   are detected at all, the validation fails.
  // - If changes are detected to existing fields within the "targets" section, 
  //   or if new fields are added to existing targets, the validation fails.
  // - If new targets are added without modifying existing ones, the validation succeeds.
  if (nonTargetChanges.length > 0 || differences.length === 0) {
    return {
      status: 'ERROR',
      message: 'Changes outside of "targets" or no changes detected.',
      differences: differences,
      nonTargetChanges: nonTargetChanges 
    };
  } else if (invalidTargetEdits.length > 0) {
    return {
      status: 'ERROR',
      message: 'Editing or adding new fields to existing "targets" is not allowed.',
      differences: differences,
      invalidTargetEdits: invalidTargetEdits
    };
  } else if (targetChanges.length > 0) {
    return {
      status: 'SUCCESS',
      message: 'Only new targets have been added, with no modifications to existing ones.',
      differences: differences
    };
  }
}

// This function generates a log file based on the validation result, 
// which can be used for review in CI/CD pipelines or other automated processes.
// The log is written in Markdown format for easy readability.
function writeLogFile(result) {
  let log = '';

  if (result.status === 'ERROR') {
    // Logs an error message if validation fails, highlighting the changes 
    // that caused the failure.
    log += '## ❌ ERROR: ' + result.message + '\n\n';
    log += '```diff\n';
    const diffOutput = fs.readFileSync('diff_output.txt', 'utf8');
    log += diffOutput;
    log += '\n```\n';

    if (result.nonTargetChanges && result.nonTargetChanges.length > 0) {
      log += '\n### ❌ WARNING: Changes outside of "targets" section:\n';
      log += '```diff\n';
      result.nonTargetChanges.forEach(change => {
        log += formatChange(change) + '\n';
      });
      log += '\n```\n';
    }

    if (result.invalidTargetEdits && result.invalidTargetEdits.length > 0) {
      log += '\n### ❌ ERROR: Editing or adding new fields to existing "targets" section is not allowed:\n';
      log += '```diff\n';
      result.invalidTargetEdits.forEach(change => {
        log += formatChange(change) + '\n';
      });
      log += '\n```\n';
    }

    fs.writeFileSync('log.md', log);
    process.exit(1);
  } else if (result.status === 'SUCCESS') {
    // Logs a success message if the validation passes, confirming that 
    // changes were restricted to the "targets" section.
    log += '## ✅ SUCCESS: ' + result.message + '\n\n';
    log += '```diff\n';
    const diffOutput = fs.readFileSync('diff_output.txt', 'utf8');
    log += diffOutput;
    log += '\n```\n';
    fs.writeFileSync('log.md', log);
    process.exit(0);
  }
}

// Helper function to format the differences for inclusion in the log file.
// This ensures that the log output is clear and understandable, highlighting 
// exactly what was changed.
function formatChange(change) {
  let formattedChange = '';
  if (change.kind === 'E') {
    formattedChange += `- ${change.path.join('.')} : ${change.lhs}`;
    formattedChange += `\n+ ${change.path.join('.')} : ${change.rhs}`;
  } else if (change.kind === 'N') {
    formattedChange += `+ ${change.path.join('.')} : ${change.rhs}`;
  } else if (change.kind === 'D') {
    formattedChange += `- ${change.path.join('.')} : ${change.lhs}`;
  }
  return formattedChange;
}

if (require.main === module) {
  const configFile = process.argv[2] || 'config.prod.json';
  const result = validateConfig(configFile);
  writeLogFile(result);
}

module.exports = validateConfig;
