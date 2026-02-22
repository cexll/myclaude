#!/usr/bin/env node
"use strict";

const crypto = require("crypto");
const fs = require("fs");
const https = require("https");
const os = require("os");
const path = require("path");
const readline = require("readline");
const zlib = require("zlib");
const { spawn, spawnSync } = require("child_process");

const REPO = { owner: "cexll", name: "myclaude" };
const API_HEADERS = {
  "User-Agent": "myclaude-npx",
  Accept: "application/vnd.github+json",
};
const WRAPPER_REQUIRED_MODULES = new Set(["do", "omo"]);
const WRAPPER_REQUIRED_SKILLS = new Set(["dev"]);

function parseArgs(argv) {
  const out = {
    command: "install",
    installDir: "~/.claude",
    force: false,
    dryRun: false,
    list: false,
    update: false,
    tag: null,
    module: null,
    yes: false,
  };

  let i = 0;
  if (argv[i] && !argv[i].startsWith("-")) {
    out.command = argv[i];
    i++;
  }

  for (; i < argv.length; i++) {
    const a = argv[i];
    if (a === "--install-dir") out.installDir = argv[++i];
    else if (a === "--force") out.force = true;
    else if (a === "--dry-run") out.dryRun = true;
    else if (a === "--list") out.list = true;
    else if (a === "--update") out.update = true;
    else if (a === "--tag") out.tag = argv[++i];
    else if (a === "--module") out.module = argv[++i];
    else if (a === "-y" || a === "--yes") out.yes = true;
    else if (a === "-h" || a === "--help") out.help = true;
    else throw new Error(`Unknown arg: ${a}`);
  }

  return out;
}

function printHelp() {
  process.stdout.write(
    [
      "myclaude (npx installer)",
      "",
      "Usage:",
      "  npx github:cexll/myclaude",
      "  npx github:cexll/myclaude --list",
      "  npx github:cexll/myclaude --update",
      "  npx github:cexll/myclaude --install-dir ~/.claude --force",
      "  npx github:cexll/myclaude uninstall",
      "  npx github:cexll/myclaude uninstall --module bmad,do -y",
      "",
      "Options:",
      "  --install-dir <path>   Default: ~/.claude",
      "  --force                Overwrite existing files",
      "  --dry-run              Print actions only",
      "  --list                 List installable items and exit",
      "  --update               Update already installed modules",
      "  --tag <tag>            Install a specific GitHub tag",
      "  --module <names>       For uninstall: comma-separated module names",
      "  -y, --yes              For uninstall: skip confirmation prompt",
    ].join("\n") + "\n"
  );
}

function withTimeout(promise, ms, label) {
  let timer;
  const timeout = new Promise((_, reject) => {
    timer = setTimeout(() => reject(new Error(`Timeout: ${label}`)), ms);
  });
  return Promise.race([promise, timeout]).finally(() => clearTimeout(timer));
}

function httpsGetJson(url) {
  return new Promise((resolve, reject) => {
    https
      .get(url, { headers: API_HEADERS }, (res) => {
        let body = "";
        res.setEncoding("utf8");
        res.on("data", (d) => (body += d));
        res.on("end", () => {
          if (res.statusCode && res.statusCode >= 400) {
            return reject(
              new Error(`HTTP ${res.statusCode}: ${url}\n${body.slice(0, 500)}`)
            );
          }
          try {
            resolve(JSON.parse(body));
          } catch (e) {
            reject(new Error(`Invalid JSON from ${url}: ${e.message}`));
          }
        });
      })
      .on("error", reject);
  });
}

function downloadToFile(url, outPath) {
  return new Promise((resolve, reject) => {
    const file = fs.createWriteStream(outPath);
    https
      .get(url, { headers: API_HEADERS }, (res) => {
        if (
          res.statusCode &&
          res.statusCode >= 300 &&
          res.statusCode < 400 &&
          res.headers.location
        ) {
          file.close();
          fs.unlink(outPath, () => {
            downloadToFile(res.headers.location, outPath).then(resolve, reject);
          });
          return;
        }
        if (res.statusCode && res.statusCode >= 400) {
          file.close();
          fs.unlink(outPath, () => {});
          return reject(new Error(`HTTP ${res.statusCode}: ${url}`));
        }
        res.pipe(file);
        file.on("finish", () => file.close(resolve));
      })
      .on("error", (err) => {
        file.close();
        fs.unlink(outPath, () => reject(err));
      });
  });
}

async function fetchLatestTag() {
  const url = `https://api.github.com/repos/${REPO.owner}/${REPO.name}/releases/latest`;
  const json = await httpsGetJson(url);
  if (!json || typeof json.tag_name !== "string" || !json.tag_name.trim()) {
    throw new Error("GitHub API: missing tag_name");
  }
  return json.tag_name.trim();
}

async function fetchRemoteConfig(tag) {
  const url = `https://api.github.com/repos/${REPO.owner}/${REPO.name}/contents/config.json?ref=${encodeURIComponent(
    tag
  )}`;
  const json = await httpsGetJson(url);
  if (!json || typeof json.content !== "string") {
    throw new Error("GitHub contents API: missing config.json content");
  }
  const buf = Buffer.from(json.content.replace(/\n/g, ""), "base64");
  return JSON.parse(buf.toString("utf8"));
}

