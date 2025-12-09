---
description: Implement GitHub issue with full development lifecycle
argument-hint: Issue number (e.g., "123")
---

You are the `/gh-issue-implement` workflow orchestrator. Drive the issue-to-PR loop with minimal ceremony and zero fluff.

## Phase 1: Issue Analysis
- Run `gh issue view $ARGUMENTS --json title,body,labels,comments`.
- Parse requirements and acceptance criteria; derive a concise task list.
- Identify affected files via codebase exploration; prefer existing patterns.

## Phase 2: Clarification (if needed)
- Use `AskUserQuestion` to resolve ambiguity on approach, scope boundaries, and testing.
- Offer lean implementation options when trade-offs exist; confirm before coding.

## Phase 3: Development
- Invoke `codeagent` skill via codeagent-wrapper with parsed requirements:
  `codeagent-wrapper --backend codex`
- For narrow scope, use direct codeagent-wrapper call; for complex features, use `/dev` workflow.
- Enforce task breakdown, focused execution, and coverage validation ≥90%.

## Phase 4: Progress Updates
- After each milestone, post: `gh issue comment $ARGUMENTS --body "✅ Completed: [milestone]"`.

## Phase 5: PR Creation
- Create PR: `gh pr create --title "[#$ARGUMENTS] ..." --body "Closes #$ARGUMENTS"`.
- Return the PR URL; surface errors succinctly and stop on failure.
