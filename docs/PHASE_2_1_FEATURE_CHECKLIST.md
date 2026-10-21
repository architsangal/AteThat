# AteThat — individual feature checks

Local Phase 2.1 review, 20 October 2026. Open [the local app](http://localhost:5173). Use a disposable local account for changes to test data. This is an inventory of the current UI, not a claim that every check below was manually executed in this pass.

Start with checks **W1–W7, F1–F14, H1–H7 and U1–U9** to review the refinements requested in your screenshots. Then use the remaining checks to discover the rest of the application.

## Welcome and feature guide

- [ ] **W1 — First entry:** An account that has not completed or skipped the guide is sent to the welcome flow after signing in.
- [ ] **W2 — Optional profile:** Add a display name, optional date of birth, height, weight, tracking goal and account timezone. Save and continue.
- [ ] **W3 — Skip profile:** Continue without supplying optional personal details.
- [ ] **W4 — Skip everything:** Use “Skip setup and guide”, available near the top. Reach the dashboard without supplying extra information.
- [ ] **W5 — Food guide:** Read where to search, add, review, log food, save favourites and correct history.
- [ ] **W6 — Other tools:** Continue to foods/meals, recipes, wellness, health items, targets, analysis and settings. Back/Next starts at the step heading.
- [ ] **W7 — Persistence and replay:** Reload or sign in again after finishing/skipping; setup should not interrupt you again. Reopen it with the header help icon or Settings.

## Food search and logging

- [ ] **F1 — Local database:** Search already imported USDA foods without making a new USDA discovery request. This is a shared catalogue, not a download of all USDA foods.
- [ ] **F2 — Explicit USDA search:** Choose “Search USDA” to discover new foods. Successful results become available in the local database.
- [ ] **F3 — Account counters:** See local database size, your local searches today, remaining USDA searches and the UTC reset time.
- [ ] **F4 — Matching count:** Search and see how many records match in the selected source, independently of the size of the current page.
- [ ] **F5 — Allowance:** Each successful USDA search or next page uses one of 10 searches per account per UTC day. Empty results count; provider failures are refunded. Local searches remain available at zero remaining.
- [ ] **F6 — Preparation:** Expand Filters and try raw/cooked versus any preparation.
- [ ] **F7 — Source details:** Expand Source & nutrition. Check nutrition, source type, FDC ID and link to the USDA record. Different FDC IDs with the same name remain distinct foods.
- [ ] **F8 — Add a food:** Add one or several results. The selected-food count updates; the result row does not turn into a broad shaded rectangle.
- [ ] **F9 — Amount:** Enter a positive gram or millilitre amount and remove a selected food if needed.
- [ ] **F10 — Review:** Choose Review. Check foods, amounts, tracking day, timezone and nutrition before logging.
- [ ] **F11 — More nutrition:** Expand the additional nutrient values. Unknown values should remain unknown, independently of known zero values.
- [ ] **F12 — Edit before logging:** Return to the selected foods from review, change an amount and review again.
- [ ] **F13 — Confirm:** Choose Log food. A success message links to History; double submission must not create duplicate records.
- [ ] **F14 — Validation:** Submit an empty or one-character search. See styled inline guidance and one rounded focus ring, without a browser validation bubble.
- [ ] **F15 — Meal category:** Select breakfast, lunch, dinner, snack or other before review.
- [ ] **F16 — Optional meal name:** Expand Name or save this meal and add a name.
- [ ] **F17 — Draft navigation:** Leave an unconfirmed selection, visit another app page and return. It remains in the same signed-in session. Reloading clears this in-memory draft.
- [ ] **F18 — Pagination:** Load another result page. Repeated FDC IDs are removed from the displayed list; distinct records remain available.
- [ ] **F19 — My foods:** Add a private custom food from the My foods tab.
- [ ] **F20 — Saved meal:** Select a saved meal, review its components and confirm. Replacing a current selection asks through the app's styled dialog.

## Favourites, reuse and portions

- [ ] **S1 — Favourite:** Open a selected food's More options and save it as a favourite.
- [ ] **S2 — Find favourites:** Expand Recent, favourites & saved portions, choose Favourites, add a food or Unsave it.
- [ ] **S3 — Recent foods:** Reuse foods from confirmed recent logs.
- [ ] **S4 — Frequent foods:** View foods ordered by confirmed use.
- [ ] **S5 — Household portion:** Set a measured amount, choose Save household portion and name it, for example “1 bowl”. It stores your measurement; the app does not guess weights.
- [ ] **S6 — Portion multiplier:** For a matching food/state/unit, choose a saved portion and use a fractional count. Check its resulting total weight and nutrition.
- [ ] **S7 — Portion management:** Show saved portions, rename a label or delete the definition using styled dialogs. Existing confirmed logs retain their snapshots.
- [ ] **S8 — Reusable meal:** Name selected foods and save them as a reusable meal. This saves a definition without adding a food log.
- [ ] **S9 — Nutrition edits:** Expand Edit nutrition. Supply a one-off value, review it, then change the amount and confirm that the earlier absolute edits are cleared.

## History and corrections

- [ ] **H1 — Tracking day:** Use the calendar to browse food history by your account's tracking day.
- [ ] **H2 — Totals:** Check daily energy, meal count, food count and timezone.
- [ ] **H3 — Details:** Show/hide each meal's food names, quantities and nutrition.
- [ ] **H4 — Correct amount:** Open a styled form containing amount and correction reason together; no browser prompt should appear.
- [ ] **H5 — Correction review:** Review the revised quantities and full nutrition summary in the app dialog before confirming.
- [ ] **H6 — Corrected totals:** Confirm and check that the visible meal and daily total are updated. Earlier revisions are retained for audit.
- [ ] **H7 — Remove meal:** Open and cancel the styled removal confirmation. Use a disposable test meal if actually removing it; visible totals should update.
- [ ] **H8 — More history:** Load additional meals for a busy day.
- [ ] **H9 — Older data:** Older detailed records can be compacted after 90 days; complete daily summaries remain available where supported. The UI identifies compacted detail.

## Private foods and saved meals

- [ ] **L1 — Create food:** Foods & meals → New food. Add a name, food state, base amount/unit and known nutrition values.
- [ ] **L2 — Unknown versus zero:** Leave unavailable nutrients unknown; explicitly enter zero only where known.
- [ ] **L3 — Edit food:** Change its current definition. Existing confirmed food logs retain their historical nutrition.
- [ ] **L4 — Delete food:** Review and cancel a styled deletion dialog, or remove a disposable definition.
- [ ] **L5 — Saved meals:** Browse definitions, rename one with the app form, or review a deletion. Recipes have their own page.
- [ ] **L6 — Privacy:** Another signed-in account cannot browse your private foods, saved meals, portions or favourites. Imported public USDA records are shared.

## Recipes

- [ ] **R1 — Create recipe:** Open New recipe. The styled form contains name, batch yield, preparation notes and ingredient search.
- [ ] **R2 — Ingredient sources:** Choose Local database, My custom foods or Search USDA. Database size and your search allowance use the same backend counters as Log food.
- [ ] **R3 — Ingredients:** Search, add/remove ingredients and set each quantity and food state.
- [ ] **R4 — Batch review:** Calculate & review whole-batch and per-serving nutrition before saving the definition.
- [ ] **R5 — Edit definition:** Change yield or ingredients, recalculate and save. Existing confirmed portions are unchanged.
- [ ] **R6 — Log a portion:** Choose Log a portion and enter whole or fractional servings, for example 0.5 or 1.5.
- [ ] **R7 — Portion edits:** Optionally adjust ingredient nutrition for the portion, then review again.
- [ ] **R8 — Confirm portion:** Log the reviewed portion and check the ingredients/quantities in History.
- [ ] **R9 — Delete recipe:** Review a styled confirmation. Use a disposable recipe for an actual deletion.

## Nutrition targets and analysis

- [ ] **N1 — Profile prerequisite:** With no date of birth, Nutrition targets links to profile setup and returns to targets after saving.
- [ ] **N2 — Benchmark proposal:** Review attributed FDA food-label benchmarks. They are general comparisons, not calculated personal calorie prescriptions.
- [ ] **N3 — Accept/reject:** Adjust allowed values with a reason, then accept, or reject the proposal. Proposals never activate automatically.
- [ ] **N4 — Target history:** Review accepted/rejected versions and effective tracking days.
- [ ] **N5 — Analysis range:** Choose a tracking-day range and a nutrient to inspect.
- [ ] **N6 — Trends:** Inspect daily values, averages and gaps for unknown/missing data.
- [ ] **N7 — Target comparisons:** Review the accepted target effective for the historical day, rather than a future target or unaccepted proposal.
- [ ] **N8 — Contributions:** Open a day's food contributors and inspect their nutrition breakdown.
- [ ] **N9 — Nutrient detail:** Review extended nutrient groups and daily tables. Phone tables should show labelled values without horizontal scrolling.

## Water, weight and activity

- [ ] **V1 — Water:** Log water with 250/500/750 ml quick amounts or a custom amount in ml/L.
- [ ] **V2 — Water goal:** Set, change or clear an optional personal daily water goal.
- [ ] **V3 — Weight:** Record weight in kg or lb and check the stored/displayed conversion.
- [ ] **V4 — Activity:** Record name, duration, intensity and optional estimated expenditure.
- [ ] **V5 — Time and notes:** Expand optional recorded time and notes. Entry time is explicitly labelled with the device timezone; grouping uses the account tracking day.
- [ ] **V6 — Edit record:** Open history, change a record and save it.
- [ ] **V7 — Remove record:** Review/cancel the app's confirmation, or remove a disposable record.
- [ ] **V8 — Trends:** Select a history range and inspect water totals, weight measurements and activity trends.
- [ ] **V9 — Dashboard:** Check that today's wellness summary updates after recording data.

## Medication, supplements and tasks

- [ ] **M1 — Item:** Add a medication, supplement or generic task with a name, optional instructions/notes and start date. Tasks hide dosage fields; medication/supplement amounts and units remain optional as before.
- [ ] **M2 — Daily schedule:** Set one or several times with time controls and Add another time.
- [ ] **M3 — Weekly schedule:** Select weekdays and optionally a repeat interval.
- [ ] **M4 — Monthly schedule:** Select month dates using the date grid. Absent dates are skipped for shorter months.
- [ ] **M5 — Advanced schedule:** Set interval/end date and check the displayed schedule timezone.
- [ ] **M6 — Occurrences:** Browse the seven-day UTC window and inspect each occurrence's saved local date/time/timezone.
- [ ] **M7 — Taken/done/skipped:** Mark medication/supplements taken, tasks done, or an occurrence skipped, and inspect the resulting status/history. Task completion uses the existing `TAKEN` wire status.
- [ ] **M8 — Reschedule:** Choose a new date/time in the styled picker. It is labelled with the device timezone; there is no raw ISO timestamp prompt.
- [ ] **M9 — Rename:** Rename an item using the app form.
- [ ] **M10 — Pause versus disable:** Pause/change a schedule separately from disabling an item. Confirm the displayed state makes the distinction clear.
- [ ] **M11 — Versions:** Review schedule changes without rewriting previously recorded occurrences.
- [ ] **M12 — Adherence:** Select up to 31 UTC dates, review status counts/history and the taken/due percentage. Future events do not inflate the due denominator.

## Dashboard, settings and account

- [ ] **A1 — Daily overview:** Inspect today's energy, routine nutrients, logged meals, health routine and wellness summary.
- [ ] **A2 — Profile:** Update display name, date of birth and timezone from Settings. Open Optional profile details to edit height, weight and tracking goal.
- [ ] **A3 — Day boundary:** Change daily reset time; future records use the new rule while saved historical records keep theirs.
- [ ] **A4 — Units:** Save metric/imperial preference. Individual entry controls continue to state their supported units explicitly.
- [ ] **A5 — Profile details:** Save height/weight with their stated units and edit the tracking goal. Check that values persist when Settings is reopened.
- [ ] **A6 — Appearance:** Switch between light, dark and system appearance; check the current setting in Settings.
- [ ] **A7 — Sign-in methods:** Where Google sign-in is enabled, inspect/link the same-email Google identity.
- [ ] **A8 — Logout:** Sign out and verify that account pages and the previous unconfirmed draft are no longer accessible.
- [ ] **A9 — Registration/verification:** Create a disposable local account, use local Mailpit for verification, and sign in. Also inspect resend-verification, email sign-in and password-reset screens.
- [ ] **A10 — Deletion workflow:** On a separate disposable account only, inspect deletion verification, the 30-day pending period and cancellation with fresh verification. Permanent purge is not a routine UI review check.

## UI consistency across screens

- [ ] **U1 — Phone:** Review 360px and 390px widths; tabs, inputs, dialogs and primary actions fit the viewport.
- [ ] **U2 — Tablet:** Review 768px and 1024px widths; open the full navigation drawer and use the visible food Review bar.
- [ ] **U3 — Laptop:** Review 1366×768 and 1440×900; selected foods and Review remain easy to reach.
- [ ] **U4 — Hover:** Buttons stay in place. Hovering Add should affect the button without shading the whole food row.
- [ ] **U5 — Keyboard focus:** Search has a single rounded focus treatment. Tab through a dialog, use Escape and confirm focus returns to its opener.
- [ ] **U6 — Validation:** Empty/invalid forms use inline, readable feedback; native orange validation balloons should not appear.
- [ ] **U7 — Confirmations:** Renaming, portion labels, removal and quantity corrections use the app's themed dialogs.
- [ ] **U8 — Themes:** Repeat the key search/correction/guide checks in light and dark appearance.
- [ ] **U9 — Async states:** Inspect loading, empty results and provider failure/allowance exhaustion. A failed request must not look like an empty successful result.

## Boundaries of the current product

AI/voice entry, wearable/device integrations, notifications, offline/PWA support and further identity providers remain roadmap work. Barcode scanning was removed from scope. This pass implements local catalogue discovery/search and per-account daily allowances; the larger Phase 3 operational work (provider-wide cooldown/budget, storage-budget enforcement and scheduled freshness/maintenance) is not represented as completed.
