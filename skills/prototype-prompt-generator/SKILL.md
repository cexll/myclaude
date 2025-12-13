---
name: prototype-prompt-generator
description: This skill should be used when users need to generate detailed, structured prompts for creating UI/UX prototypes. Trigger when users request help with "create a prototype prompt", "design a mobile app", "generate UI specifications", or need comprehensive design documentation for web/mobile applications. Works with multiple design systems including WeChat Work, iOS Native, Material Design, and Ant Design Mobile.
---

# Prototype Prompt Generator

## Overview

Generate comprehensive, production-ready prompts for UI/UX prototype creation. Transform user requirements into detailed technical specifications that include design systems, color palettes, component specifications, layout structures, and implementation guidelines. Output prompts are structured for optimal consumption by AI tools or human developers building HTML/CSS/React prototypes.

## Workflow

### Step 1: Gather Requirements

Begin by collecting essential information from the user. Ask targeted questions to understand:

**Application Type & Purpose:**
- What kind of application? (e.g., enterprise tool, e-commerce, social media, dashboard)
- Who are the target users?
- What are the primary use cases and workflows?

**Platform & Context:**
- Target platform: iOS, Android, Web, WeChat Mini Program, or cross-platform?
- Device: Mobile phone, tablet, desktop, or responsive?
- Viewport dimensions if known (e.g., 375px for iPhone, 1200px for desktop)

**Design Preferences:**
- Design style: WeChat Work, iOS Native, Material Design, Ant Design Mobile, or custom?
- Brand colors or visual preferences?
- Any design references or inspiration?

**Feature Requirements:**
- Key pages and features needed
- Navigation structure (tabs, drawer, stack navigation)
- Data to display (metrics, lists, forms, media)
- User interactions (tap, swipe, long-press, etc.)

**Content & Data:**
- Actual content to display (realistic text, numbers, names)
- Empty states, error states, loading states
- Any specific business logic or rules

**Technical Constraints:**
- Framework preference: Plain HTML, React, Vue, or framework-agnostic?
- CSS approach: Tailwind CSS, CSS Modules, styled-components?
- Image assets: Real images, placeholders, or specific sources?
- CDN dependencies or version requirements?

**Ask questions incrementally** (2-3 at a time) to avoid overwhelming the user. Many details can be inferred from context or filled with sensible defaults.

### Step 2: Select Design System

Based on the gathered requirements, choose the appropriate design system from `references/design-systems.md`:

**WeChat Work Style:**
- **When to use**: Chinese enterprise applications, work management tools, B2B platforms, internal business systems
- **Characteristics**: Simple and professional, tech blue primary color, clear information hierarchy
- **Key audience**: Chinese business users, corporate environments

**iOS Native Style:**
- **When to use**: iOS-specific apps, Apple ecosystem integration, apps targeting iPhone/iPad users
- **Characteristics**: Minimalist, spacious layouts, San Francisco font, system colors
- **Key audience**: Apple users, consumer apps, content-focused applications

**Material Design Style:**
- **When to use**: Android-first apps, Google ecosystem integration, cross-platform with Material UI
- **Characteristics**: Bold graphics, elevation system, ripple effects, Roboto font
- **Key audience**: Android users, Google services, developer tools

**Ant Design Mobile Style:**
- **When to use**: Enterprise mobile applications with complex data entry and forms
- **Characteristics**: Efficiency-oriented, consistent components, suitable for business applications
- **Key audience**: Business users, enterprise mobile apps, data-heavy interfaces

**If the user hasn't specified a design system**, recommend one based on:
- Geographic location: Chinese users → WeChat Work, Western users → iOS/Material
- Platform: iOS → iOS Native, Android → Material Design
- Application type: Enterprise B2B → WeChat Work or Ant Design, Consumer app → iOS or Material

Load the complete design system specifications from `references/design-systems.md` to ensure accurate color codes, component dimensions, and interaction patterns.