async function fetchRemoteSkills(tag) {
  const url = `https://api.github.com/repos/${REPO.owner}/${REPO.name}/contents/skills?ref=${encodeURIComponent(
    tag
  )}`;
  const json = await httpsGetJson(url);
  if (!Array.isArray(json)) throw new Error("GitHub contents API: skills is not a directory");
  return json
    .filter((e) => e && e.type === "dir" && typeof e.name === "string")
    .map((e) => e.name)
    .sort();
}

function repoRootFromHere() {
  return path.resolve(__dirname, "..");
}

function readLocalConfig() {
  const p = path.join(repoRootFromHere(), "config.json");
  return JSON.parse(fs.readFileSync(p, "utf8"));
}

function listLocalSkills() {
  const root = repoRootFromHere();
  const skillsDir = path.join(root, "skills");
  if (!fs.existsSync(skillsDir)) return [];
  return fs
    .readdirSync(skillsDir, { withFileTypes: true })
    .filter((d) => d.isDirectory())
    .map((d) => d.name)
    .sort();
}

function expandHome(p) {
  if (!p) return p;
  if (p === "~") return os.homedir();
  if (p.startsWith("~/")) return path.join(os.homedir(), p.slice(2));
  return p;
}

function readInstalledModuleNamesFromStatus(installDir) {
  const p = path.join(installDir, "installed_modules.json");
  if (!fs.existsSync(p)) return null;
  try {
    const json = JSON.parse(fs.readFileSync(p, "utf8"));
    const modules = json && json.modules;
    if (!modules || typeof modules !== "object" || Array.isArray(modules)) return null;
    return Object.keys(modules)
      .filter((k) => typeof k === "string" && k.trim())
      .sort();
  } catch {
    return null;
  }
}

function loadInstalledStatus(installDir) {
  const p = path.join(installDir, "installed_modules.json");
  if (!fs.existsSync(p)) return { modules: {} };
  try {
    const json = JSON.parse(fs.readFileSync(p, "utf8"));
    const modules = json && json.modules;
    if (!modules || typeof modules !== "object" || Array.isArray(modules)) return { modules: {} };
    return { ...json, modules };
  } catch {
    return { modules: {} };
  }
}

function saveInstalledStatus(installDir, status) {
  const p = path.join(installDir, "installed_modules.json");
  fs.mkdirSync(installDir, { recursive: true });
  fs.writeFileSync(p, JSON.stringify(status, null, 2) + "\n", "utf8");
}

function upsertModuleStatus(installDir, moduleResult) {
  const status = loadInstalledStatus(installDir);
  status.modules = status.modules || {};
  status.modules[moduleResult.module] = moduleResult;
  status.updated_at = new Date().toISOString();
  saveInstalledStatus(installDir, status);
}

function deleteModuleStatus(installDir, moduleName) {
  const status = loadInstalledStatus(installDir);
  if (status.modules && Object.prototype.hasOwnProperty.call(status.modules, moduleName)) {
    delete status.modules[moduleName];
    status.updated_at = new Date().toISOString();
    saveInstalledStatus(installDir, status);
  }
}

function loadSettings(installDir) {
  const p = path.join(installDir, "settings.json");
  if (!fs.existsSync(p)) return {};
  try {
    return JSON.parse(fs.readFileSync(p, "utf8"));
  } catch {
    return {};
  }
}

function saveSettings(installDir, settings) {
  const p = path.join(installDir, "settings.json");
  fs.mkdirSync(installDir, { recursive: true });
  fs.writeFileSync(p, JSON.stringify(settings, null, 2) + "\n", "utf8");
}

function isPlainObject(x) {
  return !!x && typeof x === "object" && !Array.isArray(x);
}

function deepEqual(a, b) {
  if (a === b) return true;
  if (Array.isArray(a) && Array.isArray(b)) {
    if (a.length !== b.length) return false;
    for (let i = 0; i < a.length; i++) if (!deepEqual(a[i], b[i])) return false;
    return true;
  }
  if (isPlainObject(a) && isPlainObject(b)) {
    const aKeys = Object.keys(a);
    const bKeys = Object.keys(b);
    if (aKeys.length !== bKeys.length) return false;
    for (const k of aKeys) {
      if (!Object.prototype.hasOwnProperty.call(b, k)) return false;
      if (!deepEqual(a[k], b[k])) return false;
    }
    return true;
  }
  return false;
}

function hooksEqual(h1, h2) {
  if (!isPlainObject(h1) || !isPlainObject(h2)) return false;
  const a = { ...h1 };
  const b = { ...h2 };
  delete a.__module__;
  delete b.__module__;
  return deepEqual(a, b);
}

function replaceHookVariables(obj, pluginRoot) {
  if (typeof obj === "string") return obj.replace(/\$\{CLAUDE_PLUGIN_ROOT\}/g, pluginRoot);
  if (Array.isArray(obj)) return obj.map((v) => replaceHookVariables(v, pluginRoot));
  if (isPlainObject(obj)) {
    const out = {};
    for (const [k, v] of Object.entries(obj)) out[k] = replaceHookVariables(v, pluginRoot);
    return out;
  }
  return obj;
}

