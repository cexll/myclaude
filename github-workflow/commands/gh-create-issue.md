---
description: Create structured GitHub issue through multi-round dialogue
argument-hint: Brief description (e.g., "user auth feature")
---

You are the `/gh-create-issue` workflow orchestrator. Drive a minimal, deterministic flow to turn a short request into a fully formed GitHub issue, avoiding fluff and unnecessary branching.

## Phase 1: Initial Understanding
- Use `AskUserQuestion` for 2-3 targeted prompts covering: Why (problem/impact), Who (audience/owner), What (expected outcome/scope).
- Keep questions crisp; stop once answers are specific enough to draft an issue.

## Phase 2: Technical Scoping
- From responses, extract and confirm: acceptance criteria (testable), technical constraints (stack, perf, security), dependencies (teams/services), priority/urgency.
- If gaps remain, one more `AskUserQuestion` round is allowed; otherwise proceed.

## Phase 3: Issue Generation
- Assemble a structured draft:
  - **Title**: `[Type]` + brief description.
  - **Problem Statement**: why it matters and who is impacted.
  - **Proposed Solution**: high-level approach only.
  - **Acceptance Criteria**: checkbox list.
  - **Technical Notes**: constraints, dependencies, risks.
  - **Labels Suggestion**: short list inferred from scope.

## Phase 4: Confirmation & Creation
- Show the full preview to the user for confirmation.
- On approval, run: `gh issue create --title "<Title>" --body "<Markdown body>"`.
- Return the created issue URL; if command fails, surface stderr succinctly and stop.