### Step 3: Structure the Prompt

Using the template from `references/prompt-structure.md`, construct a comprehensive prompt with these sections:

**1. Role Definition**
Define expertise relevant to the prototype:
```
# Role
You are a world-class UI/UX engineer and frontend developer, specializing in [specific domain] using [technologies].
```

**2. Task Description**
State clearly what to build and the design style:
```
# Task
Create a [type] prototype for [application description].
Design style must strictly follow [design system], with core keywords: [3-5 key attributes].
```

**3. Tech Stack Specifications**
List all technologies, frameworks, and resources:
- File structure (single HTML, multi-page, component-based)
- Framework and version (e.g., Tailwind CSS CDN)
- Device simulation (viewport size, device chrome)
- Asset sources (Unsplash, Pexels, real images)
- Icon libraries (FontAwesome, Material Icons)
- Custom configuration (Tailwind config, theme variables)

**4. Visual Design Requirements**
Provide detailed specifications:

**(a) Color Palette:**
Include all colors with hex codes:
- Background colors (main, section, card)
- Primary and accent colors with usage
- Status colors (success, warning, error)
- Text colors (title, body, secondary, disabled)
- UI element colors (borders, dividers)

**(b) UI Style Characteristics:**
Specify for each component type:
- Cards: background, radius, shadow, border, padding
- Buttons: variants (primary, secondary, ghost), dimensions, states
- Icons: style, sizes, colors, containers
- List items: layout, height, divider style, active state
- Shadows: type and usage

**(c) Layout Structure:**
Describe each major section:
- Top navigation bar: height, title style, icons, background
- Content areas: grids, cards, lists, spacing
- Quick access areas: icon grids, layouts
- Data display cards: metrics, layout, styling
- Feature lists: structure, icons, interactions
- Bottom tab bar: height, tabs, active/inactive states, badges

**(d) Specific Page Content:**
Provide actual content, not placeholders:
- Real page titles and section headings
- Actual data points (numbers, names, dates)
- Feature names and descriptions
- Button labels and link text
- Sample list items with realistic content

**5. Implementation Details**
Cover technical specifics:
- Page width and centering approach
- Layout systems (Flexbox, Grid, or both)
- Fixed/sticky positioning for navigation
- Spacing scale (margins, padding, gaps)
- Typography (font family, sizes, weights)
- Interactive states (hover, active, focus, disabled)
- Icon sources and usage
- Border and divider styling

**6. Tailwind Configuration**
If using Tailwind CSS, provide custom config:
```javascript
tailwind.config = {
  theme: {
    extend: {
      colors: {
        'brand-primary': '#3478F6',
        // ... all custom colors
      }
    }
  }
}
```

**7. Content Structure & Hierarchy**
Visualize the page structure as a tree:
```
Page Name
├─ Section 1
│  ├─ Element 1
│  └─ Element 2
├─ Section 2
│  ├─ Subsection A
│  │  ├─ Item 1
│  │  └─ Item 2
│  └─ Subsection B
└─ Section 3
```

**8. Special Requirements**
Highlight unique considerations:
- Design system-specific guidelines
- Primary color application scenarios
- Interaction details (tap feedback, animations, gestures)
- Accessibility requirements (contrast, touch targets, ARIA)
- Performance considerations (image optimization, lazy loading)

**9. Output Format**
Specify the exact deliverable:
```
# Output Format

Please output complete [file type] code, ensuring:
1. [Requirement 1]
2. [Requirement 2]
...

The output should be production-ready and viewable at [viewport] on [device].
```

### Step 4: Populate with Specifics

Replace all template placeholders with concrete values:

**Replace vague terms with precise specifications:**
- ❌ "Use blue colors" → ✅ "Primary: #3478F6 (tech blue), Link: #576B95 (link blue)"
- ❌ "Make buttons rounded" → ✅ "Border radius: 4px (Tailwind: rounded)"
- ❌ "Add some spacing" → ✅ "Card spacing: 12px, page margins: 16px"
- ❌ "Display user info" → ✅ "Show username (15px bold), email (13px gray), avatar (48px circle)"