function mergeHooksToSettings(moduleName, hooksConfig, installDir, pluginRoot) {
  if (!hooksConfig || !isPlainObject(hooksConfig)) return false;
  const rawHooks = hooksConfig.hooks;
  if (!rawHooks || !isPlainObject(rawHooks)) return false;

  const settings = loadSettings(installDir);
  if (!settings.hooks || !isPlainObject(settings.hooks)) settings.hooks = {};

  const moduleHooks = pluginRoot ? replaceHookVariables(rawHooks, pluginRoot) : rawHooks;
  let modified = false;

  for (const [hookType, hookEntries] of Object.entries(moduleHooks)) {
    if (!Array.isArray(hookEntries)) continue;
    if (!Array.isArray(settings.hooks[hookType])) settings.hooks[hookType] = [];

    for (const entry of hookEntries) {
      if (!isPlainObject(entry)) continue;
      const entryCopy = { ...entry, __module__: moduleName };

      let exists = false;
      for (const existing of settings.hooks[hookType]) {
        if (existing && existing.__module__ === moduleName && hooksEqual(existing, entryCopy)) {
          exists = true;
          break;
        }
      }
      if (!exists) {
        settings.hooks[hookType].push(entryCopy);
        modified = true;
      }
    }
  }

  if (modified) saveSettings(installDir, settings);
  return modified;
}

function unmergeHooksFromSettings(moduleName, installDir) {
  const settings = loadSettings(installDir);
  if (!settings.hooks || !isPlainObject(settings.hooks)) return false;

  let modified = false;
  for (const hookType of Object.keys(settings.hooks)) {
    const entries = settings.hooks[hookType];
    if (!Array.isArray(entries)) continue;
    const kept = entries.filter((e) => !(e && e.__module__ === moduleName));
    if (kept.length !== entries.length) {
      settings.hooks[hookType] = kept;
      modified = true;
    }
    if (!settings.hooks[hookType].length) {
      delete settings.hooks[hookType];
      modified = true;
    }
  }

  if (modified) saveSettings(installDir, settings);
  return modified;
}

function mergeModuleHooks(moduleName, mod, installDir) {
  const ops = Array.isArray(mod && mod.operations) ? mod.operations : [];
  let merged = false;

  for (const op of ops) {
    if (!op || op.type !== "copy_dir") continue;
    const target = typeof op.target === "string" ? op.target : "";
    if (!target) continue;

    const targetDir = path.join(installDir, target);
    const hooksFile = path.join(targetDir, "hooks", "hooks.json");
    if (!fs.existsSync(hooksFile)) continue;

    let hooksConfig;
    try {
      hooksConfig = JSON.parse(fs.readFileSync(hooksFile, "utf8"));
    } catch {
      continue;
    }
    if (mergeHooksToSettings(moduleName, hooksConfig, installDir, targetDir)) merged = true;
  }

  return merged;
}

async function dirExists(p) {
  try {
    return (await fs.promises.stat(p)).isDirectory();
  } catch {
    return false;
  }
}

async function mergeDirLooksInstalled(srcDir, installDir) {
  if (!(await dirExists(srcDir))) return false;
  const subdirs = await fs.promises.readdir(srcDir, { withFileTypes: true });
  for (const d of subdirs) {
    if (!d.isDirectory()) continue;
    const srcSub = path.join(srcDir, d.name);
    const entries = await fs.promises.readdir(srcSub, { withFileTypes: true });
    for (const e of entries) {
      if (!e.isFile()) continue;
      const dst = path.join(installDir, d.name, e.name);
      if (fs.existsSync(dst)) return true;
    }
  }
  return false;
}

async function detectInstalledModuleNames(config, repoRoot, installDir) {
  const mods = (config && config.modules) || {};
  const installed = [];

  for (const [name, mod] of Object.entries(mods)) {
    const ops = Array.isArray(mod && mod.operations) ? mod.operations : [];
    let ok = false;

    for (const op of ops) {
      const type = op && op.type;
      if (type === "copy_file" || type === "copy_dir") {
        const target = typeof op.target === "string" ? op.target : "";
        if (target && fs.existsSync(path.join(installDir, target))) {
          ok = true;
          break;
        }
      } else if (type === "merge_dir") {
        const source = typeof op.source === "string" ? op.source : "";
        if (source && (await mergeDirLooksInstalled(path.join(repoRoot, source), installDir))) {
          ok = true;
          break;
        }
      }
    }

    if (ok) installed.push(name);
  }

  return installed.sort();
}

