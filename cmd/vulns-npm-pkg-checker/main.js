#!/usr/bin/env node

const fs = require('fs');
const path = require('path');
const { execSync } = require('child_process');

// Load vulnerable packages from stdin or JSON file
function loadVulnerablePackages() {
  // Check if data is being piped from stdin
  if (!process.stdin.isTTY) {
    let stdinData = '';
    process.stdin.setEncoding('utf8');
    process.stdin.on('data', (chunk) => {
      stdinData += chunk;
    });
    process.stdin.on('end', () => {
      try {
        const vulnerablePackages = JSON.parse(stdinData);
        const checker = new VulnerabilityChecker(vulnerablePackages);
        checker.scan();
      } catch (error) {
        console.error('Error parsing JSON from stdin:', error.message);
        process.exit(1);
      }
    });
    return null; // Will be handled in stdin end event
  }

  // Fallback to JSON file
  try {
    const jsonPath = path.join(__dirname, 'vulnerable-packages.json');
    const jsonData = fs.readFileSync(jsonPath, 'utf8');
    return JSON.parse(jsonData);
  } catch (error) {
    console.error('Error loading vulnerable-packages.json:', error.message);
    process.exit(1);
  }
}

class VulnerabilityChecker {
  constructor(vulnPackages = null) {
    this.vulnerablePackages = vulnPackages || loadVulnerablePackages();
    this.results = {
      found: [],
      notFound: [],
      errors: []
    };
  }

  // Find all package.json files in the repository
  findPackageJsonFiles(dir = '.', packageFiles = []) {
    try {
      const items = fs.readdirSync(dir);

      for (const item of items) {
        const fullPath = path.join(dir, item);
        const stat = fs.statSync(fullPath);

        if (stat.isDirectory()) {
          // Skip node_modules and other common directories that should be ignored
          if (!['node_modules', '.git', '.vscode', 'coverage', 'dist', 'build'].includes(item)) {
            this.findPackageJsonFiles(fullPath, packageFiles);
          }
        } else if (item === 'package.json') {
          packageFiles.push(fullPath);
        }
      }
    } catch (error) {
      this.results.errors.push(`Error reading directory ${dir}: ${error.message}`);
    }

    return packageFiles;
  }

  // Check a single package.json file for vulnerable packages
  checkPackageJson(filePath) {
    try {
      const content = fs.readFileSync(filePath, 'utf8');
      const packageData = JSON.parse(content);
      const foundVulns = [];

      // Check both dependencies and devDependencies
      const allDeps = {
        ...packageData.dependencies || {},
        ...packageData.devDependencies || {},
        ...packageData.peerDependencies || {},
        ...packageData.optionalDependencies || {}
      };

      for (const [packageName, installedVersion] of Object.entries(allDeps)) {
        if (this.vulnerablePackages[packageName]) {
          const vulnerableVersions = this.vulnerablePackages[packageName];

          // Clean the installed version (remove ^ ~ etc.)
          const cleanVersion = installedVersion.replace(/^[\^~>=<]/, '');

          if (vulnerableVersions.includes(cleanVersion)) {
            foundVulns.push({
              package: packageName,
              installedVersion: installedVersion,
              vulnerableVersions: vulnerableVersions,
              file: filePath
            });
          }
        }
      }

      return foundVulns;
    } catch (error) {
      this.results.errors.push(`Error processing ${filePath}: ${error.message}`);
      return [];
    }
  }

  // Check installed packages using npm ls (if available)
  checkInstalledPackages() {
    const foundVulns = [];

    try {
      // Try to get the list of installed packages
      const npmLsOutput = execSync('npm ls --json --depth=0', {
        encoding: 'utf8',
        stdio: ['pipe', 'pipe', 'ignore'] // Ignore stderr to avoid warnings
      });

      const installedPackages = JSON.parse(npmLsOutput);

      if (installedPackages.dependencies) {
        for (const [packageName, packageInfo] of Object.entries(installedPackages.dependencies)) {
          if (this.vulnerablePackages[packageName]) {
            const vulnerableVersions = this.vulnerablePackages[packageName];
            const installedVersion = packageInfo.version;

            if (vulnerableVersions.includes(installedVersion)) {
              foundVulns.push({
                package: packageName,
                installedVersion: installedVersion,
                vulnerableVersions: vulnerableVersions,
                source: 'npm ls'
              });
            }
          }
        }
      }
    } catch (error) {
      // npm ls might fail in some environments, that's okay
      this.results.errors.push(`Could not run npm ls: ${error.message}`);
    }

    return foundVulns;
  }

  // Main scan function
  scan() {
    console.log('🔍 Scanning for vulnerable packages...\n');

    // Find all package.json files
    const packageFiles = this.findPackageJsonFiles();
    console.log(`Found ${packageFiles.length} package.json files`);

    // Check each package.json file
    for (const filePath of packageFiles) {
      const vulns = this.checkPackageJson(filePath);
      this.results.found.push(...vulns);
    }

    // Also check currently installed packages
    const installedVulns = this.checkInstalledPackages();
    this.results.found.push(...installedVulns);

    // Remove duplicates
    this.results.found = this.results.found.filter((vuln, index, self) =>
      index === self.findIndex(v => v.package === vuln.package &&
        v.installedVersion === vuln.installedVersion)
    );

    // Report results
    this.reportResults();
  }

  // Report the scan results
  reportResults() {
    console.log('\n📊 VULNERABILITY SCAN RESULTS\n');
    console.log('='.repeat(50));

    if (this.results.found.length > 0) {
      console.log(`\n❌ FOUND ${this.results.found.length} VULNERABLE PACKAGES:\n`);

      this.results.found.forEach((vuln, index) => {
        console.log(`${index + 1}. ${vuln.package}`);
        console.log(`   Installed: ${vuln.installedVersion}`);
        console.log(`   Vulnerable versions: ${vuln.vulnerableVersions.join(', ')}`);
        if (vuln.file) {
          console.log(`   Found in: ${vuln.file}`);
        } else if (vuln.source) {
          console.log(`   Source: ${vuln.source}`);
        }
        console.log('');
      });

      console.log('⚠️  RECOMMENDED ACTIONS:');
      console.log('   - Update these packages to secure versions');
      console.log('   - Review package changelogs for breaking changes');
      console.log('   - Test thoroughly after updates\n');
    } else {
      console.log('✅ No vulnerable packages found!\n');
    }

    if (this.results.errors.length > 0) {
      console.log(`\n⚠️  ENCOUNTERED ${this.results.errors.length} ERRORS:\n`);
      this.results.errors.forEach((error, index) => {
        console.log(`${index + 1}. ${error}`);
      });
      console.log('');
    }

    console.log('='.repeat(50));
    console.log(`\nScan completed at ${new Date().toISOString()}`);

    // Exit with error code if vulnerabilities found
    if (this.results.found.length > 0) {
      process.exit(1);
    }
  }
}

// Run the scanner if this script is executed directly
if (require.main === module) {
  const vulnerablePackages = loadVulnerablePackages();
  if (vulnerablePackages) {
    const checker = new VulnerabilityChecker(vulnerablePackages);
    checker.scan();
  }
}

module.exports = VulnerabilityChecker;