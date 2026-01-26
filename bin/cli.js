#!/usr/bin/env node
"use strict";

const crypto = require("crypto");
const fs = require("fs");
const https = require("https");
const os = require("os");
const path = require("path");
const readline = require("readline");
const zlib = require("zlib");
const { spawn } = require("child_process");

const REPO = { owner: "cexll", name: "myclaude" };
const API_HEADERS = {
  "User-Agent": "myclaude-npx",
  Accept: "application/vnd.github+json",
};

function parseArgs(argv) {
  const out = {
    installDir: "~/.claude",
    force: false,
    dryRun: false,
    list: false,
    update: false,
    tag: null,
  };

  for (let i = 0; i < argv.length; i++) {
    const a = argv[i];
    if (a === "--install-dir") out.installDir = argv[++i];
    else if (a === "--force") out.force = true;
    else if (a === "--dry-run") out.dryRun = true;
    else if (a === "--list") out.list = true;
    else if (a === "--update") out.update = true;
    else if (a === "--tag") out.tag = argv[++i];
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
      "",
      "Options:",
      "  --install-dir <path>   Default: ~/.claude",
      "  --force                Overwrite existing files",
      "  --dry-run              Print actions only",
      "  --list                 List installable items and exit",
      "  --update               Update already installed modules",
      "  --tag <tag>            Install a specific GitHub tag",
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
    for (const name of toUpdate) {
      process.stdout.write(`Updating module: ${name}\n`);
      await applyModule(name, config, repoRoot, installDir, true);
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
  if (!force && fs.existsSync(dst)) return;
  await fs.promises.mkdir(path.dirname(dst), { recursive: true });
  await fs.promises.copyFile(src, dst);
  const st = await fs.promises.stat(src);
  await fs.promises.chmod(dst, st.mode);
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
  const subdirs = await fs.promises.readdir(src, { withFileTypes: true });
  for (const d of subdirs) {
    if (!d.isDirectory()) continue;
    const srcSub = path.join(src, d.name);
    const dstSub = path.join(installDir, d.name);
    await fs.promises.mkdir(dstSub, { recursive: true });
    const entries = await fs.promises.readdir(srcSub, { withFileTypes: true });
    for (const e of entries) {
      if (!e.isFile()) continue;
      await copyFile(path.join(srcSub, e.name), path.join(dstSub, e.name), force);
    }
  }
}

function runInstallSh(repoRoot, installDir) {
  return new Promise((resolve, reject) => {
    const cmd = process.platform === "win32" ? "cmd.exe" : "bash";
    const args = process.platform === "win32" ? ["/c", "install.bat"] : ["install.sh"];
    const p = spawn(cmd, args, {
      cwd: repoRoot,
      stdio: "inherit",
      env: { ...process.env, INSTALL_DIR: installDir },
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

async function applyModule(moduleName, config, repoRoot, installDir, force) {
  const mod = config && config.modules && config.modules[moduleName];
  if (!mod) throw new Error(`Unknown module: ${moduleName}`);
  const ops = Array.isArray(mod.operations) ? mod.operations : [];

  for (const op of ops) {
    const type = op && op.type;
    if (type === "copy_file") {
      await copyFile(
        path.join(repoRoot, op.source),
        path.join(installDir, op.target),
        force
      );
    } else if (type === "copy_dir") {
      await copyDirRecursive(
        path.join(repoRoot, op.source),
        path.join(installDir, op.target),
        force
      );
    } else if (type === "merge_dir") {
      await mergeDir(path.join(repoRoot, op.source), installDir, force);
    } else if (type === "run_command") {
      const cmd = typeof op.command === "string" ? op.command.trim() : "";
      if (cmd !== "bash install.sh") {
        throw new Error(`Refusing run_command: ${cmd || "(empty)"}`);
      }
      await runInstallSh(repoRoot, installDir);
    } else {
      throw new Error(`Unsupported operation type: ${type}`);
    }
  }
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
      if (!tag) throw new Error("No tag available to download");
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
    }

    await fs.promises.mkdir(installDir, { recursive: true });

    for (const p of picks) {
      if (p.kind === "wrapper") {
        process.stdout.write("Installing codeagent-wrapper...\n");
        await runInstallSh(repoRoot, installDir);
        continue;
      }
      if (p.kind === "module") {
        process.stdout.write(`Installing module: ${p.moduleName}\n`);
        await applyModule(p.moduleName, config, repoRoot, installDir, force);
        continue;
      }
      if (p.kind === "skill") {
        process.stdout.write(`Installing skill: ${p.skillName}\n`);
        await copyDirRecursive(
          path.join(repoRoot, "skills", p.skillName),
          path.join(installDir, "skills", p.skillName),
          force
        );
      }
    }
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
  if (args.list && args.update) throw new Error("Cannot combine --list and --update");

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