async function updateInstalledModules(installDir, tag, config, dryRun) {
  const mods = (config && config.modules) || {};
  if (!Object.keys(mods).length) throw new Error("No modules found in config.json");

  let repoRoot = repoRootFromHere();
  let tmp = null;

  if (tag) {
    tmp = path.join(
      os.tmpdir(),
      `myclaude-update-${Date.now()}-${crypto.randomBytes(4).toString("hex")}`
    );
    await fs.promises.mkdir(tmp, { recursive: true });
  }

  try {
    if (tag) {
      const archive = path.join(tmp, "src.tgz");
      const url = `https://codeload.github.com/${REPO.owner}/${REPO.name}/tar.gz/refs/tags/${encodeURIComponent(
        tag
      )}`;
      process.stdout.write(`Downloading ${REPO.owner}/${REPO.name}@${tag}...\n`);
      await downloadToFile(url, archive);
      process.stdout.write("Extracting...\n");
      const extracted = path.join(tmp, "src");
      await extractTarGz(archive, extracted);
      repoRoot = extracted;
    } else {
      process.stdout.write("Offline mode: updating from local package contents.\n");
    }

    const fromStatus = readInstalledModuleNamesFromStatus(installDir);
    const installed = fromStatus || (await detectInstalledModuleNames(config, repoRoot, installDir));
    const toUpdate = installed.filter((name) => Object.prototype.hasOwnProperty.call(mods, name));

    if (!toUpdate.length) {
      process.stdout.write(`No installed modules found in ${installDir}.\n`);
      return;
    }

    if (dryRun) {
      for (const name of toUpdate) process.stdout.write(`module:${name}\n`);
      return;
    }

    await fs.promises.mkdir(installDir, { recursive: true });
    const installState = { wrapperInstalled: false };

    async function ensureWrapperInstalled() {
      if (installState.wrapperInstalled) return;
      process.stdout.write("Installing codeagent-wrapper...\n");
      await runInstallSh(repoRoot, installDir, tag);
      installState.wrapperInstalled = true;
    }

    for (const name of toUpdate) {
      if (WRAPPER_REQUIRED_MODULES.has(name)) await ensureWrapperInstalled();
      process.stdout.write(`Updating module: ${name}\n`);
      const r = await applyModule(name, config, repoRoot, installDir, true, tag, installState);
      upsertModuleStatus(installDir, r);
    }
  } finally {
    if (tmp) await rmTree(tmp);
  }
}

function buildItems(config, skills) {
  const items = [{ id: "codeagent-wrapper", label: "codeagent-wrapper", kind: "wrapper" }];

  const modules = (config && config.modules) || {};
  for (const [name, mod] of Object.entries(modules)) {
    const desc = mod && typeof mod.description === "string" ? mod.description : "";
    items.push({
      id: `module:${name}`,
      label: `module:${name}${desc ? ` - ${desc}` : ""}`,
      kind: "module",
      moduleName: name,
    });
  }

  for (const s of skills) {
    items.push({ id: `skill:${s}`, label: `skill:${s}`, kind: "skill", skillName: s });
  }

  return items;
}

function clearScreen() {
  process.stdout.write("\x1b[2J\x1b[H");
}

async function promptMultiSelect(items, title) {
  if (!process.stdin.isTTY) {
    throw new Error("No TTY. Use --list or run in an interactive terminal.");
  }

  let idx = 0;
  const selected = new Set();

  readline.emitKeypressEvents(process.stdin);
  process.stdin.setRawMode(true);

  function render() {
    clearScreen();
    process.stdout.write(`${title}\n`);
    process.stdout.write("↑↓ move  Space toggle  Enter confirm  q quit\n\n");
    for (let i = 0; i < items.length; i++) {
      const it = items[i];
      const cursor = i === idx ? ">" : " ";
      const box = selected.has(it.id) ? "[x]" : "[ ]";
      process.stdout.write(`${cursor} ${box} ${it.label}\n`);
    }
  }

  function cleanup() {
    process.stdin.setRawMode(false);
    process.stdin.removeListener("keypress", onKey);
    process.stdin.pause();
  }

  function onKey(_, key) {
    if (!key) return;
    if (key.name === "c" && key.ctrl) {
      cleanup();
      process.exit(130);
    }
    if (key.name === "q") {
      cleanup();
      process.exit(0);
    }
    if (key.name === "up") idx = (idx - 1 + items.length) % items.length;
    else if (key.name === "down") idx = (idx + 1) % items.length;
    else if (key.name === "space") {
      const id = items[idx].id;
      if (selected.has(id)) selected.delete(id);
      else selected.add(id);
    } else if (key.name === "return") {
      cleanup();
      clearScreen();
      const picked = items.filter((it) => selected.has(it.id));
      return resolvePick(picked);
    }
    render();
  }

  let resolvePick;
  const result = new Promise((resolve) => {
    resolvePick = resolve;
  });

  process.stdin.on("keypress", onKey);
  render();
  return result;
}

function isZeroBlock(b) {
  for (let i = 0; i < b.length; i++) if (b[i] !== 0) return false;
  return true;
}

function tarString(b, start, len) {
  return b
    .toString("utf8", start, start + len)
    .replace(/\0.*$/, "")
    .trim();
}

function tarOctal(b, start, len) {
  const s = tarString(b, start, len);
  if (!s) return 0;
  return parseInt(s, 8) || 0;
}

function safePosixPath(p) {
  const norm = path.posix.normalize(p);
  if (norm.startsWith("/") || norm.startsWith("..") || norm.includes("/../")) {
    throw new Error(`Unsafe path in archive: ${p}`);
  }
  return norm;
}

