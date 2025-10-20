# v6 BMAD-METHOD Workflow Analysis

## Executive Summary

This document analyzes the v6 BMAD-METHOD workflow from [bmad-code-org/BMAD-METHOD](https://github.com/bmad-code-org/BMAD-METHOD/blob/v6-alpha/src/modules/bmm/workflows/README.md) and provides recommendations for adopting its key innovations into our current workflow system.

**Analysis Date**: 2025-10-20  
**Current System**: myclaude multi-agent workflow (v3.2)  
**Comparison Target**: BMAD-METHOD v6-alpha  

---

## Key v6 Innovations

### 1. Scale-Adaptive Planning (★★★★★)

**What it is**: Projects automatically route through different workflows based on complexity levels (0-4).

**v6 Approach**:
```
Level 0: Single atomic change → tech-spec only + 1 story
Level 1: 1-10 stories, 1 epic → tech-spec + 2-3 stories  
Level 2: 5-15 stories, 1-2 epics → PRD + tech-spec
Level 3: 12-40 stories, 2-5 epics → PRD + architecture + JIT tech-specs
Level 4: 40+ stories, 5+ epics → PRD + architecture + JIT tech-specs
```

**Current System**: Fixed workflow - always runs PO → Architect → SM → Dev → Review → QA regardless of project size.

**Gap**: We waste effort on small changes by requiring full PRD and architecture docs.

**Recommendation**: **HIGH PRIORITY - Adopt Level System**

Implementation plan:
1. Create `workflow-classifier` agent to assess project complexity
2. Route to appropriate workflow based on level:
   - Level 0-1: Skip PRD, go straight to tech-spec
   - Level 2: Current workflow minus architecture
   - Level 3-4: Current full workflow
3. Add `--level` flag to bmad-pilot for manual override

**Benefits**:
- 80% faster for simple changes (Level 0-1)
- More appropriate documentation overhead
- Better resource allocation

---

### 2. Universal Entry Point - workflow-status (★★★★☆)

**What it is**: Single command that checks project status, guides workflow selection, and recommends next steps.

**v6 Approach**:
```bash
bmad analyst workflow-status
# Checks for existing status file
# If exists: Shows current phase, progress, next action
# If not: Guides to appropriate workflow based on context
```

**Current System**: Users must know which command to run (`/bmad-pilot` vs `/requirements-pilot` vs `/code`).

**Gap**: No centralized status tracking or workflow guidance.

**Recommendation**: **MEDIUM PRIORITY - Create Workflow Hub**

Implementation plan:
1. Create `/workflow-status` command
2. Implement status file at `.claude/workflow-status.md`
3. Auto-detect:
   - Project context (greenfield vs brownfield)
   - Existing artifacts
   - Current workflow phase
4. Provide smart recommendations

**Benefits**:
- Eliminates workflow confusion
- Better onboarding for new users
- Clear progress visibility

---

### 3. Just-In-Time (JIT) Technical Specifications (★★★★★)

**What it is**: Create tech specs one epic at a time during implementation, not all upfront.

**v6 Approach**:
```
FOR each epic in sequence:
    WHEN ready to implement epic:
        Architect: Run tech-spec workflow for THIS epic only
        → Creates tech-spec-epic-N.md
    IMPLEMENT epic completely
    THEN move to next epic
```

**Current System**: Architecture doc created upfront for entire project (Phase 2).

**Gap**: Over-engineering risk - we design everything before learning from implementation.

**Recommendation**: **HIGH PRIORITY - Adopt JIT Architecture**

Implementation plan:
1. Phase 2: Create high-level architecture.md only (system overview, major components)
2. Phase 3 (new): JIT tech-spec generation per epic
   - Command: `/bmad-architect-epic <epic-number>`
   - Input: architecture.md + epic details + learnings from previous epics
   - Output: tech-spec-epic-N.md
3. Update bmad-dev to read current epic's tech spec

**Benefits**:
- Prevents over-engineering
- Incorporates learnings from previous epics
- More adaptive to changes
- Reduces upfront planning paralysis

---

### 4. 4-State Story State Machine (★★★★☆)

**What it is**: Explicit story lifecycle tracking in workflow status file.

**v6 State Machine**:
```
BACKLOG → TODO → IN PROGRESS → DONE

BACKLOG: Ordered list of stories to be drafted
TODO: Single story ready for drafting (or drafted, awaiting approval)
IN PROGRESS: Single story approved for development
DONE: Completed stories with dates and points
```

**Current System**: Sprint plan has stories but no state tracking mechanism.

**Gap**: No visibility into which stories are being worked on, completed, or blocked.

**Recommendation**: **HIGH PRIORITY - Implement State Machine**

Implementation plan:
1. Enhance `03-sprint-plan.md` with state sections:
   ```markdown
   ## Story Backlog
   ### BACKLOG
   - [ ] Story-001: User login
   - [ ] Story-002: Password reset
   
   ### TODO
   - [ ] Story-003: Profile edit (Status: Draft)
   
   ### IN PROGRESS
   - [~] Story-004: Dashboard (Status: Ready)
   
   ### DONE
   - [x] Story-005: Setup (Status: Done) [2025-10-15, 3 points]
   ```

2. Create workflow commands:
   - `/bmad-sm-draft-story` - Moves BACKLOG → TODO, creates story file
   - `/bmad-sm-approve-story` - Moves TODO → IN PROGRESS (after user review)
   - `/bmad-dev-complete-story` - Moves IN PROGRESS → DONE (after DoD check)

3. Agents read status file instead of searching for "next story"

**Benefits**:
- Clear progress visibility
- No ambiguity on what to work on next
- Prevents duplicate work
- Historical tracking with dates and points

---

### 5. Dynamic Expertise Injection - story-context (★★★☆☆)

**What it is**: Generate targeted technical guidance XML per story before implementation.

**v6 Approach**:
```bash
bmad sm story-context  # Generates expertise injection XML
bmad dev dev-story     # Implements with context
```

**Current System**: Dev reads all previous artifacts (PRD, architecture, sprint plan) directly.

**Gap**: Dev agent must parse large documents to find relevant info for current story.

**Recommendation**: **MEDIUM PRIORITY - Add Context Generator**

Implementation plan:
1. Create `/bmad-sm-context` command (runs before dev-story)
2. Input: Current story + PRD + architecture
3. Output: `story-{id}-context.xml` with:
   - Relevant technical constraints
   - Integration points for this story
   - Security considerations
   - Performance requirements
   - Example implementations
4. bmad-dev reads context file first, then implements

**Benefits**:
- Reduces context window usage
- More focused implementation guidance
- Consistent technical patterns
- Faster dev agent reasoning

---

### 6. Continuous Learning - Retrospectives (★★★☆☆)

**What it is**: Capture learnings after each epic and feed improvements back into workflows.

**v6 Approach**:
```bash
bmad sm retrospective  # After epic complete
# Documents:
# - What went well
# - What could improve
# - Action items for next epic
# - Workflow adjustments
```

**Current System**: No retrospective mechanism.

**Gap**: We don't learn from successes/failures across epics.

**Recommendation**: **LOW PRIORITY - Add Retrospective Workflow**

Implementation plan:
1. Create `/bmad-retrospective` command (triggered after epic complete)
2. Generate `.claude/specs/{feature}/retrospective-epic-N.md`
3. Sections:
   - Epic summary (planned vs actual)
   - What went well
   - What didn't work
   - Learnings for next epic
   - Workflow improvements
4. Next epic's planning reads previous retrospectives

**Benefits**:
- Continuous improvement
- Team learning capture
- Better estimations over time
- Process optimization

---

### 7. Workflow Phase Structure (★★★★☆)

**v6 Four-Phase Model**:
```
Phase 1: Analysis (Optional) - Brainstorming, research, briefs
Phase 2: Planning (Required) - Scale-adaptive routing, PRD/GDD, epics
Phase 3: Solutioning (L3-4 only) - Architecture, JIT tech-specs
Phase 4: Implementation (Iterative) - Story state machine loop
```

**Current System**:
```
Phase 0: Repository Scan
Phase 1: Product Requirements (PO)
Phase 2: System Architecture (Architect)
Phase 3: Sprint Planning (SM)
Phase 4: Development (Dev)
Phase 5: Code Review (Review)
Phase 6: QA Testing (QA)
```

**Key Differences**:
- v6 has optional analysis phase (we don't)
- v6 has scale-adaptive routing (we don't)
- v6 treats implementation as iterative loop (we treat as linear)
- v6 has solutioning phase only for complex projects (we always architect)

**Recommendation**: **MEDIUM PRIORITY - Restructure Phases**

Proposed new structure:
```
Phase 0: Status Check (workflow-status) - NEW
Phase 1: Analysis (Optional) - NEW - brainstorming, research
Phase 2: Planning (Scale-Adaptive) - ENHANCED
  - Level 0-1: Tech-spec only
  - Level 2: PRD + tech-spec
  - Level 3-4: PRD + epics
Phase 3: Solutioning (L2-4 only) - ENHANCED
  - Level 2: Lightweight architecture
  - Level 3-4: Full architecture + JIT tech-specs
Phase 4: Implementation (Iterative) - ENHANCED
  - Story state machine
  - Dev → Review → Approve loop
Phase 5: QA Testing (Optional) - KEEP
  - Can be skipped with --skip-tests
```

---

## Comparison Matrix

| Feature | v6 BMAD-METHOD | Current System | Priority | Effort |
|---------|----------------|----------------|----------|--------|
| Scale-adaptive planning | ✅ Level 0-4 routing | ❌ Fixed workflow | HIGH | Medium |
| Universal entry point | ✅ workflow-status | ❌ Manual selection | MEDIUM | Low |
| JIT tech specs | ✅ One per epic | ❌ All upfront | HIGH | Medium |
| Story state machine | ✅ 4-state tracking | ❌ No tracking | HIGH | Medium |
| Story context injection | ✅ Per-story XML | ❌ Read all docs | MEDIUM | Low |
| Retrospectives | ✅ After each epic | ❌ None | LOW | Low |
| Brownfield support | ✅ Docs-first approach | ⚠️ No special handling | MEDIUM | High |
| Quality gates | ⚠️ Implicit | ✅ Explicit scoring | - | - |
| Code review phase | ❌ Not separate | ✅ Dedicated phase | - | - |
| Repository scan | ❌ Not mentioned | ✅ Phase 0 | - | - |

**Legend**:
- ✅ Fully supported
- ⚠️ Partially supported
- ❌ Not supported

---

## Adoptable Practices - Prioritized Roadmap

### Phase 1: Quick Wins (1-2 weeks)

**Goal**: Add high-value features with low implementation effort

1. **Universal Entry Point** (2 days)
   - Create `/workflow-status` command
   - Implement `.claude/workflow-status.md` tracking file
   - Auto-detect project context and recommend workflow

2. **Story Context Injection** (2 days)
   - Create `/bmad-sm-context` command
   - Generate story-specific context XMLs
   - Update bmad-dev to read context files

3. **Retrospectives** (1 day)
   - Create `/bmad-retrospective` command
   - Simple template for epic learnings
   - Store in `.claude/specs/{feature}/retrospective-epic-N.md`

**Expected Impact**: Better workflow guidance, focused dev context, learning capture

---

### Phase 2: Core Improvements (2-3 weeks)

**Goal**: Implement scale-adaptive planning and state machine

1. **Scale-Adaptive Planning** (1 week)
   - Create workflow classifier agent
   - Implement Level 0-4 routing logic
   - Add shortcuts:
     - Level 0: `/code-spec` (tech-spec only)
     - Level 1: `/mini-sprint` (tech-spec + few stories)
     - Level 2-4: `/bmad-pilot` (current workflow, enhanced)

2. **Story State Machine** (1 week)
   - Enhance sprint plan with 4-state sections
   - Create state transition commands:
     - `/bmad-sm-draft-story`
     - `/bmad-sm-approve-story`
     - `/bmad-dev-complete-story`
   - Update agents to read state file

**Expected Impact**: 80% faster for small changes, clear story tracking

---

### Phase 3: Architectural Changes (3-4 weeks)

**Goal**: Implement JIT architecture and brownfield support

1. **JIT Technical Specifications** (2 weeks)
   - Split architecture phase:
     - Phase 2: High-level architecture.md
     - Phase 3: Epic-specific tech-spec-epic-N.md (JIT)
   - Create `/bmad-architect-epic <epic-num>` command
   - Update dev workflow to request tech specs as needed

2. **Brownfield Support** (1 week)
   - Create `/bmad-analyze-codebase` command
   - Check for documentation before planning
   - Generate baseline docs for existing code

**Expected Impact**: Better architecture decisions, existing codebase support

---

### Phase 4: Workflow Restructuring (4-5 weeks)

**Goal**: Align with v6 phase model

1. **Phase Restructure** (2 weeks)
   - Add optional Analysis phase (brainstorming, research)
   - Make Solutioning phase conditional (L2-4 only)
   - Convert Implementation to iterative loop

2. **Integration & Testing** (2 weeks)
   - Test all new workflows end-to-end
   - Update documentation
   - Create migration guide

**Expected Impact**: More flexible, efficient workflows

---

## What NOT to Adopt

### 1. Remove Quality Scoring ❌ NOT RECOMMENDED

**v6**: No explicit quality gates with numeric scores  
**Current**: 90/100 threshold for PRD and Architecture

**Reasoning**: Our quality scoring system provides objective feedback and clear improvement targets. v6's implicit quality checks are less transparent. **Keep our scoring system.**

### 2. Remove Code Review Phase ❌ NOT RECOMMENDED

**v6**: No separate review phase (incorporated into dev-story)  
**Current**: Dedicated bmad-review agent between Dev and QA

**Reasoning**: Separation of concerns improves quality. Independent reviewer catches issues dev might miss. **Keep review phase.**

### 3. Remove Repository Scan ❌ NOT RECOMMENDED

**v6**: No automatic codebase analysis  
**Current**: Phase 0 repository scan

**Reasoning**: Understanding existing codebase is critical. Our scan provides valuable context. **Keep repository scan.**

---

## Implementation Strategy

### Incremental Adoption Approach

**Week 1-2: Quick Wins**
```bash
# Add new commands (parallel to existing workflow)
/workflow-status     # Universal entry point
/bmad-sm-context     # Story context injection
/bmad-retrospective  # Epic learnings
```

**Week 3-5: Core Features**
```bash
# Enhance existing workflow
/bmad-pilot --level 0  # Scale-adaptive routing
# Story state machine in sprint plan
```

**Week 6-9: Architecture**
```bash
# Split architecture phase
/bmad-architect        # High-level (Phase 2)
/bmad-architect-epic 1 # JIT tech-spec (Phase 3)
```

**Week 10-14: Full Integration**
```bash
# New phase structure with all enhancements
```

### Backward Compatibility

- Keep existing commands working (`/bmad-pilot` without flags)
- Maintain current artifact structure (`.claude/specs/`)
- Gradual migration - old and new workflows coexist
- Clear migration documentation for users

---

## Success Metrics

### Quantitative Goals

1. **Workflow Efficiency**
   - 80% reduction in time for Level 0-1 changes
   - 50% reduction in context window usage via story-context
   - 30% reduction in architecture rework via JIT approach

2. **User Experience**
   - 100% of users understand current workflow phase (workflow-status)
   - 90% reduction in "which command do I run?" confusion
   - Zero manual story selection (state machine handles it)

3. **Code Quality**
   - Maintain 90/100 quality gate threshold
   - Increase epic-to-epic estimation accuracy by 20% (via retrospectives)
   - Zero regression in review/QA effectiveness

### Qualitative Goals

- More adaptive workflows (right-sized for task)
- Clearer progress visibility
- Better learning capture across epics
- Improved brownfield project support

---

## Risks & Mitigation

| Risk | Impact | Mitigation |
|------|--------|------------|
| User confusion from workflow changes | High | Gradual rollout, clear docs, backward compatibility |
| Implementation complexity | Medium | Incremental phases, thorough testing |
| State machine bugs | Medium | Comprehensive state transition testing |
| JIT architecture quality issues | Medium | Keep quality gates, provide good context |
| Migration effort for existing users | Low | Both old and new workflows work side-by-side |

---

## Conclusion

The v6 BMAD-METHOD workflow introduces several powerful innovations that address real pain points in our current system:

**Must Adopt** (HIGH Priority):
1. ✅ Scale-adaptive planning - Eliminates workflow overhead for simple changes
2. ✅ JIT technical specifications - Prevents over-engineering, incorporates learning
3. ✅ Story state machine - Clear progress tracking, eliminates ambiguity

**Should Adopt** (MEDIUM Priority):
4. ✅ Universal entry point - Better user experience, workflow guidance
5. ✅ Phase restructure - More flexible, efficient workflows
6. ✅ Story context injection - Reduces context usage, focused implementation

**Nice to Have** (LOW Priority):
7. ✅ Retrospectives - Continuous improvement, learning capture

**Keep Our Innovations**:
- ✅ Quality scoring system (90/100 gates)
- ✅ Dedicated code review phase
- ✅ Repository scan automation

### Recommended Action Plan

**Immediate** (This sprint):
- Create `/workflow-status` command
- Implement story-context injection
- Add retrospective support

**Next Sprint**:
- Build scale-adaptive classifier
- Implement story state machine
- Add Level 0-1 fast paths

**Next Month**:
- Implement JIT architecture
- Add brownfield support
- Full phase restructure

**Timeline**: 10-14 weeks for complete v6 feature parity while preserving our quality innovations.

---

## References

- **v6 Source**: https://github.com/bmad-code-org/BMAD-METHOD/blob/v6-alpha/src/modules/bmm/workflows/README.md
- **Current Workflow**: `docs/BMAD-WORKFLOW.md`
- **Current Agents**: `bmad-agile-workflow/agents/`
- **Current Commands**: `bmad-agile-workflow/commands/`

---

*Analysis completed: 2025-10-20*  
*Analyst: SWE Agent*  
*Next Review: After Phase 1 implementation (2 weeks)*
