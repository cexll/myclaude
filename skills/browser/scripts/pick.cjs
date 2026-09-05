#!/usr/bin/env node
// Visual element picker - click to select DOM nodes
const http = require('http');
const WebSocket = require('ws');

const hint = process.argv[2] || 'Click an element to select it';

async function getTargets() {
  return new Promise((resolve, reject) => {
    http.get('http://localhost:9222/json', res => {
      let data = '';
      res.on('data', chunk => data += chunk);
      res.on('end', () => resolve(JSON.parse(data)));
    }).on('error', reject);
  });
}

const pickerScript = `
(function(hint) {
  return new Promise(resolve => {
    const overlay = document.createElement('div');
    overlay.style.cssText = 'position:fixed;top:0;left:0;right:0;bottom:0;z-index:999999;cursor:crosshair;';

    const label = document.createElement('div');
    label.textContent = hint;
    label.style.cssText = 'position:fixed;top:10px;left:50%;transform:translateX(-50%);background:#333;color:#fff;padding:8px 16px;border-radius:4px;z-index:1000000;font:14px sans-serif;';

    document.body.appendChild(overlay);
    document.body.appendChild(label);

    overlay.onclick = e => {
      overlay.remove();
      label.remove();
      const el = document.elementFromPoint(e.clientX, e.clientY);
      if (!el) return resolve(null);

      const rect = el.getBoundingClientRect();
      resolve({
        tag: el.tagName.toLowerCase(),
        id: el.id || null,
        classes: [...el.classList],
        text: el.textContent?.slice(0, 100)?.trim() || null,
        href: el.href || null,
        selector: el.id ? '#' + el.id : el.className ? el.tagName.toLowerCase() + '.' + [...el.classList].join('.') : el.tagName.toLowerCase(),
        rect: { x: rect.x, y: rect.y, width: rect.width, height: rect.height }
      });
    };
  });
})`;

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
          expression: `${pickerScript}(${JSON.stringify(hint)})`,
          returnByValue: true,
          awaitPromise: true
        }
      }));
    });

    ws.on('message', data => {
      const msg = JSON.parse(data);
      if (msg.id === 1) {
        ws.close();
        console.log(JSON.stringify(msg.result.result.value, null, 2));
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