async function extractTarGz(archivePath, destDir) {
  await fs.promises.mkdir(destDir, { recursive: true });
  const gunzip = zlib.createGunzip();
  const stream = fs.createReadStream(archivePath).pipe(gunzip);

  let buf = Buffer.alloc(0);
  let file = null;
  let pad = 0;
  let zeroBlocks = 0;

  for await (const chunk of stream) {
    buf = Buffer.concat([buf, chunk]);
    while (true) {
      if (pad) {
        if (buf.length < pad) break;
        buf = buf.slice(pad);
        pad = 0;
      }

      if (!file) {
        if (buf.length < 512) break;
        const header = buf.slice(0, 512);
        buf = buf.slice(512);

        if (isZeroBlock(header)) {
          zeroBlocks++;
          if (zeroBlocks >= 2) return;
          continue;
        }
        zeroBlocks = 0;

        const name = tarString(header, 0, 100);
        const prefix = tarString(header, 345, 155);
        const full = prefix ? `${prefix}/${name}` : name;
        const size = tarOctal(header, 124, 12);
        const mode = tarOctal(header, 100, 8);
        const typeflag = header[156];

        const rel = safePosixPath(full.split("/").slice(1).join("/"));
        if (!rel || rel === ".") {
          file = null;
          pad = 0;
          continue;
        }

        const outPath = path.join(destDir, ...rel.split("/"));
        if (typeflag === 53) {
          await fs.promises.mkdir(outPath, { recursive: true });
          if (mode) await fs.promises.chmod(outPath, mode);
          file = null;
          pad = 0;
          continue;
        }

        file = { outPath, size, remaining: size, chunks: [], mode };
        if (size === 0) {
          await fs.promises.mkdir(path.dirname(outPath), { recursive: true });
          await fs.promises.writeFile(outPath, Buffer.alloc(0));
          if (mode) await fs.promises.chmod(outPath, mode);
          file = null;
          pad = 0;
        }
        continue;
      }

      if (buf.length < file.remaining) {
        file.chunks.push(buf);
        file.remaining -= buf.length;
        buf = Buffer.alloc(0);
        break;
      }

      file.chunks.push(buf.slice(0, file.remaining));
      buf = buf.slice(file.remaining);
      file.remaining = 0;

      await fs.promises.mkdir(path.dirname(file.outPath), { recursive: true });
      await fs.promises.writeFile(file.outPath, Buffer.concat(file.chunks));
      if (file.mode) await fs.promises.chmod(file.outPath, file.mode);

      pad = (512 - (file.size % 512)) % 512;
      file = null;
    }
  }
}

async function copyFile(src, dst, force) {
  if (!force && fs.existsSync(dst)) return false;
  await fs.promises.mkdir(path.dirname(dst), { recursive: true });
  await fs.promises.copyFile(src, dst);
  const st = await fs.promises.stat(src);
  await fs.promises.chmod(dst, st.mode);
  return true;
}

async function copyDirRecursive(src, dst, force) {
  if (fs.existsSync(dst) && !force) return;
  await fs.promises.mkdir(dst, { recursive: true });

  const entries = await fs.promises.readdir(src, { withFileTypes: true });
  for (const e of entries) {
    const s = path.join(src, e.name);
    const d = path.join(dst, e.name);
    if (e.isDirectory()) await copyDirRecursive(s, d, force);
    else if (e.isFile()) await copyFile(s, d, force);
  }
}

async function mergeDir(src, installDir, force) {
  const installed = [];
  const subdirs = await fs.promises.readdir(src, { withFileTypes: true });
  for (const d of subdirs) {
    if (!d.isDirectory()) continue;
    const srcSub = path.join(src, d.name);
    const dstSub = path.join(installDir, d.name);
    await fs.promises.mkdir(dstSub, { recursive: true });
    const entries = await fs.promises.readdir(srcSub, { withFileTypes: true });
    for (const e of entries) {
      if (!e.isFile()) continue;
      const didCopy = await copyFile(path.join(srcSub, e.name), path.join(dstSub, e.name), force);
      if (didCopy) installed.push(`${d.name}/${e.name}`);
    }
  }
  return installed;
}

function runInstallSh(repoRoot, installDir, tag) {
  return new Promise((resolve, reject) => {
    const cmd = process.platform === "win32" ? "cmd.exe" : "bash";
    const args = process.platform === "win32" ? ["/c", "install.bat"] : ["install.sh"];
    const env = { ...process.env, INSTALL_DIR: installDir };
    if (tag) env.CODEAGENT_WRAPPER_VERSION = tag;
    const p = spawn(cmd, args, {
      cwd: repoRoot,
      stdio: "inherit",
      env,
    });
    p.on("exit", (code) => {
      if (code === 0) resolve();
      else reject(new Error(`install script failed (exit ${code})`));
    });
  });
}

async function rmTree(p) {
  if (!fs.existsSync(p)) return;
  if (fs.promises.rm) {
    await fs.promises.rm(p, { recursive: true, force: true });
    return;
  }
  await fs.promises.rmdir(p, { recursive: true });
}

function defaultModelsConfig() {
  return {
    default_backend: "codex",
    default_model: "gpt-4.1",
    backends: {},
    agents: {},
  };
}

