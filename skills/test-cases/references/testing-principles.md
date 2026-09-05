# Testing Principles and Best Practices

## Core Philosophy

**Test what matters** - Focus on functionality that impacts users: behavior, performance, data integrity, and user experience. Avoid testing implementation details that can change without affecting outcomes.

**Requirement-driven testing** - Every test must trace back to a specific requirement. If a requirement exists without tests, coverage is incomplete. If a test exists without a requirement, it may be testing implementation rather than behavior.

**Quality over quantity** - A small set of stable, meaningful tests is more valuable than extensive flaky tests. Flaky tests erode trust and waste time. Every shipped bug represents a process failure.

## Coverage Requirements

### 1. Happy Path Coverage
Test all normal use cases from requirements:
- Primary user flows
- Expected inputs and outputs
- Standard workflows
- Common scenarios

**Example**: For a login feature, test successful login with valid credentials.

### 2. Edge Case Coverage
Test boundary conditions and unusual inputs:
- Empty inputs (null, undefined, empty string, empty array)
- Boundary values (min, max, zero, negative)
- Maximum limits (character limits, file size limits, array lengths)
- Special characters and encoding
- Concurrent operations

**Example**: For a login feature, test with empty username, maximum length password, special characters in credentials.

### 3. Error Handling Coverage
Test failure scenarios and error conditions:
- Invalid inputs (wrong type, format, range)
- Permission errors (unauthorized access, insufficient privileges)
- Network failures (timeout, connection lost, server error)
- Resource exhaustion (out of memory, disk full)
- Dependency failures (database down, API unavailable)

**Example**: For a login feature, test with invalid credentials, account locked, server timeout.

### 4. State Transition Coverage
If the feature involves state, test all valid state changes:
- Initial state to each possible next state
- All valid state transitions
- Invalid state transitions (should be rejected)
- State persistence across sessions
- Concurrent state modifications

**Example**: For a login feature, test transitions: logged out → logging in → logged in → logging out → logged out.

## Test Case Structure

### Essential Components

Every test case must include:

1. **Unique ID** - Consistent naming convention (TC-F-001, TC-E-001, etc.)
2. **Title** - Clear, descriptive name explaining what's being tested
3. **Requirement Link** - Traceability to specific requirement
4. **Priority** - High/Medium/Low based on user impact
5. **Preconditions** - State that must exist before test execution
6. **Test Steps** - Clear, numbered, executable actions
7. **Expected Results** - Measurable, verifiable outcomes
8. **Postconditions** - State after test completion

### Test Case Naming Convention

Use prefixes to categorize test cases:
- **TC-F-XXX**: Functional tests (happy path)
- **TC-E-XXX**: Edge case tests (boundaries)
- **TC-ERR-XXX**: Error handling tests (failures)
- **TC-ST-XXX**: State transition tests (workflows)
- **TC-PERF-XXX**: Performance tests (speed, load)
- **TC-SEC-XXX**: Security tests (auth, permissions)

## Test Design Patterns

### Pattern 1: Arrange-Act-Assert (AAA)

Structure test steps using AAA pattern:
1. **Arrange** - Set up preconditions and test data
2. **Act** - Execute the action being tested
3. **Assert** - Verify expected results

**Example**:
```
Preconditions:
- User account exists with username "testuser"
- User is logged out

Test Steps:
1. Navigate to login page (Arrange)
2. Enter username "testuser" and password "password123" (Arrange)
3. Click "Login" button (Act)
4. Verify user is redirected to dashboard (Assert)
5. Verify welcome message displays "Welcome, testuser" (Assert)
```

### Pattern 2: Equivalence Partitioning

Group inputs into equivalence classes and test one representative from each class:
- Valid equivalence class
- Invalid equivalence classes
- Boundary values

**Example**: For age input (valid range 18-100):
- Valid class: 18, 50, 100
- Invalid class: 17, 101, -1, "abc"
- Boundaries: 17, 18, 100, 101

### Pattern 3: State Transition Testing

For stateful features, create a state transition table and test each transition:

| Current State | Action | Next State | Test Case |
|--------------|--------|------------|-----------|
| Logged Out | Login Success | Logged In | TC-ST-001 |
| Logged Out | Login Failure | Logged Out | TC-ST-002 |
| Logged In | Logout | Logged Out | TC-ST-003 |
| Logged In | Session Timeout | Logged Out | TC-ST-004 |

## Test Prioritization

Prioritize test cases based on:

1. **High Priority**
   - Core user flows (login, checkout, data submission)
   - Data integrity (create, update, delete operations)
   - Security-critical paths (authentication, authorization)
   - Revenue-impacting features (payment, subscription)

2. **Medium Priority**
   - Secondary user flows
   - Edge cases for high-priority features
   - Error handling for common failures
   - Performance-sensitive operations

3. **Low Priority**
   - Rare edge cases
   - Cosmetic issues
   - Nice-to-have features
   - Non-critical error scenarios

## Test Quality Indicators

### Good Test Cases
- ✓ Maps directly to a requirement
- ✓ Tests behavior, not implementation
- ✓ Has clear, executable steps
- ✓ Has measurable expected results
- ✓ Is independent of other tests
- ✓ Is repeatable and deterministic
- ✓ Fails only when behavior is broken

### Poor Test Cases
- ✗ Tests implementation details
- ✗ Has vague or ambiguous steps
- ✗ Has unmeasurable expected results
- ✗ Depends on execution order
- ✗ Is flaky or non-deterministic
- ✗ Fails due to environment issues

## Coverage Validation

Before finalizing test cases, verify:

1. **Requirement Coverage**
   - Every requirement has at least one test case
   - Critical requirements have multiple test cases
   - Coverage matrix shows complete mapping

2. **Scenario Coverage**
   - Happy path: All normal flows covered
   - Edge cases: Boundaries and limits covered
   - Error handling: Failure modes covered
   - State transitions: All valid transitions covered

3. **Risk Coverage**
   - High-risk areas have comprehensive coverage
   - Security-sensitive features are thoroughly tested
   - Data integrity operations are validated

## Common Pitfalls to Avoid

1. **Testing implementation instead of behavior** - Focus on what the system does, not how it does it
2. **Incomplete edge case coverage** - Don't forget empty inputs, boundaries, and limits
3. **Missing error scenarios** - Test failure modes, not just success paths
4. **Vague expected results** - Make results measurable and verifiable
5. **Test interdependencies** - Each test should be independent
6. **Ignoring state transitions** - For stateful features, test all transitions
7. **Over-testing trivial code** - Focus on logic that matters to users

## Test Documentation Standards

### File Organization
```
tests/
├── <feature>-test-cases.md     # Test cases for specific feature
├── <module>-test-cases.md      # Test cases for specific module
└── integration-test-cases.md   # Cross-feature integration tests
```

### Markdown Structure
- Use clear headings for test categories
- Use tables for coverage matrices
- Use code blocks for test data examples
- Use checkboxes for test execution tracking
- Include metadata (feature, date, version)

### Maintenance
- Update test cases when requirements change
- Remove obsolete test cases
- Add new test cases for bug fixes
- Review coverage regularly
- Keep test cases synchronized with implementation

## References

These principles are derived from:
- Industry-standard QA practices
- Game QA methodologies (Unity Test Framework, Unreal Automation, Godot GUT)
- Pragmatic testing philosophy: "Test what matters"
- Requirement-driven testing approach from CLAUDE.md context