**Use real content, not placeholders:**
- ❌ "Lorem ipsum dolor sit amet" → ✅ "Customer Total: 14, Today's New Customers: 1, Today's Revenue: ¥0.00"
- ❌ "[Company Name]" → ✅ "Acme Insurance Co."
- ❌ "Feature 1, Feature 2, Feature 3" → ✅ "Customer Contact, Customer Moments, Customer Groups"

**Specify all measurements:**
- Component heights (44px, 50px, 64px)
- Font sizes (13px, 15px, 16px, 18px)
- Spacing values (8px, 12px, 16px, 24px)
- Icon sizes (24px, 32px, 48px)
- Border radius (4px, 8px, 10px)

**Define all states:**
- Normal: base colors and styles
- Hover: if applicable (desktop)
- Active/Pressed: opacity or background changes
- Disabled: grayed out with reduced opacity
- Selected: highlight color (often primary brand color)

**Include all colors:**
Every color mentioned must have a hex code. Reference the chosen design system from `references/design-systems.md` for accurate values.

### Step 5: Quality Assurance

Before presenting the final prompt, verify against the checklist in `references/prompt-structure.md`:

**Completeness Check:**
- [ ] Role clearly defined with relevant expertise
- [ ] Task explicitly states what to build and design style
- [ ] All tech stack components listed with versions/CDNs
- [ ] Complete color palette with hex codes for all colors
- [ ] All UI components specified with exact dimensions and styles
- [ ] Page layout fully described with precise measurements
- [ ] Actual, realistic content provided (no placeholders like "Lorem Ipsum" or "[Name]")
- [ ] Implementation details cover all technical requirements
- [ ] Tailwind config included if using Tailwind CSS
- [ ] Content hierarchy visualized as a tree structure
- [ ] Special requirements and interactions documented
- [ ] Output format clearly defined with all deliverables

