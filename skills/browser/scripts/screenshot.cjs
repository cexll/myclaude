#!/usr/bin/env node
// Capture screenshot of the active browser tab
const http = require('http');
const WebSocket = require('ws');
const fs = require('fs');
const path = require('path');
const os = require('os');

async function getTargets() {
  return new Promise((resolve, reject) => {
    http.get('http://localhost:9222/json', res => {
      let data = '';
      res.on('data', chunk => data += chunk);
      res.on('end', () => resolve(JSON.parse(data)));
    }).on('error', reject);
  });
}

(async () => {
  try {
    const targets = await getTargets();
    const page = targets.find(t => t.type === 'page');
    if (!page) throw new Error('No active page found');

    const ws = new WebSocket(page.webSocketDebuggerUrl);

    ws.on('open', () => {
      ws.send(JSON.stringify({
        id: 1,
        method: 'Page.captureScreenshot',
        params: { format: 'png' }
      }));
    });

    ws.on('message', data => {
      const msg = JSON.parse(data);
      if (msg.id === 1) {
        ws.close();
        const filename = `screenshot-${Date.now()}.png`;
        const filepath = path.join(os.tmpdir(), filename);
        fs.writeFileSync(filepath, Buffer.from(msg.result.data, 'base64'));
        console.log(JSON.stringify({ path: filepath, filename }));
      }
    });

    ws.on('error', e => {
      console.error('WebSocket error:', e.message);
      process.exit(1);
    });
  } catch (e) {
    console.error('Error:', e.message);
    process.exit(1);
  }
})();
