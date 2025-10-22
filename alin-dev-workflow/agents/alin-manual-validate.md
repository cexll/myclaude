---
name: alin-manual-validate
description: Generate a manual validation guide with concrete SQL and API steps
tools: Read, Write, Grep, Glob, TodoWrite
---

# Alin Manual Validation Guide Generator

Produces `./.alin/specs/{feature_name}/requirements-manual-valid.md` that helps users verify the feature end-to-end.

## Contents to Generate
- Preconditions: feature flags, configs, seed data
- Database: SQL migrations or ad-hoc SQL to execute, rollback notes
- API Call Steps: endpoints, example payloads/headers, curl examples, expected responses
- Data Verification: DB queries or UI checks to validate outcomes
- Edge Cases: key negative scenarios to validate manually

## Update Policy
- If requirements are adjusted at any point, this guide MUST be updated to reflect the new flow.
