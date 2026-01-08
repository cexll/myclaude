#!/usr/bin/env node
// Execute JavaScript in the active browser tab
const http = require('http');
const WebSocket = require('ws');

const code = process.argv[2];
if (!code) {
  console.error('Usage: eval.js <javascript-expression>');
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

(async () => {
  try {
    const targets = await getTargets();
    const page = targets.find(t => t.type === 'page');
    if (!page) throw new Error('No active page found');

    const ws = new WebSocket(page.webSocketDebuggerUrl);

    ws.on('open', () => {
      ws.send(JSON.stringify({
        id: 1,
        method: 'Runtime.evaluate',
        params: {
          expression: code,
          returnByValue: true,
          awaitPromise: true
        }
      }));
    });

    ws.on('message', data => {
      const msg = JSON.parse(data);
      if (msg.id === 1) {
        ws.close();
        if (msg.result.exceptionDetails) {
          console.error('Error:', msg.result.exceptionDetails.text);
          process.exit(1);
        }
        console.log(JSON.stringify(msg.result.result.value ?? msg.result.result));
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