function mergeModuleAgentsToModels(moduleName, mod, repoRoot) {
  const moduleAgents = mod && mod.agents;
  if (!isPlainObject(moduleAgents) || !Object.keys(moduleAgents).length) return false;

  const modelsPath = path.join(os.homedir(), ".codeagent", "models.json");
  fs.mkdirSync(path.dirname(modelsPath), { recursive: true });

  let models;
  if (fs.existsSync(modelsPath)) {
    models = JSON.parse(fs.readFileSync(modelsPath, "utf8"));
  } else {
    const templatePath = path.join(repoRoot, "templates", "models.json.example");
    if (fs.existsSync(templatePath)) {
      models = JSON.parse(fs.readFileSync(templatePath, "utf8"));
      if (!isPlainObject(models)) models = defaultModelsConfig();
      models.agents = {};
    } else {
      models = defaultModelsConfig();
    }
  }

  if (!isPlainObject(models)) models = defaultModelsConfig();
  if (!isPlainObject(models.agents)) models.agents = {};

  let modified = false;
  for (const [agentName, agentCfg] of Object.entries(moduleAgents)) {
    if (!isPlainObject(agentCfg)) continue;
    const existing = models.agents[agentName];
    const canOverwrite = !isPlainObject(existing) || Object.prototype.hasOwnProperty.call(existing, "__module__");
    if (!canOverwrite) continue;
    const next = { ...agentCfg, __module__: moduleName };
    if (!deepEqual(existing, next)) {
      models.agents[agentName] = next;
      modified = true;
    }
  }

  if (modified) fs.writeFileSync(modelsPath, JSON.stringify(models, null, 2) + "\n", "utf8");
  return modified;
}

async function applyModule(moduleName, config, repoRoot, installDir, force, tag, installState) {
  const mod = config && config.modules && config.modules[moduleName];
  if (!mod) throw new Error(`Unknown module: ${moduleName}`);
  const ops = Array.isArray(mod.operations) ? mod.operations : [];
  const result = {
    module: moduleName,
    status: "success",
    operations: [],
    installed_at: new Date().toISOString(),
  };
  const mergeDirFiles = [];

  for (const op of ops) {
    const type = op && op.type;
    try {
      if (type === "copy_file") {
        await copyFile(path.join(repoRoot, op.source), path.join(installDir, op.target), force);
      } else if (type === "copy_dir") {
        await copyDirRecursive(path.join(repoRoot, op.source), path.join(installDir, op.target), force);
      } else if (type === "merge_dir") {
        mergeDirFiles.push(...(await mergeDir(path.join(repoRoot, op.source), installDir, force)));
      } else if (type === "run_command") {
        const cmd = typeof op.command === "string" ? op.command.trim() : "";
        if (cmd !== "bash install.sh") {
          throw new Error(`Refusing run_command: ${cmd || "(empty)"}`);
        }
        if (installState && installState.wrapperInstalled) {
          result.operations.push({ type, status: "success", skipped: true });
          continue;
        }
        await runInstallSh(repoRoot, installDir, tag);
        if (installState) installState.wrapperInstalled = true;
      } else {
        throw new Error(`Unsupported operation type: ${type}`);
      }
      result.operations.push({ type, status: "success" });
    } catch (err) {
      result.status = "failed";
      result.operations.push({
        type,
        status: "failed",
        error: err && err.message ? err.message : String(err),
      });
      throw err;
    }
  }

  if (mergeDirFiles.length) result.merge_dir_files = mergeDirFiles;

  try {
    if (mergeModuleHooks(moduleName, mod, installDir)) {
      result.has_hooks = true;
      result.operations.push({ type: "merge_hooks", status: "success" });
    }
  } catch (err) {
    result.operations.push({
      type: "merge_hooks",
      status: "failed",
      error: err && err.message ? err.message : String(err),
    });
  }

  try {
    if (mergeModuleAgentsToModels(moduleName, mod, repoRoot)) {
      result.has_agents = true;
      result.operations.push({ type: "merge_agents", status: "success" });
    }
  } catch (err) {
    result.operations.push({
      type: "merge_agents",
      status: "failed",
      error: err && err.message ? err.message : String(err),
    });
  }

  return result;
}

async function tryRemoveEmptyDir(p) {
  try {
    const entries = await fs.promises.readdir(p);
    if (!entries.length) await fs.promises.rmdir(p);
  } catch {
    // ignore
  }
}

async function removePathIfExists(p) {
  if (!fs.existsSync(p)) return;
  const st = await fs.promises.lstat(p);
  if (st.isDirectory()) {
    await rmTree(p);
    return;
  }
  try {
    await fs.promises.unlink(p);
  } catch (err) {
    if (!err || err.code !== "ENOENT") throw err;
  }
}

