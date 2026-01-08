#!/usr/bin/env node
// Navigate to URL in current or new tab
const http = require('http');

const url = process.argv[2];
const newTab = process.argv.includes('--new');

if (!url) {
  console.error('Usage: nav.js <url> [--new]');
  process.exit(1);
}

async function getTargets() {
  return new Promise((resolve, reject) => {
    http.get('http://localhost:9222/json', res => {
      let data = '';
      res.on('data', chunk => data += chunk);
      res.on('end', () => resolve(JSON.parse(data)));
    }).on('error', reject);
  });
}

async function createTab(url) {
  return new Promise((resolve, reject) => {
    http.get(`http://localhost:9222/json/new?${encodeURIComponent(url)}`, res => {
      let data = '';
      res.on('data', chunk => data += chunk);
      res.on('end', () => resolve(JSON.parse(data)));
    }).on('error', reject);
  });
}

async function navigate(targetId, url) {
  const WebSocket = require('ws');
  const targets = await getTargets();
  const target = targets.find(t => t.id === targetId);

  return new Promise((resolve, reject) => {
    const ws = new WebSocket(target.webSocketDebuggerUrl);
    ws.on('open', () => {
      ws.send(JSON.stringify({ id: 1, method: 'Page.navigate', params: { url } }));
    });
    ws.on('message', data => {
      const msg = JSON.parse(data);
      if (msg.id === 1) {
        ws.close();
        resolve(msg.result);
      }
    });
    ws.on('error', reject);
  });
}

(async () => {
  try {
    if (newTab) {
      const tab = await createTab(url);
      console.log(JSON.stringify({ action: 'created', tabId: tab.id, url }));
    } else {
      const targets = await getTargets();
      const page = targets.find(t => t.type === 'page');
      if (!page) throw new Error('No active page found');
      await navigate(page.id, url);
      console.log(JSON.stringify({ action: 'navigated', tabId: page.id, url }));
    }
  } catch (e) {
    console.error('Error:', e.message);
    process.exit(1);
  }
})();
