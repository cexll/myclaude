# Prototype Prompt Structure Template

This document provides a comprehensive template for generating high-quality prototype prompts. Each section includes guidelines and examples.

---

## Standard Prompt Structure

A well-structured prototype prompt should contain the following sections in order:

### 1. Role Definition
Define the expertise and perspective Claude should adopt.

**Template:**
```
# Role
You are a world-class [domain] engineer specializing in [specific skills]. You excel at [key capabilities].
```

**Example:**
```
# Role
You are a world-class UI/UX engineer and frontend developer, specializing in creating professional, efficient enterprise-grade mobile application interfaces using Tailwind CSS.
```

---

### 2. Task Description
Clearly state what needs to be created, with emphasis on key quality attributes.

**Template:**
```
# Task
Create a [type of prototype] for [application description].
Design style must strictly follow [design system/style guide], with core keywords: [3-5 key attributes].
```

**Example:**
```
# Task
Create a high-fidelity prototype (HTML + Tailwind CSS) for an insurance agent work application.
Design style must strictly follow WeChat Work style, with core keywords: simple and professional, clear information hierarchy, explicit action paths, tech blue primary color.
```

---

### 3. Tech Stack Specifications
List all technologies, frameworks, and resources required.

**Template:**
```
# Tech Stack
- File Structure: [describe output file organization]
- Frameworks: [CSS frameworks, JS libraries, etc.]
- Device Simulation: [viewport size, device features]
- Asset Sources: [image sources, icon libraries]
- CDN Resources: [external dependencies]
- Custom Configuration: [theme extensions, variables]
```

**Example:**
```
# Tech Stack
- Each page outputs as an independent HTML file (e.g., home.html, discover.html)
- index.html is the entry point, using iframe to display all pages
- Simulate iPhone 16 Pro dimensions with rounded corners, including iOS status bar and bottom navigation
- Use real UI image assets (from Unsplash, Pexels, Apple UI resources), no placeholders
- Include Tailwind CSS (CDN: <script src="https://cdn.tailwindcss.com"></script>)
- Use FontAwesome (CDN) for icons
- Custom Tailwind config with brand colors: Tech Blue #3478F6, Link Blue #576B95, Alert Red #FA5151
```

---

### 4. Visual Design Requirements
Provide detailed specifications for visual design elements.

**Sections to include:**
1. **Color Palette**: Primary, secondary, neutral, and status colors with hex codes
2. **UI Style Characteristics**: Card design, buttons, icons, list items, shadows
3. **Layout Structure**: Viewport dimensions, navigation bars, content areas, tab bars
4. **Specific Page Content**: Actual text, data, and functional elements

**Template:**

```
# Visual Design Requirements

## 1. Color Palette
**Background Colors:**
- Main Background: [color name] [hex code]
- Section Background: [color name] [hex code]

**Primary Colors:**
- Main Color: [color name] [hex code] (usage: [buttons, icons, emphasis])
- Accent Color: [color name] [hex code] (usage: [links, secondary actions])

**Status Colors:**
- Success: [hex code]
- Warning: [hex code]
- Error: [hex code]

**Text Colors:**
- Title: [hex code]
- Body: [hex code]
- Secondary: [hex code]

**UI Elements:**
- Divider: [hex code]
- Border: [hex code]

## 2. UI Style Characteristics

**Card Design:**
- Background: [color]
- Border Radius: [size] ([Tailwind class])
- Shadow: [type] ([Tailwind class])
- Border: [style]
- Spacing: [measurements]

**Buttons:**
- Primary Button: [background] + [text color], height [px], radius [size] ([Tailwind class])
- Active State: [opacity/color] ([Tailwind class])
- Additional variants: [outline, ghost, text, etc.]

**Icons:**
- Style: [rounded/square/circular] ([Tailwind class])
- Primary Icon Color: [color]
- Sizes: [24px / 32px / 48px]
- Container: [if applicable]

**List Items:**
- Layout: [Left element] + [Title size/weight] + [Subtitle size/color] + [Right element]
- Height: [px]
- Divider: [position, style]
- Active State: [interaction feedback]

**Shadows:**
- Style: [subtle/prominent] ([Tailwind classes])
- Usage: [where shadows apply]

## 3. Layout Structure (Mobile View - [width]px)

**Top Navigation Bar ([height]px):**
- Title: [position, size, weight, color]
- Left/Right Icons: [elements]
- Background: [color]
- Border: [style and position]

**Content Sections:**
[Describe each major section with:]
- Section Name
- Layout Pattern: [grid/flex/list]
- Elements: [what it contains]
- Styling: [colors, spacing, borders]

**Quick Access Area:**
[If applicable]
- Grid: [columns x rows]
- Icon Style: [shape, size]
- Text: [size, position]

**Data Display Cards:**
[If applicable]
- Metrics: [list key data points]
- Number Display: [size, weight]
- Label Style: [size, color]
- Layout: [horizontal/vertical/grid]

**Feature List:**
[If applicable]
- Section Title: [size, color, alignment]
- List Items: [icon + text layout]
- Item Height: [px]
- Interaction: [tap feedback]

**Bottom TabBar ([height]px, Sticky):**
- Tab Count: [number]
- Tabs: [list tab names]
- Active State: [color, style]
- Inactive State: [color, style]
- Badges: [if applicable]
- Position: [fixed/sticky]

## 4. Specific Page Content
[Include actual content for the prototype:]

**Page Title:** [name]

**Section 1: [Name]**
- Element 1: [description]
- Element 2: [description]
...

**Section 2: [Name]**
- [Actual text and data to display]

**Bottom Navigation:**
- [List all tabs with states]
```