async function uninstallModule(moduleName, config, repoRoot, installDir, dryRun) {
  const mod = config && config.modules && config.modules[moduleName];
  if (!mod) throw new Error(`Unknown module: ${moduleName}`);
  const ops = Array.isArray(mod.operations) ? mod.operations : [];
  const status = loadInstalledStatus(installDir);
  const moduleStatus = (status.modules && status.modules[moduleName]) || {};
  const recordedMerge = Array.isArray(moduleStatus.merge_dir_files) ? moduleStatus.merge_dir_files : null;

  for (const op of ops) {
    const type = op && op.type;
    if (type === "copy_file" || type === "copy_dir") {
      const target = typeof op.target === "string" ? op.target : "";
      if (!target) continue;
      const p = path.join(installDir, target);
      if (dryRun) process.stdout.write(`- remove ${p}\n`);
      else await removePathIfExists(p);
      continue;
    }

    if (type !== "merge_dir") continue;
    const source = typeof op.source === "string" ? op.source : "";
    if (!source) continue;

    if (recordedMerge && recordedMerge.length) {
      for (const rel of recordedMerge) {
        const parts = String(rel).split("/").filter(Boolean);
        if (parts.includes("..")) continue;
        const p = path.join(installDir, ...parts);
        if (dryRun) process.stdout.write(`- remove ${p}\n`);
        else {
          await removePathIfExists(p);
          await tryRemoveEmptyDir(path.dirname(p));
        }
      }
      continue;
    }

    const srcDir = path.join(repoRoot, source);
    if (!(await dirExists(srcDir))) continue;
    const subdirs = await fs.promises.readdir(srcDir, { withFileTypes: true });
    for (const d of subdirs) {
      if (!d.isDirectory()) continue;
      const srcSub = path.join(srcDir, d.name);
      const entries = await fs.promises.readdir(srcSub, { withFileTypes: true });
      for (const e of entries) {
        if (!e.isFile()) continue;
        const dst = path.join(installDir, d.name, e.name);
        if (!fs.existsSync(dst)) continue;
        try {
          const [srcBuf, dstBuf] = await Promise.all([
            fs.promises.readFile(path.join(srcSub, e.name)),
            fs.promises.readFile(dst),
          ]);
          if (Buffer.compare(srcBuf, dstBuf) !== 0) continue;
        } catch {
          continue;
        }
        if (dryRun) process.stdout.write(`- remove ${dst}\n`);
        else {
          await removePathIfExists(dst);
          await tryRemoveEmptyDir(path.dirname(dst));
        }
      }
    }
  }

  if (dryRun) return;
  unmergeHooksFromSettings(moduleName, installDir);
  deleteModuleStatus(installDir, moduleName);
}

async function installDefaultConfigs(installDir, repoRoot) {
  try {
    const claudeMdTarget = path.join(installDir, "CLAUDE.md");
    const claudeMdSrc = path.join(repoRoot, "memorys", "CLAUDE.md");
    if (!fs.existsSync(claudeMdTarget) && fs.existsSync(claudeMdSrc)) {
      await fs.promises.copyFile(claudeMdSrc, claudeMdTarget);
      process.stdout.write(`Installed CLAUDE.md to ${claudeMdTarget}\n`);
    }
  } catch (err) {
    process.stderr.write(`Warning: could not install default configs: ${err.message}\n`);
  }
}

function printPostInstallInfo(installDir) {
  process.stdout.write("\n");

  // Check codeagent-wrapper version
  const wrapperBin = path.join(installDir, "bin", "codeagent-wrapper");
  let wrapperVersion = null;
  try {
    const r = spawnSync(wrapperBin, ["--version"], { timeout: 5000 });
    if (r.status === 0 && r.stdout) {
      wrapperVersion = r.stdout.toString().trim();
    }
  } catch {}

  // Check PATH
  const binDir = path.join(installDir, "bin");
  const envPath = process.env.PATH || "";
  const pathOk = envPath.split(path.delimiter).some((p) => {
    try { return fs.realpathSync(p) === fs.realpathSync(binDir); } catch { return p === binDir; }
  });

  // Check backend CLIs
  const whichCmd = process.platform === "win32" ? "where" : "which";
  const backends = ["codex", "claude", "gemini", "opencode"];
  const detected = {};
  for (const name of backends) {
    try {
      const r = spawnSync(whichCmd, [name], { timeout: 3000 });
      detected[name] = r.status === 0;
    } catch {
      detected[name] = false;
    }
  }

  process.stdout.write("Setup Complete!\n");
  process.stdout.write(`  codeagent-wrapper: ${wrapperVersion || "(not found)"} ${wrapperVersion ? "✓" : "✗"}\n`);
  process.stdout.write(`  PATH: ${binDir} ${pathOk ? "✓" : "✗ (not in PATH)"}\n`);
  process.stdout.write("\nBackend CLIs detected:\n");
  process.stdout.write("  " + backends.map((b) => `${b} ${detected[b] ? "✓" : "✗"}`).join("  |  ") + "\n");
  process.stdout.write("\nNext steps:\n");
  process.stdout.write("  1. Configure API keys in ~/.codeagent/models.json\n");
  process.stdout.write('  2. Try: /do "your first task"\n');
  process.stdout.write("\n");
}

async function installSelected(picks, tag, config, installDir, force, dryRun) {
  const needRepo = picks.some((p) => p.kind !== "wrapper");
  const needWrapper = picks.some((p) => p.kind === "wrapper");

  if (dryRun) {
    for (const p of picks) process.stdout.write(`- ${p.id}\n`);
    return;
  }

  const tmp = path.join(
    os.tmpdir(),
    `myclaude-${Date.now()}-${crypto.randomBytes(4).toString("hex")}`
  );
  await fs.promises.mkdir(tmp, { recursive: true });

  try {
    let repoRoot = repoRootFromHere();
    if (needRepo || needWrapper) {
      if (tag) {
        const archive = path.join(tmp, "src.tgz");
        const url = `https://codeload.github.com/${REPO.owner}/${REPO.name}/tar.gz/refs/tags/${encodeURIComponent(
          tag
        )}`;
        process.stdout.write(`Downloading ${REPO.owner}/${REPO.name}@${tag}...\n`);
        await downloadToFile(url, archive);
        process.stdout.write("Extracting...\n");
        const extracted = path.join(tmp, "src");
        await extractTarGz(archive, extracted);
        repoRoot = extracted;
      } else {
        process.stdout.write("Offline mode: installing from local package contents.\n");
      }
    }

    await fs.promises.mkdir(installDir, { recursive: true });
    const installState = { wrapperInstalled: false };

    async function ensureWrapperInstalled() {
      if (installState.wrapperInstalled) return;
      process.stdout.write("Installing codeagent-wrapper...\n");
      await runInstallSh(repoRoot, installDir, tag);
      installState.wrapperInstalled = true;
    }

    for (const p of picks) {
      if (p.kind === "wrapper") {
        await ensureWrapperInstalled();
        continue;
      }
      if (p.kind === "module") {
        if (WRAPPER_REQUIRED_MODULES.has(p.moduleName)) await ensureWrapperInstalled();
        process.stdout.write(`Installing module: ${p.moduleName}\n`);
        const r = await applyModule(
          p.moduleName,
          config,
          repoRoot,
          installDir,
          force,
          tag,
          installState
        );
        upsertModuleStatus(installDir, r);
        continue;
      }
      if (p.kind === "skill") {
        if (WRAPPER_REQUIRED_SKILLS.has(p.skillName)) await ensureWrapperInstalled();
        process.stdout.write(`Installing skill: ${p.skillName}\n`);
        await copyDirRecursive(
          path.join(repoRoot, "skills", p.skillName),
          path.join(installDir, "skills", p.skillName),
          force
        );
      }
    }

    await installDefaultConfigs(installDir, repoRoot);
    printPostInstallInfo(installDir);
  } finally {
    await rmTree(tmp);
  }
}

