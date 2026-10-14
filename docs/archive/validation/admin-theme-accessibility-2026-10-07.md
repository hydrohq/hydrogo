# Admin Studio theme accessibility matrix

**Date:** 2026-10-07  
**Result:** 112/112 Admin route × theme/mode scan passed  
**Scope:** Admin dashboard, page/news creation, media, users, permissions, security, audit, backup, control, operations, developer, and report surfaces; light and dark modes of HydroGo, shadcn, Nova, and Aurora themes.

The real Edition Chromium test in `e2e/custom/admin-management.spec.ts` reopens the route under each combination, verifies the selected theme/mode, and scans WCAG 2.1 A/AA rules with axe-core. Verification command:

```sh
cd e2e
npx playwright test custom/admin-management.spec.ts --project=chromium --workers=1 \
  --grep 'editor and viewer sessions|admin login opens the Edition management areas|admin routes meet WCAG' \
  --reporter=line
```

Initial runs revealed contrast issues not apparent in the default HydroGo theme alone. Green and secondary text shades in Shadcn/Aurora, amber shades in Shadcn/Nova/Aurora, secondary text shade in Nova, and content badge color were corrected. In the final run, 3 tests passed; 0 axe violations remain across the 14 routes × 8 theme/mode combinations.

This Chromium/axe evidence is not physical device or WebKit visual parity testing. The generated list/form 8/8 matrix is a separate scope; public and all generated page types are not included in this matrix.