---

### 5. Implementation Details
Provide technical implementation guidelines.

**Template:**
```
# Implementation Details
- Page Width: [max-width] and [alignment]
- Layout System: [Flexbox/Grid/both]
- Navigation: [fixed/sticky positions]
- Spacing: [margins, padding, gaps]
- Typography: [font family, weights, sizes]
- Interactive States: [hover, active, focus effects]
- Icons: [source and usage]
- Borders: [colors and positions]
```

**Example:**
```
# Implementation Details
- Page width set to max-w-[375px] and centered to simulate phone screen
- Use Flexbox and Grid layout systems
- Top navigation bar: sticky top-0 z-10
- Bottom TabBar: fixed bottom-0 w-full
- Card spacing: 12px, page left/right margins: 16px
- Typography: System default sans-serif, titles use font-semibold or font-bold
- All clickable elements have tap feedback: active:opacity-90 or active:bg-gray-50
- Icons: FontAwesome or placeholder div + background color
- Dividers: border-gray-200 or border-gray-100
```

---

### 6. Tailwind Configuration
If using Tailwind CSS, provide custom configuration.

**Template:**
```
# Tailwind Config (in <script> tag)

```javascript
tailwind.config = {
  theme: {
    extend: {
      colors: {
        'brand-primary': '[hex]',
        'brand-secondary': '[hex]',
        // ... more colors
      },
      spacing: {
        // custom spacing if needed
      },
      fontSize: {
        // custom font sizes if needed
      }
    }
  }
}
```
```

---

### 7. Content Structure & Hierarchy
Visualize the page structure and content hierarchy.

**Template:**
```
# Example Content & Structure

[Page Name]
├─ [Section 1 Name]
│  ├─ [Element 1]
│  ├─ [Element 2]
│  └─ [Element 3]
├─ [Section 2 Name]
│  ├─ [Subsection A]
│  │  ├─ [Item 1]
│  │  └─ [Item 2]
│  └─ [Subsection B]
└─ [Section 3 Name]
    └─ [Elements]
```

**Example:**
```
# Example Content & Structure

Work Dashboard
├─ Quick Access (4-grid)
│  ├─ Manage Enterprise
│  ├─ Select Apps
│  ├─ Find Services
│  └─ Submit Requests
├─ Feature Card
│  └─ Customer Data Stats (14 customers, 1 new, ¥0.00)
├─ Common Features List
│  ├─ Customer Contact
│  ├─ Customer Moments
│  ├─ Customer Groups
│  ├─ WeChat Service
│  └─ Broadcast Assistant
└─ Bottom TabBar
    ├─ Messages
    ├─ Email
    ├─ Documents
    ├─ Workspace (current)
    └─ Contacts (16)
```

---

### 8. Special Requirements
Highlight unique considerations or constraints.

**Template:**
```
# Special Requirements

- [Design System] Key Points:
  - [Characteristic 1]
  - [Characteristic 2]
  - [Characteristic 3]

- [Primary Color] Application Scenarios:
  - [Usage 1]
  - [Usage 2]
  - [Usage 3]

- Interaction Details:
  - [Interaction 1]: [behavior]
  - [Interaction 2]: [behavior]
  - [Interaction 3]: [behavior]

- Accessibility:
  - [Requirement 1]
  - [Requirement 2]

- Performance:
  - [Consideration 1]
  - [Consideration 2]
```

---

### 9. Output Format
Specify the exact deliverable format.

**Template:**
```
# Output Format

Please output complete [file type] code, ensuring:
1. [Requirement 1]
2. [Requirement 2]
3. [Requirement 3]
...

The output should be production-ready and viewable at [viewport size] on [device type].
```

**Example:**
```
# Output Format

Please output complete index.html code, ensuring:
1. Perfect display on 375px width mobile screen
2. All interactive elements have proper feedback
3. Real image assets (no placeholders)
4. Proper iOS status bar and safe area simulation
5. Smooth scrolling and fixed navigation elements

The output should be immediately viewable in a browser and represent a pixel-perfect work dashboard homepage.
```

---

## Prompt Generation Workflow

When generating a prototype prompt, follow this workflow:

### Step 1: Gather Requirements
Ask the user:
- What type of application? (e.g., enterprise tool, e-commerce, social media)
- Target platform? (iOS, Android, Web, WeChat Mini Program)
- Design style preference? (WeChat Work, iOS Native, Material Design, custom)
- Key features and pages needed?
- Target users and use cases?
- Any specific content or data to display?
- Brand colors or visual preferences?

