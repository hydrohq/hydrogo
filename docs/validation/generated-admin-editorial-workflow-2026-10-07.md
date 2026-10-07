# Generated Admin editorial workflow validation

**Date:** 2026-10-07  
**Scope:** Browser-level review-copy editing and publish transition for a
Blueprint-generated Admin resource.

## Result

The Chromium test now exercises a generated collection with versioned
published content, an isolated draft, a review state, and an Edition-provided
publish callback:

1. Open the generated draft inbox and select the review copy.
2. Edit the draft title and save it without changing the published API record.
3. Publish the updated draft through the generated workflow control.
4. Confirm the published API record and generated collection list both show
   the new title and published state.

A second browser path opens a separate unpublished draft, discards it from the
editor, confirms it disappears from the draft inbox, and verifies the
published API returns 404 for its ID. A localized path edits and publishes the
Turkish translation, confirms it appears in the Turkish generated list, and
checks that the default English API record remains unchanged.

The shared Studio shell appears on the generated screens. axe-core WCAG 2.1
A/AA checks pass on the draft inbox and draft editor.
The desktop draft editor capture was visually reviewed. That review found and
fixed a capitalization-sensitive `Content` group mismatch: default content
groups now match the active Turkish or English Studio locale regardless of
Blueprint capitalization.

## Test command

```sh
cd e2e && npx playwright test custom/generated-richtext.spec.ts --project=chromium --workers=1 --reporter=line
```

Result: **6/6 Chromium tests passed**, including formatted rich-text create/edit,
stale edit conflict recovery, relation/media selection, draft-review-publish,
unpublished draft discard, and localized draft save/publish with default-locale
isolation. axe-core checks pass on the draft inbox, draft editor, and localized
editor.

The editor/publish journey uses an in-memory Edition API and callback fixture;
it validates the generated Admin HTTP/UI contract and state returned by those
callbacks. SQLite/PostgreSQL draft storage is covered by separate contract
tests, not by this browser fixture. Scheduling and live tenant isolation remain
separate acceptance cases. The browser path found and fixed locale propagation
on workflow submissions and query-safe locale navigation in generated lists.
