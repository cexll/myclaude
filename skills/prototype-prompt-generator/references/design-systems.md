# Design Systems Reference

This document provides detailed specifications for common design systems used in mobile and web applications.

## 企业微信 (WeChat Work) Style

### Core Characteristics
- **Simplicity & Professionalism**: Clean interface with clear information hierarchy
- **Operational Clarity**: Clear action paths with explicit next steps
- **Tech Blue**: Primary color scheme emphasizing efficiency and trust

### Color Palette
```
Primary Colors:
- Tech Blue: #3478F6 (buttons, icons, emphasis)
- Link Blue: #576B95 (hyperlinks)
- Alert Red: #FA5151 (warnings, errors)
- Warning Orange: #FF976A (cautions)

Neutral Colors:
- Title Black: #191919 (headings)
- Text Gray: #333333 (body text)
- Light Text: #999999 (secondary text)
- Divider: #E5E5E5 (borders, separators)
- Background Area: #F7F8FA (section backgrounds)
- White: #FFFFFF (card backgrounds)
```

### Component Specifications

#### Cards
- Background: White (#FFFFFF)
- Border Radius: 8px (rounded-lg)
- Shadow: Subtle (shadow-sm)
- No thick borders
- Spacing: 12px between cards

#### Buttons
- Primary: Blue background (#3478F6), white text
- Height: 44px
- Border Radius: 4px (rounded)
- Active State: 90% opacity (active:opacity-90)

#### Icons
- Style: Rounded square background (rounded-lg) or pure icon
- Primary Icons: Tech Blue (#3478F6)
- Sizes: 24px / 32px / 48px

#### List Items
- Layout: Left icon/avatar (48px) + Title (15px bold) + Subtitle (13px gray) + Right arrow
- Height: 64px
- Divider: Left indent 64px (aligned with content)
- Active State: Gray background (active:bg-gray-50)

#### Navigation Bar (Top)
- Height: 44px
- Title: Centered, 16px bold, deep black
- Background: White with 1px bottom border (border-gray-200)
- Icons: Search and menu icons on right

#### Tab Bar (Bottom)
- Height: 50px
- Layout: Icon + Text (vertical)
- Selected: Tech Blue (#3478F6)
- Unselected: Gray
- Position: Fixed bottom (fixed bottom-0)
- Badge Support: Red circular badge with white text

### Typography
- System Default: Sans-serif
- Headings: font-semibold or font-bold
- Title: 16px bold
- Body: 15px
- Caption: 13px
- Helper Text: 13px gray

### Spacing & Layout
- Page Margins: 16px (left/right)
- Card Spacing: 12px
- Content Padding: 16px
- Mobile Width: 375px (max-w-[375px])

### Interaction Patterns
- Tap Feedback: Background changes to light gray (bg-gray-50)
- Button Press: Opacity reduces to 90%
- Card Elevation: Subtle shadow (shadow-sm)
- List Navigation: Right arrow indicator

---

## iOS Native Style

### Core Characteristics
- **Minimalism**: Clean, spacious layouts with generous whitespace
- **Clarity**: Clear visual hierarchy and focus on content
- **Depth**: Subtle shadows and layering to create depth

### Color Palette
```
System Colors:
- Blue: #007AFF (primary actions, links)
- Green: #34C759 (success, positive actions)
- Red: #FF3B30 (destructive actions, errors)
- Orange: #FF9500 (warnings)
- Yellow: #FFCC00 (cautions)
- Gray: #8E8E93 (secondary text)

Background Colors:
- System Background: #FFFFFF (light mode), #000000 (dark mode)
- Secondary Background: #F2F2F7 (light mode), #1C1C1E (dark mode)
- Tertiary Background: #FFFFFF (light mode), #2C2C2E (dark mode)

Text Colors:
- Primary: #000000 (light mode), #FFFFFF (dark mode)
- Secondary: #3C3C43 (60% opacity)
- Tertiary: #3C3C43 (30% opacity)
```

### Component Specifications

#### Navigation Bar
- Height: 44px (compact), 96px (large title)
- Title: 34px bold (large), 17px semibold (inline)
- Background: Translucent blur effect
- Buttons: Text or icon, 17px regular

#### Tab Bar
- Height: 49px
- Layout: Icon + Text (vertical, 10pt text)
- Selected: Blue (#007AFF)
- Unselected: Gray (#8E8E93)
- Position: Fixed bottom
- Blur Effect: Translucent background

#### List/Table View
- Cell Height: 44px minimum
- Layout: Left accessory + Title + Detail + Right accessory/chevron
- Separator: 0.5px hairline, inset from left
- Active State: Gray highlight (bg-gray-100)

#### Buttons
- Primary: Blue background (#007AFF), white text, 50px height, rounded corners (10px)
- Secondary: No background, blue text
- Destructive: Red text
- Corner Radius: 10px (rounded-lg)

#### Cards
- Border Radius: 10px (rounded-lg)
- Background: White (light mode), #1C1C1E (dark mode)
- Shadow: Subtle (0 1px 3px rgba(0,0,0,0.1))
- Padding: 16px

### Typography (San Francisco Font)
- Large Title: 34px bold
- Title 1: 28px regular
- Title 2: 22px regular
- Title 3: 20px regular
- Headline: 17px semibold
- Body: 17px regular
- Callout: 16px regular
- Subheadline: 15px regular
- Footnote: 13px regular
- Caption 1: 12px regular
- Caption 2: 11px regular

### Spacing
- Edge Margins: 16px (standard), 20px (large screens)
- Inter-Element Spacing: 8px / 16px / 24px
- Section Spacing: 32px

### Interaction Patterns
- Tap Feedback: Subtle highlight (no ripple effect)
- Swipe Gestures: Swipe to delete, swipe back navigation
- Pull to Refresh: System spinner at top
- Haptic Feedback: Light impact on selection

---

## Material Design (Google) Style

### Core Characteristics
- **Material Metaphor**: Physical surfaces and realistic motion
- **Bold Graphics**: Intentional use of color, imagery, typography
- **Motion**: Meaningful animations that provide feedback

### Color Palette
```
Primary Colors:
- Primary: #6200EE (brand color, main actions)
- Primary Variant: #3700B3 (darker shade)
- Secondary: #03DAC6 (accent color, floating actions)
- Secondary Variant: #018786 (darker accent)

Status Colors:
- Error: #B00020 (errors, alerts)
- Success: #4CAF50 (success states)
- Warning: #FF9800 (warnings)
- Info: #2196F3 (informational)

Background/Surface:
- Background: #FFFFFF (light), #121212 (dark)
- Surface: #FFFFFF (light), #121212 (dark)
- Surface Variant: #F5F5F5 (light), #1E1E1E (dark)

Text Colors:
- High Emphasis: #000000 (87% opacity)
- Medium Emphasis: #000000 (60% opacity)
- Disabled: #000000 (38% opacity)
```

### Component Specifications

#### App Bar (Top)
- Height: 56px (mobile), 64px (desktop)
- Title: 20px medium weight
- Background: Primary color or white
- Icons: 24px, white or primary color
- Elevation: 4dp

#### Bottom Navigation
- Height: 56px
- Layout: Icon (24px) + Label (12px)
- Active: Primary color with ripple effect
- Inactive: 60% opacity
- Elevation: 8dp

#### Cards
- Border Radius: 4px (rounded)
- Background: White (light), #1E1E1E (dark)
- Elevation: 1dp (resting), 8dp (raised)
- Padding: 16px

#### Buttons
- Contained: Primary color background, white text, 36px height, 4px radius
- Outlined: 1px border, primary color text, transparent background
- Text: No border/background, primary color text
- Corner Radius: 4px (rounded)
- Ripple Effect: Material ripple on tap

#### Lists
- Item Height: 48px (single-line), 64px (two-line), 88px (three-line)
- Left Icon: 24px (40px container)
- Right Icon: 24px
- Divider: 1px, 87% opacity
- Ripple Effect: On tap

#### Floating Action Button (FAB)
- Size: 56px (standard), 40px (mini)
- Shape: Circular
- Color: Secondary color
- Elevation: 6dp (resting), 12dp (pressed)
- Icon: 24px white

### Typography (Roboto Font)
- H1: 96px light
- H2: 60px light
- H3: 48px regular
- H4: 34px regular
- H5: 24px regular
- H6: 20px medium
- Subtitle 1: 16px regular
- Subtitle 2: 14px medium
- Body 1: 16px regular
- Body 2: 14px regular
- Button: 14px medium, uppercase
- Caption: 12px regular
- Overline: 10px regular, uppercase

### Spacing & Grid
- Base Unit: 8dp
- Layout Grid: 8dp increments
- Margins: 16dp (mobile), 24dp (tablet)
- Gutters: 16dp (mobile), 24dp (tablet)
- Touch Target: 48dp minimum

### Elevation System
- Level 0: 0dp (background)
- Level 1: 1dp (cards at rest)
- Level 2: 2dp (buttons at rest)
- Level 3: 3dp (refresh indicator)
- Level 4: 4dp (app bar)
- Level 6: 6dp (FAB at rest)
- Level 8: 8dp (bottom nav, menus)
- Level 12: 12dp (FAB pressed)
- Level 16: 16dp (nav drawer)
- Level 24: 24dp (dialog, picker)

### Interaction Patterns
- Ripple Effect: Circular expanding animation from tap point
- State Changes: Smooth color transitions (300ms)
- Entry/Exit Animations: Fade + scale/slide
- Touch Feedback: Immediate visual response

---

## Ant Design Mobile Style

### Core Characteristics
- **Enterprise-Grade**: Professional UI for business applications
- **Efficiency-Oriented**: Optimized for quick task completion
- **Consistent**: Unified design language across platforms

### Color Palette
```
Primary Colors:
- Brand Blue: #108EE9 (primary actions)
- Link Blue: #108EE9 (links, emphasis)
- Success: #00A854 (success states)
- Warning: #FFBF00 (warnings)
- Error: #F04134 (errors, destructive actions)

Neutral Colors:
- Heading: #000000 (85% opacity)
- Body Text: #000000 (65% opacity)
- Secondary Text: #000000 (45% opacity)
- Disabled: #000000 (25% opacity)
- Border: #E9E9E9 (borders, dividers)
- Background: #F5F5F5 (page background)
- White: #FFFFFF (component background)
```

### Component Specifications

#### Navigation Bar
- Height: 45px
- Title: 18px bold, centered
- Background: #108EE9 or white
- Icons: 22px
- Border Bottom: 1px (#E9E9E9)

#### Tab Bar
- Height: 50px
- Layout: Icon (22px) + Text (10px)
- Active: Brand blue (#108EE9)
- Inactive: #888888
- Badge: Red dot or number

#### List
- Item Height: 44px minimum
- Left Icon: 22px (44px container)
- Right Arrow: 16px
- Divider: 1px, inset 15px from left
- Active State: #DDDDDD background

#### Buttons
- Default: 47px height, 5px radius
- Primary: Blue background (#108EE9), white text
- Ghost: Transparent background, blue border and text
- Disabled: Gray with 60% opacity

#### Cards
- Border Radius: 2px (rounded-sm)
- Background: White
- Border: 1px (#E9E9E9) or none
- Shadow: Optional subtle shadow
- Padding: 15px

### Typography (System Default)
- Heading: 18px bold
- Subheading: 15px bold
- Body: 14px regular
- Caption: 12px regular
- Button: 16px medium

### Spacing
- Base Unit: 5px
- Standard Spacing: 15px
- Edge Margins: 15px
- Component Padding: 15px

---

## Usage Guidelines

When generating a prototype prompt, reference the appropriate design system based on user requirements:

1. **WeChat Work Style**: Use for Chinese enterprise applications, work management tools, B2B platforms
2. **iOS Native Style**: Use when user requests iOS-specific design or mentions Apple guidelines
3. **Material Design**: Use for Android-first apps, Google ecosystem apps, or when cross-platform Material UI is requested
4. **Ant Design Mobile**: Use for enterprise mobile applications with complex data and forms

For each design system, include:
- Complete color palette with hex codes
- Component specifications (dimensions, spacing, states)
- Typography scale (sizes, weights, line heights)
- Interaction patterns (tap feedback, animations)
- Accessibility considerations
- Code examples (Tailwind classes or CSS)
