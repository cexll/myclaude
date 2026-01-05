---
name: skill-install
description: Install Claude skills from GitHub repositories with automated security scanning. Triggers when users want to install skills from a GitHub URL, need to browse available skills in a repository, or want to safely add new skills to their Claude environment.
---

# Skill Install

## Overview

Install Claude skills from GitHub repositories with built-in security scanning to protect against malicious code, backdoors, and vulnerabilities.

## When to Use

Trigger this skill when the user:
- Provides a GitHub repository URL and wants to install skills
- Asks to "install skills from GitHub"
- Wants to browse and select skills from a repository
- Needs to add new skills to their Claude environment

## Workflow

### Step 1: Parse GitHub URL

Accept a GitHub repository URL from the user. The URL should point to a repository containing a `skills/` directory.

Supported URL formats:
- `https://github.com/user/repo`
- `https://github.com/user/repo/tree/main/skills`
- `https://github.com/user/repo/tree/branch-name/skills`

Extract:
- Repository owner
- Repository name
- Branch (default to `main` if not specified)

### Step 2: Fetch Skills List

Use the WebFetch tool to retrieve the skills directory listing from GitHub.

GitHub API endpoint pattern:
```
https://api.github.com/repos/{owner}/{repo}/contents/skills?ref={branch}
```

Parse the response to extract:
- Skill directory names
- Each skill should be a subdirectory containing a SKILL.md file

### Step 3: Present Skills to User

Use the AskUserQuestion tool to let the user select which skills to install.

Set `multiSelect: true` to allow multiple selections.

Present each skill with:
- Skill name (directory name)
- Brief description (if available from SKILL.md frontmatter)

### Step 4: Fetch Skill Content

For each selected skill, fetch all files in the skill directory:

1. Get the file tree for the skill directory
2. Download all files (SKILL.md, scripts/, references/, assets/)
3. Store the complete skill content for security analysis

Use WebFetch with GitHub API:
```
https://api.github.com/repos/{owner}/{repo}/contents/skills/{skill_name}?ref={branch}
```

For each file, fetch the raw content:
```
https://raw.githubusercontent.com/{owner}/{repo}/{branch}/skills/{skill_name}/{file_path}
```

### Step 5: Security Scan

**CRITICAL:** Before installation, perform a thorough security analysis of each skill.

Read the security scan prompt template from `references/security_scan_prompt.md` and apply it to analyze the skill content.

Examine for:
1. **Malicious Command Execution** - eval, exec, subprocess with shell=True
2. **Backdoor Detection** - obfuscated code, suspicious network requests
3. **Credential Theft** - accessing ~/.ssh, ~/.aws, environment variables
4. **Unauthorized Network Access** - external requests to suspicious domains
5. **File System Abuse** - destructive operations, unauthorized writes
6. **Privilege Escalation** - sudo attempts, system modifications
7. **Supply Chain Attacks** - suspicious package installations

Output the security analysis with:
- Security Status: SAFE / WARNING / DANGEROUS
- Risk Level: LOW / MEDIUM / HIGH / CRITICAL
- Detailed findings with file locations and severity
- Recommendation: APPROVE / APPROVE_WITH_WARNINGS / REJECT

### Step 6: User Decision

Based on the security scan results:

**If SAFE (APPROVE):**
- Proceed directly to installation

**If WARNING (APPROVE_WITH_WARNINGS):**
- Display the security warnings to the user
- Use AskUserQuestion to confirm: "Security warnings detected. Do you want to proceed with installation?"
- Options: "Yes, install anyway" / "No, skip this skill"

**If DANGEROUS (REJECT):**
- Display the critical security issues
- Refuse to install
- Explain why the skill is dangerous
- Do NOT provide an option to override for CRITICAL severity issues

### Step 7: Install Skills

For approved skills, install to `~/.claude/skills/`:

1. Create the skill directory: `~/.claude/skills/{skill_name}/`
2. Write all skill files maintaining the directory structure
3. Ensure proper file permissions (executable for scripts)
4. Verify SKILL.md exists and has valid frontmatter

Use the Write tool to create files.

### Step 8: Confirmation

After installation, provide a summary:
- List of successfully installed skills
- List of skipped skills (if any) with reasons
- Location: `~/.claude/skills/`
- Next steps: "The skills are now available. Restart Claude or use them directly."

## Example Usage

**User:** "Install skills from https://github.com/example/claude-skills"

**Assistant:**
1. Fetches skills list from the repository
2. Presents available skills: "skill-a", "skill-b", "skill-c"
3. User selects "skill-a" and "skill-b"
4. Performs security scan on each skill
5. skill-a: SAFE - proceeds to install
6. skill-b: WARNING (makes HTTP request) - asks user for confirmation
7. Installs approved skills to ~/.claude/skills/
8. Confirms: "Successfully installed: skill-a, skill-b"

## Security Notes

- **Never skip security scanning** - Always analyze skills before installation
- **Be conservative** - When in doubt, flag as WARNING and let user decide
- **Critical issues are blocking** - CRITICAL severity findings cannot be overridden
- **Transparency** - Always show users what was found during security scans
- **Sandboxing** - Remind users that skills run with Claude's permissions

## Resources

### references/security_scan_prompt.md

Contains the detailed security analysis prompt template with:
- Complete list of security categories to check
- Output format requirements
- Example analyses for safe, suspicious, and dangerous skills
- Decision criteria for APPROVE/REJECT recommendations

Load this file when performing security scans to ensure comprehensive analysis.
