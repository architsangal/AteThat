# v2.7.1 — schedule clarity and app-wide scrollbars

## Changes

- Repeat pattern and **Repeat every [number] day/week/month(s)** are visible together. Interval 1 means every period; interval 2 means every other period. The interval selects repeated dates, not the number of occurrences per day.
- **Times on each scheduled day** and **Add another time on the same day** explain multiple daily occurrences. A live **Your schedule** preview combines the repeat pattern, selected weekdays/month dates, times and effective start date.
- Creation, schedule editing, routine summaries and history use the same cadence wording. Existing schedule versioning and recurrence calculations are unchanged.
- Shared thin, theme-matched scrollbars apply to the viewport and all application scroll containers, vertically and horizontally. Transparent tracks and rounded WebKit fallback thumbs reduce visual clutter. Stable gutters apply only where needed; high-contrast mode retains system styling. Native OS/browser-owned dropdown popups may retain system appearance.

## Verification

- Production build, 53 frontend regressions and existing Core recurrence tests passed locally.
- Chrome checks covered interval validation, alternate-day/twice-per-day previews, saved dates, weekly/monthly edits and preservation of prior schedule versions using a disposable local fixture.
- Desktop and 390px mobile dialog layout were checked. Computed styles confirmed the shared scrollbar treatment on the viewport, sidebar, tables, chart containers and summary textarea. Light/dark colors updated together; System theme was restored after testing. No console errors were observed.
- No backend, contract, infrastructure or database migration changes. Production health records are not modified for sanity checks.

Published GitHub release notes record the promoted commit, required CI results and actual production deployment verification.
