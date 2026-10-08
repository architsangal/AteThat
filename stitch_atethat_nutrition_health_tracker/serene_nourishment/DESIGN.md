---
name: Serene Nourishment
colors:
  surface: '#f8f9ff'
  surface-dim: '#cbdbf5'
  surface-bright: '#f8f9ff'
  surface-container-lowest: '#ffffff'
  surface-container-low: '#eff4ff'
  surface-container: '#e5eeff'
  surface-container-high: '#dce9ff'
  surface-container-highest: '#d3e4fe'
  on-surface: '#0b1c30'
  on-surface-variant: '#43474e'
  inverse-surface: '#213145'
  inverse-on-surface: '#eaf1ff'
  outline: '#74777f'
  outline-variant: '#c4c6cf'
  surface-tint: '#455f87'
  primary: '#022448'
  on-primary: '#ffffff'
  primary-container: '#1e3a5f'
  on-primary-container: '#8aa4cf'
  inverse-primary: '#adc8f5'
  secondary: '#236960'
  on-secondary: '#ffffff'
  secondary-container: '#a9ede1'
  on-secondary-container: '#286d64'
  tertiary: '#3f1900'
  on-tertiary: '#ffffff'
  tertiary-container: '#602900'
  on-tertiary-container: '#ef8946'
  error: '#ba1a1a'
  on-error: '#ffffff'
  error-container: '#ffdad6'
  on-error-container: '#93000a'
  primary-fixed: '#d5e3ff'
  primary-fixed-dim: '#adc8f5'
  on-primary-fixed: '#001c3b'
  on-primary-fixed-variant: '#2d486d'
  secondary-fixed: '#acefe4'
  secondary-fixed-dim: '#90d3c8'
  on-secondary-fixed: '#00201c'
  on-secondary-fixed-variant: '#005049'
  tertiary-fixed: '#ffdbc9'
  tertiary-fixed-dim: '#ffb68c'
  on-tertiary-fixed: '#321200'
  on-tertiary-fixed-variant: '#753400'
  background: '#f8f9ff'
  on-background: '#0b1c30'
  surface-variant: '#d3e4fe'
typography:
  headline-xl:
    fontFamily: Plus Jakarta Sans
    fontSize: 2.25rem
    fontWeight: '600'
    lineHeight: 2.75rem
    letterSpacing: -0.025em
  headline-xl-mobile:
    fontFamily: Plus Jakarta Sans
    fontSize: 1.75rem
    fontWeight: '600'
    lineHeight: 2.25rem
    letterSpacing: -0.02em
  headline-lg:
    fontFamily: Plus Jakarta Sans
    fontSize: 1.5rem
    fontWeight: '600'
    lineHeight: 2rem
    letterSpacing: -0.02em
  headline-md:
    fontFamily: Plus Jakarta Sans
    fontSize: 1.25rem
    fontWeight: '600'
    lineHeight: 1.75rem
    letterSpacing: -0.015em
  headline-sm:
    fontFamily: Plus Jakarta Sans
    fontSize: 1.125rem
    fontWeight: '600'
    lineHeight: 1.5rem
    letterSpacing: -0.01em
  body-lg:
    fontFamily: Inter
    fontSize: 1.125rem
    fontWeight: '400'
    lineHeight: 1.75rem
  body-md:
    fontFamily: Inter
    fontSize: 1rem
    fontWeight: '400'
    lineHeight: 1.5rem
  body-sm:
    fontFamily: Inter
    fontSize: 0.875rem
    fontWeight: '400'
    lineHeight: 1.25rem
  label-lg:
    fontFamily: Inter
    fontSize: 0.875rem
    fontWeight: '500'
    lineHeight: 1.25rem
    letterSpacing: 0.01em
  label-md:
    fontFamily: Inter
    fontSize: 0.75rem
    fontWeight: '500'
    lineHeight: 1rem
    letterSpacing: 0.02em
  label-sm:
    fontFamily: Inter
    fontSize: 0.6875rem
    fontWeight: '600'
    lineHeight: 0.875rem
    letterSpacing: 0.04em
  data-metric:
    fontFamily: Inter
    fontSize: 1.375rem
    fontWeight: '600'
    lineHeight: 1.625rem
    letterSpacing: -0.02em
