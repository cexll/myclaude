#!/usr/bin/env node

const fs = require("fs");
const path = require("path");

function readInput() {
  const raw = fs.readFileSync(0, "utf8").trim();
  if (!raw) return {};
  try {
    return JSON.parse(raw);
  } catch (_err) {
    return {};
  }
}

function extractPrompt(payload) {
  return (
    payload.prompt ||
    payload.text ||
    payload.userPrompt ||
    (payload.data && payload.data.prompt) ||
    ""
  ).toString();
}

function loadRules() {
  const rulesPath = path.resolve(__dirname, "../skills/skill-rules.json");
  try {
    const file = fs.readFileSync(rulesPath, "utf8");
    return JSON.parse(file);
  } catch (_err) {
    return { skills: {} };
  }
}

function matchSkill(prompt, rule, skillName) {
  const triggers = (rule && rule.promptTriggers) || {};
  const keywords = [...(triggers.keywords || []), skillName].filter(Boolean);
  const patterns = triggers.intentPatterns || [];
  const promptLower = prompt.toLowerCase();

  const keyword = keywords.find((k) => promptLower.includes(k.toLowerCase()));
  if (keyword) {
    return `命中关键词 "${keyword}"`;
  }

  for (const pattern of patterns) {
    try {
      if (new RegExp(pattern, "i").test(prompt)) {
        return `命中模式 /${pattern}/`;
      }
    } catch (_err) {
      continue;
    }
  }
  return null;
}

function main() {
  const payload = readInput();
  const prompt = extractPrompt(payload);
  if (!prompt.trim()) {
    console.log(JSON.stringify({ suggestedSkills: [] }, null, 2));
    return;
  }

  const rules = loadRules();
  const suggestions = [];

  for (const [name, rule] of Object.entries(rules.skills || {})) {
    const matchReason = matchSkill(prompt, rule, name);
    if (matchReason) {
      suggestions.push({
        skill: name,
        enforcement: rule.enforcement || "suggest",
        priority: rule.priority || "normal",
        reason: matchReason
      });
    }
  }

  console.log(JSON.stringify({ suggestedSkills: suggestions }, null, 2));
}

main();
