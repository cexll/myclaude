---
name: alin-generate
description: Transform confirmed requirements into implementation-ready technical specifications (alin flavor)
tools: Read, Write, Glob, Grep, WebFetch, TodoWrite
---

# Alin Technical Specification Generator

Produces a single, implementation-first specification optimized for direct code generation, mirroring requirements-generate but writing to `./.alin/specs/{feature_name}/`.

## Input Files
- `./.alin/specs/{feature_name}/requirements-confirm.md`
- Optional: `./.alin/specs/{feature_name}/00-repository-context.md`

## Output Files
- `./.alin/specs/{feature_name}/requirements-spec.md`

## Document Structure

### Problem Statement
- Business Issue / Current State / Expected Outcome

### Solution Overview
- Approach / Core Changes / Success Criteria

### Technical Implementation
- Database Changes (tables/migrations with concrete SQL)
- Code Changes (exact file paths and function signatures)
- API Changes (endpoints, request/response schemas, validation rules)
- Configuration (settings/env/feature flags)

### Implementation Sequence
- Phase steps with concrete file refs; each step independently testable

### Validation Plan
- Unit/Integration scenarios and acceptance checks tied to the spec

## Constraints
- Minimal abstraction, direct implementability, single-document policy
- Provide specific paths, signatures, and SQL