async function main() {
  const args = parseArgs(process.argv.slice(2));
  if (args.help) {
    printHelp();
    return;
  }

  const installDir = expandHome(args.installDir);
  if (args.command !== "install" && args.command !== "uninstall") {
    throw new Error(`Unknown command: ${args.command}`);
  }
  if (args.list && args.update) throw new Error("Cannot combine --list and --update");

  if (args.command === "uninstall") {
    const config = readLocalConfig();
    const repoRoot = repoRootFromHere();
    const fromStatus = readInstalledModuleNamesFromStatus(installDir);
    const installed = fromStatus || (await detectInstalledModuleNames(config, repoRoot, installDir));
    const installedSet = new Set(installed);

    let toRemove = [];
    if (args.module) {
      const v = String(args.module).trim();
      if (v.toLowerCase() === "all") {
        toRemove = installed;
      } else {
        toRemove = v
          .split(",")
          .map((s) => s.trim())
          .filter(Boolean);
      }
    } else {
      const modules = (config && config.modules) || {};
      const items = [];
      for (const [name, mod] of Object.entries(modules)) {
        if (!installedSet.has(name)) continue;
        const desc = mod && typeof mod.description === "string" ? mod.description : "";
        items.push({
          id: `module:${name}`,
          label: `module:${name}${desc ? ` - ${desc}` : ""}`,
          kind: "module",
          moduleName: name,
        });
      }
      if (!items.length) {
        process.stdout.write(`No installed modules found in ${installDir}.\n`);
        return;
      }
      const picks = await promptMultiSelect(items, "myclaude uninstall");
      toRemove = picks.map((p) => p.moduleName);
    }

    toRemove = toRemove.filter((m) => installedSet.has(m));
    if (!toRemove.length) {
      process.stdout.write("Nothing selected.\n");
      return;
    }

    if (!args.yes && !args.dryRun) {
      if (!process.stdin.isTTY) {
        throw new Error("No TTY. Use -y/--yes to skip confirmation.");
      }
      const rl = readline.createInterface({ input: process.stdin, output: process.stdout });
      const answer = await new Promise((resolve) => rl.question("Confirm uninstall? (y/N): ", resolve));
      rl.close();
      if (String(answer).trim().toLowerCase() !== "y") {
        process.stdout.write("Cancelled.\n");
        return;
      }
    }

    for (const name of toRemove) {
      process.stdout.write(`Uninstalling module: ${name}\n`);
      await uninstallModule(name, config, repoRoot, installDir, args.dryRun);
    }
    process.stdout.write("Done.\n");
    return;
  }

  let tag = args.tag;
  if (!tag) {
    try {
      tag = await withTimeout(fetchLatestTag(), 5000, "fetch latest tag");
    } catch {
      tag = null;
    }
  }

  let config = null;
  let skills = [];
  if (tag) {
    try {
      [config, skills] = await withTimeout(
        Promise.all([fetchRemoteConfig(tag), fetchRemoteSkills(tag)]),
        8000,
        "fetch config/skills"
      );
    } catch {
      config = null;
      skills = [];
    }
  }

  if (!config) config = readLocalConfig();
  if (!skills.length) skills = listLocalSkills();

  if (args.update) {
    await updateInstalledModules(installDir, tag, config, args.dryRun);
    process.stdout.write("Done.\n");
    return;
  }

  const items = buildItems(config, skills);
  if (args.list) {
    for (const it of items) process.stdout.write(`${it.id}\n`);
    return;
  }

  const title = tag ? `myclaude installer (latest: ${tag})` : "myclaude installer (offline mode)";
  const picks = await promptMultiSelect(items, title);
  if (!picks.length) {
    process.stdout.write("Nothing selected.\n");
    return;
  }

  await installSelected(picks, tag, config, installDir, args.force, args.dryRun);
  process.stdout.write("Done.\n");
}

main().catch((err) => {
  process.stderr.write(`ERROR: ${err && err.message ? err.message : String(err)}\n`);
  process.exit(1);
});