rounded:
  sm: 0.25rem
  DEFAULT: 0.5rem
  md: 0.75rem
  lg: 1rem
  xl: 1.5rem
  full: 9999px
spacing:
  gutter: 1.5rem
  gutter-mobile: 1rem
  margin: 2rem
  margin-mobile: 1rem
  space-xs: 0.25rem
  space-sm: 0.5rem
  space-md: 1rem
  space-lg: 1.5rem
  space-xl: 2rem
  space-2xl: 3rem
---

## Brand & Style

This design system embodies a serene, clinical-adjacent yet deeply compassionate health journal. Built around mindful sustenance, it treats food, medication, and supplements with equal dignity, moving entirely away from guilt-driven calorie counting, gym-bro gamification, neon accents, or clinical hostility.

The emotional core is **grounded intentionality**: clear, quiet, and reliable. The aesthetic merges contemporary minimalism with soft editorial warmth—employing generous negative space, gentle tinted containers, precision tabular data, and purposeful tonal cues. Visuals never yell or scold; instead, they inform and reassure.

## Colors

The palette balances deep botanical indigo authority with restorative sage and mindful nutrient markers.

### Role Tokens & Context

#### Canvas & Surfaces
- **Light Mode Canvas:** `#FAFCFE` base with `#F4F7FB` backdrop tint for sub-panes.
- **Light Mode Surfaces:** `#FFFFFF` on `#FAFCFE` for top-level cards, `#EDF2F7` for nested sections, `#E4ECF4` for interactive chip states.
- **Dark Mode Canvas:** `#0B1017` deep nocturnal ground, `#111722` structural layer.
- **Dark Mode Surfaces:** `#17212F` for elevated cards, `#1E2B3E` for interactive/selected states.
- **Borders:** `#DCE5EE` (Light) and `#223043` (Dark) provide soft, razor-sharp structural outlines without heaviness.

#### Typography & Content
- **Light Mode:** Primary text `#0F172A` (deepest slate), secondary text `#475569`, tertiary/meta text `#64748B`.
- **Dark Mode:** Primary text `#F1F5F9` (crisp chalk white), secondary text `#94A3B8`, tertiary `#64748B`.

#### Nutrient & Domain Indicators
Nutrient tokens must maintain a balanced, desaturated presence:
- **Energy / Calories:** Deep Indigo Slate `#1E3A5F` (Light) / `#60A5FA` (Dark) or subtle Amber-Gold `#D97706`.
- **Protein:** Slate Blue `#4B6B94`.
- **Carbohydrates:** Warm Ochre `#C28338`.
- **Fats:** Soft Rose-Coral `#C96E69`.
- **Dietary Fibre:** Muted Sage Green `#4E8773`.
- **Medications & Supplements:** Calming Amethyst Slate `#6366F1` and Medical Teal `#2E8B83`.

## Typography

The type system blends the human, organic curves of Plus Jakarta Sans for structural headings with the clinical precision and legibility of Inter for body copy, tabular nutrients, dose metrics, and timestamps.

- **Tabular Figures:** Always apply `font-feature-settings: 'tnum' on, 'cv05' on` on nutrient amounts, macronutrient counts, pill dosages, and clock stamps (e.g., `08:30 IST UTC+05:30`) to avoid visual jittering across lists.
- **Hierarchy Rhythm:** Headings remain balanced and understated. Large displays are restricted to summary dashboards and morning check-ins.
- **Metadata:** All labels and micro-metrics use uppercase or medium-weight title-case with slight tracking (`letter-spacing: 0.02em - 0.04em`) to ensure legibility on tinted badges and chip indicators.

## Layout & Spacing

A disciplined, fluid 12-column grid is enforced on desktop and tablet, collapsing to a single-column layout on mobile devices.

### Breakpoints & Canvas Bounds
- **Mobile (< 640px):** 4 columns, 16px outer margin, 16px column gutters. Focus on single-column feed streams and stacked nutrient cards.
- **Tablet (640px - 1024px):** 8 columns, 24px outer margin, 20px column gutters. Split view: summary ledger on left, log stream on right.
- **Desktop (> 1024px):** 12 columns, max-width of 1280px centered, 32px outer margin, 24px column gutters. Three-tier layout: navigation rail, central feed/entry, and macro/chronobiology sidebar.

