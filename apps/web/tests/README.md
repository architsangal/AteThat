# Shared spacing regression checks

With Vite running, open `http://localhost:5173/tests/spacing.html` in Chrome.
This fixture uses the app's real styles and synthetic markup only; it makes no
API requests and does not edit account data. It is not an application route or
part of the production build entry points.

Check at desktop, tablet and narrow mobile widths, in light and dark themes:

- Every visible direct page section is separated by 24px, or 18px at 600px and below.
- The empty notification wrapper takes no space.
- Success/error banners have gaps before and after them, including before a history card.
- Card/dialog padding is 24px, or 18px at 600px and below.
- Feedback inside a card has 16px separation, except at the card's padded outer edges.
- Grid forms keep their existing gap without additional feedback margins.
- Nested card grids and messages wrap without horizontal page overflow.

Page rhythm is owned by `src/features/spacing.css`, imported after feature
styles. Do not add a new page-specific margin for messages or page sections.
Use a component's internal gap for its own content, and keep spacing tokens
responsive rather than adding inline padding patches.

Also check real targets, history, food search, meals, recipes, wellness,
analysis, health and settings pages. Inspect states without modifying personal
records; use the fixture to exercise success/error combinations safely.