**Clarity Check:**
- [ ] No ambiguous terms or vague descriptions (e.g., "some padding", "nice colors")
- [ ] All measurements specified with units (px, rem, %, vh, etc.)
- [ ] All colors defined with hex codes (e.g., #3478F6, not just "blue")
- [ ] Component states described (normal, hover, active, disabled, selected)
- [ ] Layout relationships clear (parent-child, spacing, alignment, z-index)

**Specificity Check:**
- [ ] Design system explicitly named (WeChat Work, iOS Native, Material Design, etc.)
- [ ] Viewport dimensions provided (e.g., 375px × 812px for iPhone)
- [ ] Typography scale defined (sizes, weights, line heights)
- [ ] Interactive behaviors documented with timing if animated
- [ ] Edge cases considered (long text overflow, empty states, loading, errors)

**Realism Check:**
- [ ] Real content examples, not Latin placeholder text
- [ ] Authentic data points (realistic numbers, names, dates, amounts)
- [ ] Practical feature set (not overengineered or underspecified)
- [ ] Appropriate complexity for the stated use case

**Technical Accuracy Check:**
- [ ] Valid Tailwind class names (if using Tailwind)
- [ ] Correct CDN links with versions (e.g., https://cdn.tailwindcss.com)
- [ ] Proper HTML structure implied (semantic elements, hierarchy)
- [ ] Feasible layout techniques (Flexbox/Grid patterns that work)
- [ ] Accessible markup considerations (touch targets ≥44px, color contrast)

If any checks fail, refine the prompt before proceeding.

### Step 6: Present and Iterate

**Present the generated prompt to the user** with a brief explanation:
- What design system was selected and why
- Key design decisions made
- Any assumptions or defaults applied
- How to use the prompt (copy and provide to another AI tool or developer)

**Offer refinement options:**
- "Would you like to adjust any colors or spacing?"
- "Should we add more pages or features?"
- "Do you want to change the design system?"
- "Any specific interactions or animations to emphasize?"

**Iterate based on feedback:**
If the user requests changes:
1. Update the relevant sections of the prompt
2. Maintain consistency across all sections
3. Re-verify against the quality checklist
4. Present the updated prompt

**Save or Export:**
Offer to save the prompt to a file:
- Markdown file for documentation
- Text file for easy copying
- Include as a code block for immediate use

## Best Practices

**1. Default to High Quality:**
Even if the user provides minimal requirements, generate a comprehensive prompt. It's easier to remove details than to add them later. Include:
- Complete color palettes (8-12 colors minimum)
- All common UI components (buttons, cards, lists, inputs)
- Multiple component states (normal, active, disabled)
- Responsive considerations
- Accessibility basics (contrast, touch targets)

**2. Use Design System Defaults Intelligently:**
When user requirements are vague:
- Apply the full design system consistently
- Use standard component dimensions from the design system
- Follow established patterns (e.g., WeChat Work's 64px list items)
- Include typical interaction patterns for the platform

**3. Prioritize Clarity Over Brevity:**
Longer, detailed prompts produce better prototypes than short, vague ones. Include:
- Exact hex codes instead of color names
- Precise measurements instead of relative terms
- Specific component layouts instead of general descriptions
- Actual content instead of placeholder text

**4. Think Mobile-First:**
For mobile applications, always consider:
- Safe areas (iOS notch, Android gesture bar)
- Touch target sizes (minimum 44px × 44px)
- Thumb-reachable zones (bottom navigation over top)
- Portrait orientation primarily (landscape as secondary)
- One-handed operation where possible

**5. Balance Flexibility and Specificity:**
- Be specific about core design elements (colors, typography, key components)
- Allow flexibility in implementation details (exact animation timing, minor spacing adjustments)
- Specify "must-haves" clearly, mark "nice-to-haves" as optional

**6. Consider the Full User Journey:**
Include specifications for:
- Entry points (splash screen, onboarding if applicable)
- Primary workflows (happy path through key features)
- Edge cases (empty states, error states, loading states)
- Exit points (logout, back navigation, completion states)

**7. Provide Context, Not Just Specs:**
Explain the "why" behind design decisions:
- "Tech blue (#3478F6) for trust and professionalism in enterprise context"
- "64px list item height for comfortable thumb tapping on mobile"
- "Fixed bottom tab bar for quick access to primary features"

**8. Validate Technical Feasibility:**
Before finalizing the prompt:
- Ensure CSS/Tailwind classes can achieve the described design
- Verify that layout patterns work with the stated grid/flexbox approach
- Confirm that the specified viewport can accommodate all content
- Check that CDN links and versions are correct and available

**9. Make It Actionable:**
The prompt should enable immediate implementation:
- Include all necessary CDN links and imports
- Provide complete Tailwind config (no "...add more as needed")
- Specify file structure and organization
- Define clear deliverables (HTML file, React components, etc.)

**10. Anticipate Questions:**
Address common uncertainties in the prompt:
- Font fallbacks (e.g., "sans-serif" system font stack)
- Image dimensions and aspect ratios
- Icon usage (when to use FontAwesome vs SVG vs emoji)
- Z-index layering (what's on top)
- Overflow behavior (scroll, truncate, wrap)

## Common Patterns

### Pattern 1: Enterprise Work Dashboard (WeChat Work Style)
**Typical Structure:**
- Top navigation bar (44px, title + search/menu icons)
- Quick access grid (4-column icon grid)
- Data summary cards (key metrics in horizontal layout)
- Feature list (icon + text rows, 64px height each)
- Bottom tab bar (5 tabs, 50px height)

**Key Elements:**
- Tech blue (#3478F6) for primary actions and active states
- White cards with subtle shadows on light gray background
- 48px icons with rounded-lg containers
- Right arrow indicators for navigation

### Pattern 2: iOS Consumer App (iOS Native Style)
**Typical Structure:**
- Large title navigation bar (96px when expanded)
- Card-based content sections
- System standard lists (44px minimum row height)
- Tab bar with SF Symbols icons

**Key Elements:**
- System blue (#007AFF) for interactive elements
- Generous whitespace (20px margins, 16px padding)
- Subtle dividers with left inset
- Translucent blur effects on navigation

### Pattern 3: Android App (Material Design Style)
**Typical Structure:**
- Top app bar (56px on mobile, 64px on tablet)
- FAB (Floating Action Button) for primary action
- Card-based content with elevation
- Bottom navigation or navigation drawer

**Key Elements:**
- Bold primary color (#6200EE) with elevation shadows
- Ripple effects on tap
- 16dp grid system
- Material icons (24px)

### Pattern 4: Enterprise Form App (Ant Design Mobile)
**Typical Structure:**
- Simple navigation bar (45px)
- Form sections with grouped inputs
- List views with detailed information
- Fixed bottom action bar with primary button

**Key Elements:**
- Professional blue (#108EE9) for actions
- Dense information layout
- Clear form field labels and validation
- Breadcrumb or step indicators for multi-step flows

## Troubleshooting

**Issue: User requirements are too vague**
**Solution:** Ask focused questions, provide examples of similar apps, suggest design systems to choose from, or create a default prompt and offer iteration.

**Issue: User wants multiple design styles mixed**
**Solution:** Pick a primary design system for overall structure and consistency, then incorporate specific elements from other systems as accent features. Explain trade-offs.

**Issue: User specifies impossible or conflicting requirements**
**Solution:** Identify the conflict, explain why it's problematic (e.g., "64px icons won't fit in a 44px navigation bar"), suggest alternatives, and seek clarification.

**Issue: Too many features for one prompt**
**Solution:** Focus on the primary page/workflow first, generate that prompt, then create separate prompts for additional features. Maintain consistency across prompts.

**Issue: User lacks technical knowledge**
**Solution:** Avoid jargon, explain design decisions in plain language, provide visual descriptions instead of technical terms, and include helpful comments in the prompt.

**Issue: Prototype prompt doesn't produce good results**
**Solution:** Review against the quality checklist, ensure all colors have hex codes, verify all measurements are specified, add more specific content examples, check for ambiguous language.

## Resources

This skill includes reference documentation to support prompt generation:

### references/design-systems.md
Comprehensive specifications for major design systems:
- **WeChat Work Style**: Chinese enterprise applications
- **iOS Native Style**: Apple ecosystem apps
- **Material Design**: Google/Android apps
- **Ant Design Mobile**: Enterprise mobile apps

Each design system includes:
- Complete color palettes with hex codes
- Component specifications (dimensions, spacing, states)
- Typography scales (sizes, weights, line heights)
- Interaction patterns (animations, gestures, feedback)
- Layout guidelines (grids, spacing, safe areas)
- Code examples (Tailwind classes, CSS snippets)

**When to reference:** Always load this file when generating a prompt to ensure accurate design system specifications. Use it to populate color values, component dimensions, and interaction patterns.

### references/prompt-structure.md
Detailed template and guidelines for prompt construction:
- Standard prompt structure (9 sections)
- Template syntax with placeholders
- Examples for each section
- Quality checklist (completeness, clarity, specificity)
- Workflow guidance (requirements → prompt → iteration)
- Tips for effective prompts
- Common pitfalls to avoid

**When to reference:** Use this as the skeleton for every generated prompt. It ensures consistency and completeness across all prompts you create.

---

**Note:** This skill generates prompts for prototype creation—it does not create the prototypes themselves. The output is a comprehensive text prompt that can be provided to another AI tool, developer, or design tool to generate the actual HTML/CSS/React code.
