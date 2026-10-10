# AteThat web app

React, TypeScript, and Vite frontend. The Stitch export at the repository root is a visual reference for layout, typography, spacing, and light/dark colour treatment. Product features and API behaviour come from `docs/3. SYSTEM_DESIGN.md` and `docs/5. API_DESIGN.md`.

Run `npm install` from the repository root, then `npm run dev:web`. Set `VITE_API_BASE_URL` in `apps/web/.env.local` when the API Service is available. The public client must never contain secrets.

The current frontend foundation includes the responsive shell, theme system, landing page, authentication forms, and layouts for the main product sections. Screens that need backend data show explicit empty states; they do not simulate saved records or successful API operations.
