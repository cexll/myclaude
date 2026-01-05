# Security Scan Prompt for Skills

Use this prompt template to analyze skill content for security vulnerabilities before installation.

## Prompt Template

```
You are a security expert analyzing a Claude skill for potential security risks.

Analyze the following skill content for security vulnerabilities:

**Skill Name:** {skill_name}
**Skill Content:**
{skill_content}

## Security Analysis Criteria

Examine the skill for the following security concerns:

### 1. Malicious Command Execution
- Detect `eval()`, `exec()`, `subprocess` with `shell=True`
- Identify arbitrary code execution patterns
- Check for command injection vulnerabilities

### 2. Backdoor Detection
- Look for obfuscated code (base64, hex encoding)
- Identify suspicious network requests to unknown domains
- Detect file hash patterns matching known malware
- Check for hidden data exfiltration mechanisms

### 3. Credential Theft
- Detect attempts to access environment variables containing secrets
- Identify file operations on sensitive paths (~/.ssh, ~/.aws, ~/.netrc)
- Check for credential harvesting patterns
- Look for keylogging or clipboard monitoring

### 4. Unauthorized Network Access
- Identify external network requests
- Check for connections to suspicious domains (pastebin, ngrok, bit.ly, etc.)
- Detect data exfiltration via HTTP/HTTPS
- Look for reverse shell patterns

### 5. File System Abuse
- Detect destructive file operations (rm -rf, shutil.rmtree)
- Identify unauthorized file writes to system directories
- Check for file permission modifications
- Look for attempts to modify critical system files

### 6. Privilege Escalation
- Detect sudo or privilege escalation attempts
- Identify attempts to modify system configurations
- Check for container escape patterns

### 7. Supply Chain Attacks
- Identify suspicious package installations
- Detect dynamic imports from untrusted sources
- Check for dependency confusion attacks

## Output Format

Provide your analysis in the following format:

**Security Status:** [SAFE / WARNING / DANGEROUS]

**Risk Level:** [LOW / MEDIUM / HIGH / CRITICAL]

**Findings:**
1. [Category]: [Description]
   - File: [filename:line_number]
   - Severity: [LOW/MEDIUM/HIGH/CRITICAL]
   - Details: [Explanation]
   - Recommendation: [How to fix or mitigate]

**Summary:**
[Brief summary of the security assessment]

**Recommendation:**
[APPROVE / REJECT / APPROVE_WITH_WARNINGS]

## Decision Criteria

- **APPROVE**: No security issues found, safe to install
- **APPROVE_WITH_WARNINGS**: Minor concerns but generally safe, user should be aware
- **REJECT**: Critical security issues found, do not install

Be thorough but avoid false positives. Consider the context and legitimate use cases.
```

## Example Analysis

### Safe Skill Example

```
**Security Status:** SAFE
**Risk Level:** LOW
**Findings:** None
**Summary:** The skill contains only documentation and safe tool usage instructions. No executable code or suspicious patterns detected.
**Recommendation:** APPROVE
```

### Suspicious Skill Example

```
**Security Status:** WARNING
**Risk Level:** MEDIUM
**Findings:**
1. [Network Access]: External HTTP request detected
   - File: scripts/helper.py:42
   - Severity: MEDIUM
   - Details: Script makes HTTP request to api.example.com without user consent
   - Recommendation: Review the API endpoint and ensure it's legitimate

**Summary:** The skill makes external network requests that should be reviewed.
**Recommendation:** APPROVE_WITH_WARNINGS
```

### Dangerous Skill Example

```
**Security Status:** DANGEROUS
**Risk Level:** CRITICAL
**Findings:**
1. [Command Injection]: Arbitrary command execution detected
   - File: scripts/malicious.py:15
   - Severity: CRITICAL
   - Details: Uses subprocess.call() with shell=True and unsanitized input
   - Recommendation: Do not install this skill

2. [Data Exfiltration]: Suspicious network request
   - File: scripts/malicious.py:28
   - Severity: HIGH
   - Details: Sends data to pastebin.com without user knowledge
   - Recommendation: This appears to be a data exfiltration attempt

**Summary:** This skill contains critical security vulnerabilities including command injection and data exfiltration. It appears to be malicious.
**Recommendation:** REJECT
```
