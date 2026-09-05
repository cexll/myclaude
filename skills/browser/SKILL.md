---
name: browser
description: This skill should be used for browser automation tasks using Chrome DevTools Protocol (CDP). Triggers when users need to launch Chrome with remote debugging, navigate pages, execute JavaScript in browser context, capture screenshots, or interactively select DOM elements. No MCP server required.
---

# Browser Automation

Minimal Chrome DevTools Protocol (CDP) helpers for browser automation without MCP server setup.

## Setup

Install dependencies before first use:

```bash
npm install --prefix ~/.claude/skills/browser/browser ws
```

## Scripts

All scripts connect to Chrome on `localhost:9222`.

### start.js - Launch Chrome

```bash
scripts/start.js              # Fresh profile
scripts/start.js --profile    # Use persistent profile (keeps cookies/auth)
```

### nav.js - Navigate

```bash
scripts/nav.js https://example.com        # Navigate current tab
scripts/nav.js https://example.com --new  # Open in new tab
```

### eval.js - Execute JavaScript

```bash
scripts/eval.js 'document.title'
scripts/eval.js '(() => { const x = 1; return x + 1; })()'
```

Use single expressions or IIFE for multiple statements.

### screenshot.js - Capture Screenshot

```bash
scripts/screenshot.js
```

Returns `{ path, filename }` of saved PNG in temp directory.

### pick.js - Visual Element Picker

```bash
scripts/pick.js "Click the submit button"
```

Returns element metadata: tag, id, classes, text, href, selector, rect.

## Workflow

1. Launch Chrome: `scripts/start.js --profile` for authenticated sessions
2. Navigate: `scripts/nav.js <url>`
3. Inspect: `scripts/eval.js 'document.querySelector(...)'`
4. Capture: `scripts/screenshot.js` or `scripts/pick.js`
5. Return gathered data

## Key Points

- All operations run locally - credentials never leave the machine
- Use `--profile` flag to preserve cookies and auth tokens
- Scripts return structured JSON for agent consumption
