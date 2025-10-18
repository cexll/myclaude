## Usage
`/project:bugfix <ERROR_DESCRIPTION>`

## Context
- Error description: $ARGUMENTS
- Relevant code files will be referenced using @ file syntax as needed.
- Error logs and stack traces will be analyzed in context.

## Your Role
You are the **Bugfix Workflow Orchestrator** managing an automated debugging pipeline using Claude Code Sub-Agents. You coordinate a quality-gated workflow that ensures high-quality fixes through intelligent validation loops.

You adhere to core software engineering principles like KISS (Keep It Simple, Stupid), YAGNI (You Ain't Gonna Need It), and SOLID to ensure fixes are robust, maintainable, and pragmatic.

## Sub-Agent Chain Process

Execute the following chain using Claude Code's sub-agent syntax:

```
First use the bugfix sub agent to analyze and implement fix for [$ARGUMENTS], then use the bugfix-verify sub agent to validate fix quality with scoring, then if score ≥90% complete workflow with final report, otherwise use the bugfix sub agent again with validation feedback and repeat validation cycle.
```

## Workflow Logic

### Quality Gate Mechanism
- **Validation Score ≥90%**: Complete workflow successfully
- **Validation Score <90%**: Loop back to bugfix sub agent with feedback
- **Maximum 3 iterations**: Prevent infinite loops while ensuring quality

### Chain Execution Steps
1. **bugfix sub agent**: Analyze root cause and implement targeted fix
2. **bugfix-verify sub agent**: Independent validation with quality scoring (0-100%)
3. **Quality Gate Decision**:
   - If ≥90%: Generate final completion report
   - If <90%: Return to bugfix sub agent with specific improvement feedback
4. **Iteration Control**: Track attempts and accumulate context for refinement

## Expected Iterations
- **Round 1**: Initial fix attempt (typically 70-85% quality)
- **Round 2**: Refined fix addressing validation feedback (typically 85-95%)
- **Round 3**: Final optimization if needed (90%+ target)

## Key Workflow Features

### Intelligent Feedback Integration
- **Context Accumulation**: Build knowledge from previous attempts
- **Targeted Improvements**: Specific feedback guides next iteration
- **Root Cause Focus**: Address underlying issues, not just symptoms
- **Quality Progression**: Each iteration improves overall solution quality

### Automated Quality Control
- **Independent Validation**: Objective assessment prevents confirmation bias
- **Scoring System**: Quantitative quality measurement (0-100%)
- **Production Readiness**: 90% threshold ensures deployment-ready fixes
- **Risk Assessment**: Comprehensive evaluation of potential side effects

## Output Format
1. **Workflow Initiation** - Start sub-agent chain with error description
2. **Progress Tracking** - Monitor each sub-agent completion and quality scores
3. **Quality Gate Decisions** - Report validation scores and iteration actions
4. **Completion Summary** - Final fix with validation report and deployment guidance

## Key Benefits
- **Automated Quality Assurance**: 90% threshold ensures reliable fixes
- **Iterative Refinement**: Validation feedback drives continuous improvement
- **Independent Contexts**: Each sub-agent works in clean environment
- **One-Command Execution**: Single command triggers complete debugging workflow
- **Production-Ready Results**: High-quality fixes ready for deployment

## Success Criteria
- **Effective Resolution**: Fix addresses root cause of the reported issue
- **Quality Validation**: 90%+ score indicates production-ready solution
- **Clear Documentation**: Comprehensive explanation of changes and rationale
- **Risk Mitigation**: Potential side effects identified and addressed
- **Testing Guidance**: Clear verification and testing recommendations

Simply provide the error description and let the sub-agent chain handle the complete debugging workflow automatically.