### Spacing Rhythm
Vertical cadence relies strictly on consistent 8px multiples. Tight associations (e.g., nutrient label to metric value) utilize `space-xs` (4px) or `space-sm` (8px). Structural section dividers and feed logs decouple at `space-lg` (24px) to `space-xl` (32px), creating a spacious, relaxed breathing room.

## Elevation & Depth

This design system avoids heavy drop shadows and high-contrast bevels. Depth is articulated through **tonal stacking, subtle surface tints, and fine-line borders**.

### The Layering Principle
1. **Canvas Layer (Base):** Light (`#FAFCFE`) / Dark (`#0B1017`). The ambient, calm foundation.
2. **Structural Layer (Cards, Timelines):** Light (`#FFFFFF` or `#EDF2F7`) / Dark (`#17212F`). Outlined with a 1px border (`#DCE5EE` in light, `#223043` in dark).
3. **Elevated Overlays (Sheets, Drawers, Modals):** Light (`#FFFFFF`) with a warm, diffuse ambient shadow:
   `box-shadow: 0 10px 30px -10px rgba(30, 58, 95, 0.08), 0 1px 3px rgba(30, 58, 95, 0.04)`.
   In Dark Mode:
   `box-shadow: 0 12px 32px -8px rgba(0, 0, 0, 0.5), 0 0 0 1px #223043`.

### Interactive Depths
- Cards do not push outward toward the user on hover. Instead, their border transitions smoothly to primary slate-blue (`#1E3A5F` / `#3A7D74` at 40% opacity) and background shifts subtly (e.g., Light Mode `#EDF2F7` -> `#E4ECF4`), preserving calmness.

## Shapes

The design system employs a soft, approachable geometry without turning childlike or overly pill-shaped for structural components.

- **Cards & Dashboard Panels:** Standardized at `rounded-xl` (16px / 1rem) to frame data with soft precision.
- **Interactive Buttons & Form Inputs:** Formed with `rounded-lg` (8px to 12px / 0.5rem to 0.75rem), producing an intentional, stable boundary for interaction.
- **Pills, Status Badges & Timezone Tags:** Full pills (`border-radius: 9999px`) are reserved exclusively for contextual tags, dosage pills, and macro proportion chips.

## Components

### Buttons
- **Primary:** Deep slate botanical indigo (`#1E3A5F`) fill with white text (`#FFFFFF`). `rounded-lg`, font weight 500. Focus rings use `#3A7D74` with 2px offset.
- **Secondary / Ghost:** Clear background with a 1px border (`#DCE5EE` light, `#223043` dark), text in `#1E3A5F` (light) or `#F1F5F9` (dark). Hover transitions background to `#EDF2F7` or `#17212F`.
- **Destructive:** Soft muted crimson tint (`#FEE2E2` bg with `#991B1B` text) rather than stark red alerts.

### Input Fields & Selectors
- Standard height of 44px for effortless touch ergonomics.
- Bordered in `#DCE5EE` with `#FFFFFF` background (light) and `#223043` with `#17212F` background (dark).
- On focus: Crisp 1.5px border transition to `#3A7D74` without jarring glowing rings.
- Helper text and units (e.g., `g`, `mg`, `kcal`, `ml`) are anchored on the right edge with `font-feature-settings: 'tnum'`, set in `#64748B`.

### Nutrient & Metric Rings / Progress Bars
- Background track is muted: `#E4ECF4` in light mode, `#1E2B3E` in dark mode.
- Foreground fills use the exact semantic nutrient tokens (Energy, Protein, Carbs, Fat, Fibre).
- Rounded ends (`stroke-linecap: round`), accompanied directly by tabular numbers (`e.g., 42g / 75g`) rather than ambiguous radial fills alone.

### Cards & Feed Items
- Meal and supplement log items live inside structural cards.
- Left-hand anchor contains a quiet timestamp with the timezone tag (`07:45 · IST UTC+05:30`) in `label-md`.
- Content body clearly separates name, brand/source, and dose or macro pill stack.
- Completed supplement or medication items feature a muted teal-sage check indicator; pending doses use a soft circular dot.

### Chips & Timezone Tags
- **Timezone & Temporal Badges:** Rendered in small mono/tabular font with `border-radius: 9999px`, background `#EDF2F7` (light) or `#1E2B3E` (dark), padding `2px 8px`.
- **Status Chips:** Delicate solid background tinted to match status with 10% opacity, paired with high-contrast text of the same hue family.