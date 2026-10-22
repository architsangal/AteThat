# Next release — collected fixes

These changes are bundled for v2.7.0. Source implementation and local verification are recorded in [the release scope](PHASE_2_7_RELEASE.md); publication and production deployment are confirmed by the published GitHub release notes.

## Health occurrence actions

- Replace the **More actions** disclosure with a directly visible **Reschedule** action wherever rescheduling is available, because it currently contains only that one action.
- Preserve existing eligibility: completed occurrences must not gain a Reschedule action. Keep Mark done and Skip behavior unchanged.
- Verify direct access on desktop and mobile, including Active and Missed/“Skipped by you” occurrences.

## Combined analysis

- Shared timeline above Nutrition analysis, with **Explore** refreshing nutrition and health item charts.
- Health item analysis before Text Summary, with a settings/plus picker for account-saved task, medication and supplement selections.
- Per-item daily status and completion charts, exact daily tables, and explicit timezone/reset/rescheduling/future-event semantics.
