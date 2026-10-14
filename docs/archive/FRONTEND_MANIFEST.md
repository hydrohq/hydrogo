# Versioned frontend manifest

`ui.FrontendManifest` is the reviewable contract for browser-facing surfaces.
It keeps route methods/paths/access and stable `data-*` selectors explicit so
server-rendered HTML, HTMX enhancements, HydroGo theme mode, and a future
separate frontend can evolve without guessing DOM details.

Each manifest has a positive `version`. Route names and selector names are
unique; routes use absolute paths, a supported state-changing/read method, and
an explicit `public` or `protected` access class. Selectors must document their
purpose. `Validate` runs before publishing a manifest; it does not grant route
authorization and cannot replace the router’s access classification.
