# Next release — collected fixes

## v2.8.1 — visible Dashboard progress bars

Fix missing painted fill boxes in Dashboard energy and nutrient meters. Fills now have block layout, full track height, rounded ends and theme-aware colors; carbohydrate/fiber colors match their dots. Existing percentages and 0–100% visual clamping are unchanged. No API, data or target calculation changes.

Local Chrome checks verified the 469/2000 kcal example at 23.45%, empty/partial/full tracks, light/dark colors and a 390px viewport without horizontal overflow. Production build and 56 frontend regressions passed. Published release notes establish deployment status.

## v2.8.0 — Food Library and focused nutrition

Food Library/Recipes hierarchy, combined My foods/Quick Log/Search workflow, app-wide tracked nutrients, account email display, public wording cleanup and automatic seven-day Analysis loading. See [scope and compatibility](FOOD_LIBRARY_AND_TRACKED_NUTRIENTS.md) and [release verification](RELEASE_2_8_0.md). Published GitHub release notes establish deployment status. Emailed PDF Analysis reports are recorded for Phase 3 only.

Includes app-wide accessible checkbox styling, larger labelled option rows, selected/unsaved nutrient counts, clearer Quick Log and Settings wording, and a proper fractional recipe-serving dialog. Reviewed locally in Chrome on desktop/mobile and light/dark themes.

## v2.7.1 — schedule wording and app-wide scrollbars

See [release scope and verification](RELEASE_2_7_1.md). GitHub release notes establish actual publication/deployment status.

- Put **Repeat every [number] day/week/month(s)** beside **Repeat pattern**, not under Advanced options. Explain 1 versus 2 directly beneath it.
- Show a live **Your schedule** preview that distinguishes the repeated dates from the number of occurrences on each date. For example: “Every other day. Twice on each scheduled day, at 08:00 and 20:00.”
- Rename the times section to **Times on each scheduled day** and the add action to **Add another time on the same day**. Use matching wording in the schedule editor and history.
- Apply one shared scrollbar style across all application pages and scrollable elements, including dialogs, tables/charts, text areas, menus, lists and sidebars, vertically and horizontally. Use thin theme-matched thumbs, transparent tracks, rounded WebKit fallbacks and system styling in forced-color mode. Reserve stable gutters only on the viewport/dialogs to avoid shrinking other controls. Native OS/browser-owned popup controls may retain system appearance. Keep scrolling available; no recurrence-engine or backend changes.

Local verification: 53 frontend regressions, production build and existing recurrence-engine tests passed. Chrome checks covered the live alternate-day/twice-per-day preview, zero-interval validation and desktop/390px mobile dialog layout. A disposable local schedule produced two occurrences on 23 and 25 October, with none on 24 October, matching the preview. Computed styles confirmed matching thin, transparent-track scrollbars on the viewport, sidebar, tables, chart containers and summary textarea; light/dark theme colors updated together and the original System preference was restored. No production records were changed.

## Delivered — v2.7.0

These changes are bundled for v2.7.0. Source implementation and local verification are recorded in [the release scope](PHASE_2_7_RELEASE.md); publication and production deployment are confirmed by the published GitHub release notes.

## Health occurrence actions

- Replace the **More actions** disclosure with a directly visible **Reschedule** action wherever rescheduling is available, because it currently contains only that one action.
- Preserve existing eligibility: completed occurrences must not gain a Reschedule action. Keep Mark done and Skip behavior unchanged.
- Verify direct access on desktop and mobile, including Active and Missed/“Skipped by you” occurrences.

## Combined analysis

- Shared timeline above Nutrition analysis, with **Explore** refreshing nutrition and health item charts.
- Health item analysis before Text Summary, with a settings/plus picker for account-saved task, medication and supplement selections.
- Per-item daily status and completion charts, exact daily tables, and explicit timezone/reset/rescheduling/future-event semantics.