### Step 2: Select Design System
Based on requirements, choose the appropriate design system from references/design-systems.md:
- **WeChat Work**: Chinese enterprise apps, B2B tools
- **iOS Native**: iOS apps, Apple ecosystem
- **Material Design**: Android apps, Google ecosystem
- **Ant Design Mobile**: Enterprise mobile apps with complex forms

### Step 3: Structure the Prompt
Using the sections above:
1. Define the role (UI/UX engineer with specific expertise)
2. Describe the task (what to build, design style, key attributes)
3. Specify tech stack (frameworks, device, assets, CDNs)
4. Detail visual design (colors, components, layout, content)
5. Provide implementation guidelines (technical details)
6. Include Tailwind config (if using Tailwind)
7. Show content hierarchy (visual tree structure)
8. Add special requirements (design system specifics, interactions)
9. Specify output format (deliverable expectations)

### Step 4: Populate with Specifics
- Replace all template placeholders with actual values
- Include real content (text, numbers, labels)
- Specify exact colors (hex codes)
- Define precise measurements (px, rem, etc.)
- List all page elements and features
- Add interaction states and behaviors

### Step 5: Review and Refine
Ensure the prompt:
- Is complete and self-contained
- Has no ambiguous placeholders
- Includes all necessary technical details
- Specifies exact design requirements
- Provides realistic content examples
- Defines clear deliverables

---

## Quality Checklist

Before finalizing a prototype prompt, verify:

**Completeness:**
- [ ] Role clearly defined with relevant expertise
- [ ] Task explicitly states what to build and design style
- [ ] All tech stack components listed with versions/CDNs
- [ ] Complete color palette with hex codes
- [ ] All UI components specified with dimensions and styles
- [ ] Page layout fully described with measurements
- [ ] Actual content provided (not placeholders)
- [ ] Implementation details cover technical requirements
- [ ] Tailwind config included (if applicable)
- [ ] Content hierarchy visualized
- [ ] Special requirements and interactions documented
- [ ] Output format clearly defined

**Clarity:**
- [ ] No ambiguous terms or vague descriptions
- [ ] Measurements specified (px, rem, %, etc.)
- [ ] Colors defined with hex codes
- [ ] Component states described (normal, hover, active, disabled)
- [ ] Layout relationships clear (parent-child, spacing, alignment)

**Specificity:**
- [ ] Design system explicitly named
- [ ] Viewport dimensions provided
- [ ] Typography scale defined (sizes, weights, line heights)
- [ ] Interactive behaviors documented
- [ ] Edge cases considered (long text, empty states, etc.)

**Realism:**
- [ ] Real content examples (not Lorem Ipsum)
- [ ] Authentic data points (numbers, names, dates)
- [ ] Practical feature set (not overengineered)
- [ ] Appropriate complexity for use case

**Technical Accuracy:**
- [ ] Valid Tailwind class names
- [ ] Correct CDN links
- [ ] Proper HTML structure implied
- [ ] Feasible layout techniques (Flexbox/Grid)
- [ ] Accessible markup considerations

---

## Example Variations

### Minimal Prompt (for simple landing page)
```
# Role
You are a frontend developer specializing in clean, modern web design.

# Task
Create a single-page landing page (HTML + Tailwind CSS) for a SaaS product.
Style: Modern, minimal, professional with soft colors.

# Tech Stack
- Single index.html file
- Tailwind CSS (CDN)
- Desktop-first responsive (max 1200px)
- Use Unsplash for hero image

# Visual Design
- Colors: Primary #6366F1 (indigo), Background #F9FAFB (gray-50), Text #111827 (gray-900)
- Sections: Hero (full viewport height) + Features (3 columns) + CTA
- Typography: System sans-serif, Hero 48px bold, Body 18px regular

# Output Format
Single HTML file, production-ready, mobile-responsive.
```

### Complex Prompt (for multi-page app)
[Follow full template structure with all sections, detailed component specs, multiple pages, complex interactions]

---

## Tips for Effective Prompts

1. **Be Specific**: "Blue button" → "Primary button: #3478F6 background, white text, 44px height, 4px rounded corners"

2. **Provide Context**: Don't just list features, explain their purpose and how they relate to user workflows

3. **Use Real Content**: "Insurance agent dashboard" with "14 customers, ¥1,234.56 revenue" is better than "Dashboard with metrics"

4. **Reference Standards**: "Follow WeChat Work design guidelines" gives Claude a comprehensive style framework

5. **Include Edge Cases**: Mention empty states, loading states, error states, long text handling

6. **Specify Interactions**: Don't assume—explicitly state tap feedback, hover effects, animation timing

7. **Think Mobile-First**: For mobile apps, always specify touch target sizes (minimum 44px), safe areas, gesture support

8. **Consider Accessibility**: Mention color contrast, touch target sizes, semantic HTML, ARIA labels

9. **Balance Detail and Brevity**: Be thorough but organized—use sections and hierarchy to keep prompts scannable

10. **Test Your Mental Model**: Can someone else understand exactly what to build from this prompt? If not, add clarity.
