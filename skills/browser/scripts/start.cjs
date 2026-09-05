#!/usr/bin/env node
// Launch Chrome with remote debugging on port 9222
const { execSync, spawn } = require('child_process');
const path = require('path');
const os = require('os');

const useProfile = process.argv.includes('--profile');
const port = 9222;

// Find Chrome executable
const chromePaths = {
  darwin: '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome',
  linux: '/usr/bin/google-chrome',
  win32: 'C:\\Program Files\\Google\\Chrome\\Application\\chrome.exe'
};
const chromePath = chromePaths[process.platform];

// Build args
const args = [
  `--remote-debugging-port=${port}`,
  '--no-first-run',
  '--no-default-browser-check'
];

if (useProfile) {
  const profileDir = path.join(os.homedir(), '.chrome-debug-profile');
  args.push(`--user-data-dir=${profileDir}`);
} else {
  args.push(`--user-data-dir=${path.join(os.tmpdir(), 'chrome-debug-' + Date.now())}`);
}

console.log(`Starting Chrome on port ${port}${useProfile ? ' (with profile)' : ''}...`);
const chrome = spawn(chromePath, args, { detached: true, stdio: 'ignore' });
chrome.unref();
console.log(`Chrome launched (PID: ${chrome.pid})`);
